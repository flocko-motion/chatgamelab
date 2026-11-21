package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameActionsTestSuite tests what the game list says each caller may do with
// a game: editing each section (the settings cover deleting it) and copying.
type GameActionsTestSuite struct {
	testutil.BaseSuite
}

func TestGameActionsSuite(t *testing.T) {
	s := &GameActionsTestSuite{}
	s.SuiteName = "Game Actions Tests"
	suite.Run(t, s)
}

// The creator may edit and copy their own game.
func (s *GameActionsTestSuite) TestTheCreatorMayEditAndCopy() {
	alice := s.CreateUser("actions-owner")
	game := Must(alice.UploadGame("alien-first-contact"))

	listed := Must(alice.GameInList(game.Id.String()))
	s.True(listed.MayEditCard)
	s.True(listed.MayEditContent)
	s.True(listed.MayEditSettings)
	s.True(listed.MayCopy)
}

// Another account may copy a public game, not edit it.
func (s *GameActionsTestSuite) TestOthersMayCopyAPublicGameButNotEditIt() {
	alice := s.CreateUser("actions-public-owner")
	bob := s.CreateUser("actions-public-other")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	listed := Must(bob.GameInList(game.Id.String()))
	s.False(listed.MayEditCard)
	s.False(listed.MayEditContent)
	s.False(listed.MayEditSettings)
	s.True(listed.MayCopy)
}

// An admin may edit anyone's game, but copy only public or own ones.
func (s *GameActionsTestSuite) TestAnAdminMayEditAPrivateGameButNotCopyIt() {
	alice := s.CreateUser("actions-admin-owner")
	game := Must(alice.UploadGame("alien-first-contact"))

	listed := Must(s.DevUser().GameInList(game.Id.String()))
	s.True(listed.MayEditCard)
	s.True(listed.MayEditContent)
	s.True(listed.MayEditSettings)
	s.False(listed.MayCopy)
}

// A head may edit a participant's game of their organisation's workshop, not
// copy it.
func (s *GameActionsTestSuite) TestAHeadMayEditAParticipantsGame() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Actions Org"))
	head := s.CreateUser("actions-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Actions Workshop"))
	workshopID := workshop.Id.String()
	participant := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshopID)).InviteToken))
	game := Must(participant.UploadGame("alien-first-contact"))

	Must(head.SetActiveWorkshop(&workshopID))
	listed := Must(head.GameInList(game.Id.String()))
	s.True(listed.MayEditCard)
	s.True(listed.MayEditContent)
	s.True(listed.MayEditSettings)
	s.False(listed.MayCopy)

	// The participant, in their workshop, may edit their own game too.
	own := Must(participant.GameInList(game.Id.String()))
	s.True(own.MayEditCard)
	s.True(own.MayEditContent)
	s.True(own.MayEditSettings)
}
