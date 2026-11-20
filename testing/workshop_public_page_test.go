package testing

import (
	"regexp"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// WorkshopPublicPageTestSuite tests the public workshop page /w/<slug>: its
// generated link, editing it, what visitors see, downloads, and the share links
// visitors play through.
type WorkshopPublicPageTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopPublicPageSuite(t *testing.T) {
	s := &WorkshopPublicPageTestSuite{}
	s.SuiteName = "Workshop Public Page Tests"
	suite.Run(t, s)
}

type publicPageFixture struct {
	head        *testutil.UserClient
	participant *testutil.UserClient
	wsID        string
	wsName      string
	slug        string
}

// setup creates an institution with a head, a workshop with a working key, and
// one participant. withKey false leaves the workshop without a key.
func (s *WorkshopPublicPageTestSuite) setup(prefix string, withKey bool) publicPageFixture {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	name := prefix + " Workshop"
	workshop := Must(head.CreateWorkshop(inst.Id.String(), name))
	wsID := workshop.Id.String()
	if withKey {
		keyShare := Must(head.AddApiKey("mock-"+prefix, prefix+" Key", "openai"))
		orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))
		orgShareID := orgShare.Id.String()
		Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	}

	invite := Must(head.CreateWorkshopInvite(wsID))
	participant := Must(s.JoinWorkshop(*invite.InviteToken))

	ws := Must(head.GetWorkshop(wsID))
	s.Require().NotNil(ws.PublicSlug, "a new workshop must have a public page link")
	return publicPageFixture{head: head, participant: participant, wsID: wsID, wsName: name, slug: *ws.PublicSlug}
}

// updateWorkshop saves the workshop as the head: active, public or not, and
// whatever edit sets; every other setting is off and the page's slug, text
// and links stay as they are.
func (s *WorkshopPublicPageTestSuite) updateWorkshop(f publicPageFixture, public bool, edit func(*testutil.WorkshopUpdate)) (testutil.Workshop, error) {
	update := testutil.WorkshopUpdate{Name: f.wsName, Active: true, Public: public}
	if edit != nil {
		edit(&update)
	}
	return f.head.UpdateWorkshop(f.wsID, update)
}

// withSlug sets the page's link.
func (s *WorkshopPublicPageTestSuite) withSlug(slug string) func(*testutil.WorkshopUpdate) {
	return func(w *testutil.WorkshopUpdate) { w.PublicSlug = &slug }
}

// withDescription sets the page's text.
func (s *WorkshopPublicPageTestSuite) withDescription(text string) func(*testutil.WorkshopUpdate) {
	return func(w *testutil.WorkshopUpdate) { w.PublicDescription = &text }
}

func (s *WorkshopPublicPageTestSuite) setGamePublic(u *testutil.UserClient, gameID string, public bool) {
	Must(u.SetGamePublic(gameID, public))
}

func (s *WorkshopPublicPageTestSuite) page(slug string) (testutil.PublicWorkshopPage, error) {
	return s.Public().GetPublicWorkshopPage(slug)
}

func (s *WorkshopPublicPageTestSuite) TestSlugFromWorkshopName() {
	f := s.setup("Robotik-AG: Ärger & Spaß", false)
	s.Regexp(regexp.MustCompile(`^robotik-ag-aerger-spass-[a-z]{4,10}-[a-z]{4,10}$`), f.slug)
}

func (s *WorkshopPublicPageTestSuite) TestPageOnlyWhilePublic() {
	f := s.setup("pubpage-toggle", true)
	_, err := s.page(f.slug)
	s.NotFound(err, "a workshop that is not public has no page")

	Must(s.updateWorkshop(f, true, s.withDescription("  Wir bauen Spiele.\nZwei Tage lang.  ")))
	page := Must(s.page(f.slug))
	s.Equal(f.wsName, page.Name)
	s.Equal("Wir bauen Spiele.\nZwei Tage lang.", page.Description)
	s.Empty(page.Games)

	// Typed with capitals still finds the page.
	Must(s.page(strings.ToUpper(f.slug)))

	_, err = s.page("gibt-es-nicht-hier")
	s.NotFound(err)
}

func (s *WorkshopPublicPageTestSuite) TestListsOnlyPublicGamesWithoutCreators() {
	f := s.setup("pubpage-games", true)
	Must(s.updateWorkshop(f, true, nil))

	shown := Must(f.participant.UploadGame("alien-first-contact"))
	hidden := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, shown.Id.String(), true)

	// A public game from another workshop stays off this page.
	other := s.setup("pubpage-games-other", true)
	foreign := Must(other.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(other.participant, foreign.Id.String(), true)

	page := Must(s.page(f.slug))
	s.True(page.PlayAvailable)
	s.Require().Len(page.Games, 1)
	s.Equal(shown.Id, page.Games[0].Id)
	s.Equal(shown.Name, page.Games[0].Name)
	s.Require().NotNil(page.Games[0].Play)
	s.Equal(50, page.Games[0].Play.Remaining)
	s.Equal(50, page.Games[0].Play.Limit)

	body := Must(s.Public().GetPublicWorkshopPageJSON(f.slug))
	s.NotContains(body, "creator", "the page must never name who made a game")
	s.NotContains(body, "institution")

	// Download: listed games only.
	yaml := Must(s.Public().GetPublicWorkshopGameYAML(f.slug, shown.Id.String()))
	s.Contains(yaml, shown.Name)
	_, err := s.Public().GetPublicWorkshopGameYAML(f.slug, hidden.Id.String())
	s.NotFound(err, "a game not on the page can't be downloaded from it")
	_, err = s.Public().GetPublicWorkshopGameYAML(f.slug, foreign.Id.String())
	s.NotFound(err, "another workshop's game can't be downloaded from this page")
}

func (s *WorkshopPublicPageTestSuite) TestNoPlayWithoutWorkshopKey() {
	f := s.setup("pubpage-nokey", false)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.Id.String(), true)

	page := Must(s.page(f.slug))
	s.False(page.PlayAvailable)
	s.Require().Len(page.Games, 1)
	s.Nil(page.Games[0].Play)
}

func (s *WorkshopPublicPageTestSuite) TestPlayStaysApartFromHandMadeLinks() {
	f := s.setup("pubpage-play", true)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.Id.String(), true)

	play := Must(s.page(f.slug)).Games[0].Play
	s.Require().NotNil(play)
	s.Equal(50, play.Remaining)

	again := Must(s.page(f.slug)).Games[0].Play
	s.Equal(play.Token, again.Token, "the page keeps one link per game")

	// A leader's workshop share is a separate, unlimited link.
	handMade := Must(f.head.CreateWorkshopGameShare(game.Id.String(), f.wsID, nil))
	s.NotEqual(play.Token, handMade.Token)
	s.Nil(handMade.Remaining)
}

func (s *WorkshopPublicPageTestSuite) TestLinksRevoked() {
	f := s.setup("pubpage-revoke", true)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.Id.String(), true)

	// Unpublishing the game revokes its link.
	token := Must(s.page(f.slug)).Games[0].Play.Token
	s.setGamePublic(f.head, game.Id.String(), false)
	_, err := s.Public().GuestGetGameInfo(token)
	s.NotFound(err, "an unpublished game's page link must stop working")

	// Switching the page off revokes every link; switching it on makes fresh ones.
	s.setGamePublic(f.head, game.Id.String(), true)
	token = Must(s.page(f.slug)).Games[0].Play.Token
	Must(s.updateWorkshop(f, false, nil))
	_, err = s.Public().GuestGetGameInfo(token)
	s.NotFound(err, "switching the page off must revoke its links")

	Must(s.updateWorkshop(f, true, nil))
	fresh := Must(s.page(f.slug)).Games[0].Play
	s.NotEqual(token, fresh.Token)
	s.Equal(50, fresh.Remaining)

	// Deleting the workshop revokes them as well.
	MustSucceed(f.head.DeleteWorkshop(f.wsID))
	_, err = s.Public().GuestGetGameInfo(fresh.Token)
	s.NotFound(err, "deleting the workshop must revoke its page links")
	_, err = s.page(f.slug)
	s.NotFound(err)
}

func (s *WorkshopPublicPageTestSuite) TestEditSlug() {
	f := s.setup("pubpage-slug", true)
	Must(s.updateWorkshop(f, true, nil))

	ws := Must(s.updateWorkshop(f, true, s.withSlug("  Unser-Spiele-Workshop ")))
	s.Equal("unser-spiele-workshop", *ws.PublicSlug)
	Must(s.page("unser-spiele-workshop"))
	_, err := s.page(f.slug)
	s.NotFound(err, "the old link must stop working")

	// Other settings leave the link alone when publicSlug is omitted.
	ws = Must(s.updateWorkshop(f, true, func(w *testutil.WorkshopUpdate) { w.IsPaused = true }))
	s.Equal("unser-spiele-workshop", *ws.PublicSlug)

	for _, bad := range []string{"ab", "mit leerzeichen", "-vorne", "hinten-", "doppel--strich", "ümlaut", strings.Repeat("a", 61)} {
		_, err := s.updateWorkshop(f, true, s.withSlug(bad))
		s.Refused(err, testutil.CodePublicSlugInvalid, bad)
	}

	other := s.setup("pubpage-slug-other", true)
	_, err = s.updateWorkshop(other, false, s.withSlug("unser-spiele-workshop"))
	s.Refused(err, testutil.CodePublicSlugTaken)

	_, err = s.updateWorkshop(f, true, s.withDescription(strings.Repeat("ä", 2001)))
	s.Refused(err, testutil.CodePublicDescriptionTooLong)
	Must(s.updateWorkshop(f, true, s.withDescription(strings.Repeat("ä", 2000))))
}

func (s *WorkshopPublicPageTestSuite) TestParticipantCannotChangePage() {
	f := s.setup("pubpage-perm", true)
	_, err := f.participant.UpdateWorkshop(f.wsID, testutil.WorkshopUpdate{
		Name: f.wsName, Active: true, Public: true,
	})
	s.Forbidden(err)
}
