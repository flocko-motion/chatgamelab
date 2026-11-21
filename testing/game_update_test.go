package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameUpdateTestSuite tests changing a game through the secure core: only
// what is sent and differs changes, each changed section must be editable,
// and making a game public is a right of its own.
type GameUpdateTestSuite struct {
	testutil.BaseSuite
}

func TestGameUpdateSuite(t *testing.T) {
	s := &GameUpdateTestSuite{}
	s.SuiteName = "Game Update Tests"
	suite.Run(t, s)
}

// A request that sends one field changes that field and nothing else.
func (s *GameUpdateTestSuite) TestOnlyWhatIsSentChanges() {
	alice := s.CreateUser("update-partial")
	game := Must(alice.UploadGame("alien-first-contact"))

	updated := Must(alice.ChangeGame(game.Id.String(), testutil.GameUpdate{Name: some("Renamed")}))
	s.Equal("Renamed", updated.Name)
	s.Equal(game.Description, updated.Description, "the card's other fields stay")
	s.Require().NotNil(updated.SystemMessageScenario)
	s.Equal(game.SystemMessageScenario, *updated.SystemMessageScenario, "the content stays")
}

// Whoever may read a game but not edit it is refused, and nothing changes.
func (s *GameUpdateTestSuite) TestReadableIsNotEditable() {
	alice := s.CreateUser("update-readonly")
	bob := s.CreateUser("update-readonly-bob")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	view := Must(bob.ReadGame(game.Id.String()))
	s.False(view.MayEditCard)

	_, err := bob.ChangeGame(game.Id.String(), testutil.GameUpdate{Name: some("Hacked")})
	s.Forbidden(err)
	s.Equal(game.Name, Must(alice.ReadGame(game.Id.String())).Name)
}

// Sending a game back as it is changes nothing, so it needs no right.
func (s *GameUpdateTestSuite) TestSendingItBackUnchangedIsNoChange() {
	alice := s.CreateUser("update-unchanged")
	bob := s.CreateUser("update-unchanged-bob")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))
	before := Must(alice.ReadGame(game.Id.String()))

	Must(bob.ChangeGame(game.Id.String(), testutil.GameUpdate{Name: some(game.Name)}))
	s.Equal(before.Meta.ModifiedAt, Must(alice.ReadGame(game.Id.String())).Meta.ModifiedAt, "nothing was written")
}

// A head may make a game of their organisation's workshop public.
func (s *GameUpdateTestSuite) TestHeadPublishesAWorkshopGame() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Update Publish Org"))
	head := s.CreateUser("update-publish-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Update Publish Workshop"))
	participant := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
	game := Must(participant.UploadGame("alien-first-contact"))

	s.True(Must(head.ReadGame(game.Id.String())).MayPublish)
	updated := Must(head.ChangeGame(game.Id.String(), testutil.GameUpdate{Public: some(true)}))
	s.True(updated.Public)
}

// An admin publishes any game, and takes it offline again: unpublishing is
// editing the card.
func (s *GameUpdateTestSuite) TestAnAdminPublishesAndUnpublishesAnyGame() {
	admin := s.DevUser()
	alice := s.CreateUser("update-admin-publish")
	game := Must(alice.UploadGame("alien-first-contact"))

	s.True(Must(admin.ReadGame(game.Id.String())).MayPublish)
	s.True(Must(admin.ChangeGame(game.Id.String(), testutil.GameUpdate{Public: some(true)})).Public)
	s.False(Must(admin.ChangeGame(game.Id.String(), testutil.GameUpdate{Public: some(false)})).Public)
}

// A genre the engine does not know is refused.
func (s *GameUpdateTestSuite) TestUnknownGenreIsRefused() {
	alice := s.CreateUser("update-genre")
	game := Must(alice.UploadGame("alien-first-contact"))

	_, err := alice.ChangeGame(game.Id.String(), testutil.GameUpdate{Genre: some("opera")})
	s.Refused(err, testutil.CodeValidation)
}

func some[T any](v T) *T { return &v }
