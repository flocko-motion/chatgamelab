package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// OrganisationGamesTestSuite checks that an organisation's head and staff
// work together on its games (#564): the games written in its workshops and
// the games its head and staff write anywhere. They do with them what the
// designer does, except deleting a colleague's game outside a workshop.
// Whether a game is the organisation's is read each time: a designer who
// leaves takes their games along.
type OrganisationGamesTestSuite struct {
	testutil.BaseSuite
}

func TestOrganisationGamesSuite(t *testing.T) {
	s := &OrganisationGamesTestSuite{}
	s.SuiteName = "Organisation games"
	suite.Run(t, s)
}

// colleagues is an organisation with a head and two staff members; the
// designer wrote a game outside any workshop, and the head gave a key to the
// organisation.
type colleagues struct {
	org, game, share         string
	head, designer, coworker *testutil.UserClient
}

func (s *OrganisationGamesTestSuite) colleagues() colleagues {
	org := Must(s.DevUser().CreateInstitution(testutil.Unique("Kollegium"))).Id.String()
	join := func(name, role string) *testutil.UserClient {
		u := s.CreateUser(testutil.Unique(name))
		invite := Must(s.DevUser().InviteToInstitution(org, role, u.ID))
		Must(u.AcceptInviteByToken(invite.Id.String()))
		return u
	}
	head := join("head", "head")
	designer := join("designer", "staff")
	coworker := join("coworker", "staff")
	key := Must(head.AddApiKey(testutil.Unique("mock-org-games"), "Org key", "mock"))
	share := Must(head.ShareApiKeyWithInstitution(key.Id.String(), org)).Id.String()
	game := Must(designer.UploadGame("alien-first-contact")).Id.String()
	return colleagues{org: org, game: game, share: share, head: head, designer: designer, coworker: coworker}
}

// Head and staff list, read, edit, share, publish and copy a game a
// colleague wrote outside any workshop, and cannot delete it.
func (s *OrganisationGamesTestSuite) TestColleaguesWorkOnAGameWrittenOutsideAWorkshop() {
	c := s.colleagues()
	for name, colleague := range map[string]*testutil.UserClient{"head": c.head, "staff": c.coworker} {
		s.Run(name, func() {
			listed := Must(colleague.GameInList(c.game))
			s.True(listed.MayEditCard, "edits the card")
			s.True(listed.MayEditContent, "edits the content")
			s.True(listed.MayEditSettings, "edits the settings")
			s.True(listed.MayShare, "shares it")
			s.True(listed.MayCopy, "copies it")
			s.True(listed.MayPublish, "publishes it")
			s.False(listed.MayDelete, "a colleague's game outside a workshop is deleted only by its designer")

			read := Must(colleague.ReadGame(c.game))
			s.Require().NotNil(read.OrganisationId, "the game is the organisation's, through its designer")
			s.Equal(c.org, read.OrganisationId.String())

			Must(colleague.ChangeGame(c.game, testutil.GameUpdate{Name: some(testutil.Unique("Edited by " + name))}))
			Must(colleague.CreateGameShare(c.game, c.share, nil))
			Must(colleague.CloneGame(c.game))
			s.Forbidden(colleague.DeleteGame(c.game))
		})
	}

	published := Must(c.coworker.ChangeGame(c.game, testutil.GameUpdate{Public: some(true)}))
	s.True(published.Public, "staff publish a colleague's game")
	Must(c.designer.ReadGame(c.game))
}

// The organisation filter lists the organisation's games for its head and
// staff, a colleague's game outside a workshop included, and nobody else's.
func (s *OrganisationGamesTestSuite) TestTheOrganisationFilterListsTheOrganisationsGames() {
	c := s.colleagues()
	outsider := s.CreateUser(testutil.Unique("outsider"))
	theirs := Must(outsider.UploadGame("alien-first-contact")).Id.String()
	Must(outsider.ChangeGame(theirs, testutil.GameUpdate{Public: some(true)}))

	for name, colleague := range map[string]*testutil.UserClient{"head": c.head, "staff": c.coworker} {
		s.Run(name, func() {
			ids := gameIDs(Must(colleague.ListGamesWithFilter("organisation")))
			s.Contains(ids, c.game, "a colleague's game outside a workshop")
			s.NotContains(ids, theirs, "a public game of someone outside the organisation")
			s.Contains(gameIDs(Must(colleague.ListGames())), c.game, "the unfiltered list shows it too")
			s.NotContains(gameIDs(Must(colleague.ListGamesWithFilter("own"))), c.game, "only mine is what I wrote")
		})
	}

	page := Must(outsider.ListGamesPage(0, 0, "organisation"))
	s.Zero(page.Total, "an individual is in no organisation: the filter lists nothing")
}

// A designer who leaves takes their games along: their former colleagues
// lose access, the designer keeps it.
func (s *OrganisationGamesTestSuite) TestADesignerWhoLeavesTakesTheirGamesAlong() {
	c := s.colleagues()
	Must(c.coworker.ReadGame(c.game))

	MustSucceed(c.designer.RemoveMember(c.org, c.designer.ID))

	s.NotFound(errOf(c.coworker.ReadGame(c.game)), "staff lose a former colleague's game")
	s.NotFound(errOf(c.head.ReadGame(c.game)), "the head too")
	s.NotContains(gameIDs(Must(c.head.ListGames())), c.game)
	s.NotContains(gameIDs(Must(c.head.ListGamesWithFilter("organisation"))), c.game)
	mine := Must(c.designer.ReadGame(c.game))
	s.True(mine.MayEditCard, "the designer keeps their game")
	s.Nil(mine.OrganisationId, "an individual's game belongs to no organisation")
}

// A game written in one of the organisation's workshops stays the
// organisation's to delete, as moderation of what is made there.
func (s *OrganisationGamesTestSuite) TestStaffDeleteGamesOfTheOrganisationsWorkshops() {
	c := s.colleagues()
	workshop := Must(c.designer.CreateWorkshop(c.org, testutil.Unique("Werkstatt"))).Id.String()
	Must(c.designer.SetActiveWorkshop(&workshop))
	game := Must(c.designer.UploadGame("alien-first-contact")).Id.String()
	Must(c.designer.SetActiveWorkshop(nil))

	s.True(Must(c.coworker.ReadGame(game)).MayDelete)
	MustSucceed(c.coworker.DeleteGame(game))
}

func gameIDs(games []testutil.Game) []string {
	ids := make([]string, 0, len(games))
	for _, g := range games {
		ids = append(ids, g.Id.String())
	}
	return ids
}

func errOf[T any](_ T, err error) error { return err }
