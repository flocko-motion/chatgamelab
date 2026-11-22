package testing

import (
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameCopyTestSuite tests copying a game and importing a game file through
// the secure core.
type GameCopyTestSuite struct {
	testutil.BaseSuite
}

func TestGameCopySuite(t *testing.T) {
	s := &GameCopyTestSuite{}
	s.SuiteName = "Game Copy Tests"
	suite.Run(t, s)
}

// A copy is private, the caller's, and names whoever first created the game,
// across copies of copies.
func (s *GameCopyTestSuite) TestACopyNamesTheOriginalCreator() {
	alice := s.CreateUser("copy-origin-a")
	bob := s.CreateUser("copy-origin-b")
	carol := s.CreateUser("copy-origin-c")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	first := Must(bob.CloneGame(game.Id.String()))
	Must(bob.SetGamePublic(first.Id.String(), true))
	second := Must(carol.ReadGame(Must(carol.CloneGame(first.Id.String())).Id.String()))

	s.False(second.Public)
	s.True(second.MayEditCard, "the copy is carol's")
	s.Require().NotNil(second.OriginallyCreatedBy)
	s.Equal(game.Meta.CreatedBy, second.OriginallyCreatedBy, "alice, not bob")
	s.Equal(1, Must(alice.ReadGame(game.Id.String())).CloneCount)
}

// Reading a game is not copying it: a guest reads their workshop's
// games, but copies only public ones and their own.
func (s *GameCopyTestSuite) TestReadingIsNotCopying() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Copy Org"))
	head := s.CreateUser("copy-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Copy Workshop"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))
	first := Must(s.JoinWorkshop(*invite.InviteToken))
	second := Must(s.JoinWorkshop(*invite.InviteToken))
	game := Must(first.UploadGame("alien-first-contact"))

	s.False(Must(second.ReadGame(game.Id.String())).MayCopy)
	_, err := second.CloneGame(game.Id.String())
	s.Forbidden(err)
}

// A copy's name stays within the limit.
func (s *GameCopyTestSuite) TestACopysNameStaysWithinTheLimit() {
	alice := s.CreateUser("copy-long")
	game := Must(alice.CreateGame(strings.Repeat("x", 70)))

	copied := Must(alice.CloneGame(game.Id.String()))
	s.LessOrEqual(len(copied.Name), 70)
	s.True(strings.HasSuffix(copied.Name, " (Copy)"))
}

// Importing a game file is an update: whoever may only read the game is
// refused.
func (s *GameCopyTestSuite) TestImportTakesTheRightToEdit() {
	alice := s.CreateUser("import-edit-a")
	bob := s.CreateUser("import-edit-b")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	yaml := Must(bob.GetGameYAML(game.Id.String()))
	s.Forbidden(bob.PutGameYAML(game.Id.String(), strings.Replace(yaml, game.Name, "Taken Over", 1)))
	s.Equal(game.Name, Must(alice.ReadGame(game.Id.String())).Name)
}

// Game names are unique: a copy takes the first free "(Copy)" name, so
// everyone copies a game however often it was copied; renaming to a taken
// name is refused as such.
func (s *GameCopyTestSuite) TestCopiesFindAFreeName() {
	author := s.CreateUser("copy-name-author")
	game := Must(author.UploadGame("alien-first-contact"))
	Must(author.SetGamePublic(game.Id.String(), true))

	first := Must(s.CreateUser("copy-name-a").CloneGame(game.Id.String()))
	second := Must(s.CreateUser("copy-name-b").CloneGame(game.Id.String()))
	s.Equal(game.Name+" (Copy)", first.Name)
	s.Equal(game.Name+" (Copy 2)", second.Name)

	_, err := author.RenameGame(game.Id.String(), first.Name)
	s.Refused(err, testutil.CodeDuplicateName, "a taken name")
}
