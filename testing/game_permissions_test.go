package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// GamePermissionsTestSuite tests game deletion cleanup and workshop game permissions.
type GamePermissionsTestSuite struct {
	testutil.BaseSuite
}

func TestGamePermissionsSuite(t *testing.T) {
	s := &GamePermissionsTestSuite{}
	s.SuiteName = "Game Permissions Tests"
	suite.Run(t, s)
}

// TestStaffCanDeleteWorkshopGame tests that a staff member of the org can delete
// a participant's workshop game.
func (s *GamePermissionsTestSuite) TestStaffCanDeleteWorkshopGame() {
	admin := s.DevUser()

	// Setup: institution + head + staff + workshop + participant
	inst := Must(admin.CreateInstitution("Game Perm Org"))
	head := s.CreateUser("gp-head")
	staff := s.CreateUser("gp-staff")

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Game Perm Workshop"))
	wsIDStr := workshop.ID.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant creates a game (auto-assigned to workshop)
	game := Must(participant.UploadGame("alien-first-contact"))
	s.NotEmpty(game.ID)
	s.T().Logf("Participant created game: %s (ID: %s)", game.Name, game.ID)

	// Staff deletes the participant's workshop game — should succeed
	MustSucceed(staff.DeleteGame(game.ID.String()))
	s.T().Logf("Staff deleted participant's workshop game")

	// Game should no longer be visible
	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.ID, g.ID, "game should be deleted")
	}
	s.T().Logf("Game correctly deleted by staff")
}

// TestHeadCanDeleteWorkshopGame tests that a head of the org can delete
// a participant's workshop game.
func (s *GamePermissionsTestSuite) TestHeadCanDeleteWorkshopGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Game Org"))
	head := s.CreateUser("gp-head2")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Head Game Workshop"))
	wsIDStr := workshop.ID.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)

	game := Must(participant.UploadGame("alien-first-contact"))
	s.T().Logf("Participant created game: %s", game.ID)

	// Head deletes the participant's workshop game — should succeed
	MustSucceed(head.DeleteGame(game.ID.String()))
	s.T().Logf("Head deleted participant's workshop game")

	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.ID, g.ID, "game should be deleted")
	}
}

// TestParticipantCannotDeleteOtherParticipantGame tests that a participant
// cannot delete another participant's game in the same workshop.
func (s *GamePermissionsTestSuite) TestParticipantCannotDeleteOtherParticipantGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Part Game Org"))
	head := s.CreateUser("gp-head3")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Part Game Workshop"))
	wsIDStr := workshop.ID.String()

	// Create two participants
	invite1 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	participant1 := s.CreateUserWithToken(*resp1.AuthToken)

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	participant2 := s.CreateUserWithToken(*resp2.AuthToken)

	// Participant1 creates a game
	game := Must(participant1.UploadGame("alien-first-contact"))
	s.T().Logf("Participant1 created game: %s", game.ID)

	// Participant2 tries to delete participant1's game — should fail
	err = participant2.DeleteGame(game.ID.String())
	s.Error(err, "participant should not be able to delete another participant's game")
	s.T().Logf("Correctly rejected: %v", err)

	// Game should still exist (check via owner)
	games := Must(participant1.ListGames())
	found := false
	for _, g := range games {
		if g.ID == game.ID {
			found = true
			break
		}
	}
	s.True(found, "game should still exist after failed deletion")
}

// TestStaffCannotDeleteNonWorkshopGame tests that a staff member cannot delete
// a game that doesn't belong to their org's workshop (e.g. an individual's personal game).
func (s *GamePermissionsTestSuite) TestStaffCannotDeleteNonWorkshopGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Non WS Game Org"))
	staff := s.CreateUser("gp-staff2")
	staffInvite := Must(admin.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Create an individual user (not in any workshop) with a game
	individual := s.CreateUser("gp-individual")
	game := Must(individual.UploadGame("alien-first-contact"))
	s.T().Logf("Individual created game: %s (not in any workshop)", game.ID)

	// Staff tries to delete the individual's game — should fail
	err := staff.DeleteGame(game.ID.String())
	s.Error(err, "staff should not be able to delete a non-workshop game")
	s.T().Logf("Correctly rejected: %v", err)
}

// TestGameDeletionCleansUpSessions tests that deleting a game also removes
// its sessions and messages.
func (s *GamePermissionsTestSuite) TestGameDeletionCleansUpSessions() {
	// Create user with API key, game, and a session
	user := s.CreateUser("gp-session-user")
	Must(user.AddApiKey("mock-gp-sess", "Session Key", "mock"))

	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User created game: %s", game.ID)

	// Create a session for the game
	session, err := user.CreateGameSession(game.ID.String())
	s.NoError(err)
	s.NotEmpty(session.ID)
	s.T().Logf("Created session: %s", session.ID)

	// Verify session exists
	loadedSession, err := user.GetGameSession(session.ID.String())
	s.NoError(err)
	s.Equal(session.ID, loadedSession.ID)
	s.T().Logf("Session verified")

	// Delete the game
	MustSucceed(user.DeleteGame(game.ID.String()))
	s.T().Logf("Game deleted")

	// Session should no longer exist
	_, err = user.GetGameSession(session.ID.String())
	s.Error(err, "session should be deleted along with game")
	s.T().Logf("Session correctly cleaned up")
}
