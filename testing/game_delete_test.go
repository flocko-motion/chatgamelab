package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameDeleteTestSuite tests deleting a game through the secure core: it takes
// the right to delete it (mayDelete), and a deleted game is gone.
type GameDeleteTestSuite struct {
	testutil.BaseSuite
}

func TestGameDeleteSuite(t *testing.T) {
	s := &GameDeleteTestSuite{}
	s.SuiteName = "Game Delete Tests"
	suite.Run(t, s)
}

// A deleted game no longer exists, not even for its creator.
func (s *GameDeleteTestSuite) TestADeletedGameIsGone() {
	alice := s.CreateUser("delete-gone")
	game := Must(alice.UploadGame("alien-first-contact"))

	s.Require().NoError(alice.DeleteGame(game.Id.String()))
	_, err := alice.ReadGame(game.Id.String())
	s.NotFound(err)
}

// A head may delete a game of their organisation's workshop; whoever may
// only read it may not.
func (s *GameDeleteTestSuite) TestAHeadDeletesAWorkshopGame() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Delete Org"))
	head := s.CreateUser("delete-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Delete Workshop"))
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
	game := Must(guest.UploadGame("alien-first-contact"))
	Must(guest.SetGamePublic(game.Id.String(), true))

	bob := s.CreateUser("delete-bob")
	s.False(Must(bob.ReadGame(game.Id.String())).MayDelete)
	s.Forbidden(bob.DeleteGame(game.Id.String()), "reading is not deleting")

	s.True(Must(head.ReadGame(game.Id.String())).MayDelete)
	s.Require().NoError(head.DeleteGame(game.Id.String()))
}
