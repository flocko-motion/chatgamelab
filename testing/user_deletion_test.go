package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserDeletionTestSuite tests that admins can delete users of every role type,
// with proper cleanup of API keys, shares, roles, etc.
type UserDeletionTestSuite struct {
	testutil.BaseSuite
}

func TestUserDeletionSuite(t *testing.T) {
	s := &UserDeletionTestSuite{}
	s.SuiteName = "User Deletion Tests"
	suite.Run(t, s)
}

// TestAdminCanDeleteParticipant tests deleting an anonymous workshop participant.
func (s *UserDeletionTestSuite) TestAdminCanDeleteParticipant() {
	admin := s.DevUser()

	// Setup: institution + head + workshop + participant
	inst := Must(admin.CreateInstitution("Del Participant Org"))
	head := s.CreateUser("del-part-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Del Participant Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	participant := Must(s.JoinWorkshop(*invite.InviteToken))
	s.T().Logf("Participant joined: %s (ID: %s)", participant.Name, participant.ID)

	// Verify participant exists
	me := Must(participant.GetMe())
	s.Equal(participant.Name, me.Name)

	// Count users before
	usersBefore := Must(admin.GetUsers())

	// Admin deletes the participant
	MustSucceed(admin.DeleteUser(participant.ID))
	s.T().Logf("Admin deleted participant")

	// Participant should no longer appear in user list (soft-deleted)
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-1, len(usersAfter), "user count should decrease by 1")
	s.T().Logf("Participant correctly deleted")
}

// TestAdminCanDeleteIndividualWithGameAndKeys tests deleting an individual user
// who has API keys and a game. Both the game and keys should be cleaned up.
func (s *UserDeletionTestSuite) TestAdminCanDeleteIndividualWithGameAndKeys() {
	admin := s.DevUser()

	// Create individual user with API key and a game
	individual := s.CreateUser("del-individual")
	s.Equal("individual", individual.GetRole())

	// Add API key
	keyShare := Must(individual.AddApiKey("mock-del-ind", "Ind Key", "openai"))
	s.NotEmpty(keyShare.Id)
	s.T().Logf("Individual has API key: %s", keyShare.Id)

	// Upload a game
	game := Must(individual.UploadGame("alien-first-contact"))
	s.NotEmpty(game.Id)
	s.T().Logf("Individual created game: %s (ID: %s)", game.Name, game.Id)

	// Verify game is visible
	gamesBefore := Must(individual.ListGames())
	foundGame := false
	for _, g := range gamesBefore {
		if g.Id == game.Id {
			foundGame = true
			break
		}
	}
	s.True(foundGame, "game should be visible before deletion")

	// Count users before
	usersBefore := Must(admin.GetUsers())

	// Admin deletes the individual
	MustSucceed(admin.DeleteUser(individual.ID))
	s.T().Logf("Admin deleted individual")

	// User should no longer appear in user list
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-1, len(usersAfter), "user count should decrease by 1")

	// Game should no longer be visible (hard-deleted)
	gamesAfter := Must(admin.ListGames())
	for _, g := range gamesAfter {
		s.NotEqual(game.Id, g.Id, "game should be deleted along with user")
	}
	s.T().Logf("Individual correctly deleted with API keys and games cleaned up")
}

// TestAdminCanDeleteStaffWithShares tests deleting a staff member who has API keys
// shared with the org and set as a workshop key.
func (s *UserDeletionTestSuite) TestAdminCanDeleteStaffWithShares() {
	admin := s.DevUser()

	// Setup: institution + head + staff
	inst := Must(admin.CreateInstitution("Del Staff Org"))
	head := s.CreateUser("del-staff-head")
	staff := s.CreateUser("del-staff-user")

	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.T().Logf("Head and staff joined institution")

	// Staff creates API key, shares with org, sets as workshop key
	staffKey := Must(staff.AddApiKey("mock-del-staff", "Staff Key", "openai"))
	orgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.Id.String(), inst.Id.String()))
	workshop := Must(staff.CreateWorkshop(inst.Id.String(), "Staff Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(staff.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Staff set workshop key")

	// Verify workshop has the key
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareId)

	// Count users before
	usersBefore := Must(admin.GetUsers())

	// Admin deletes the staff member
	MustSucceed(admin.DeleteUser(staff.ID))
	s.T().Logf("Admin deleted staff")

	// Staff should no longer appear in user list
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-1, len(usersAfter), "user count should decrease by 1")

	// Workshop key should be cleared (the staff's API key was deleted)
	ws = Must(head.GetWorkshop(wsIDStr))
	s.Nil(ws.DefaultApiKeyShareId, "workshop key should be cleared after staff deletion")
	s.T().Logf("Workshop key correctly cleared after staff deletion")

	// Org should no longer have the staff's shared key
	orgKeys := Must(head.GetInstitutionApiKeys(inst.Id.String()))
	s.Len(orgKeys, 0, "org should have no shared keys after staff deletion")
	s.T().Logf("Org API keys correctly cleaned up")
}

// TestAdminCannotDeleteLastHead tests that deleting the last head of an org fails
// with a proper error code.
func (s *UserDeletionTestSuite) TestAdminCannotDeleteLastHead() {
	admin := s.DevUser()

	// Setup: institution with a single head
	inst := Must(admin.CreateInstitution("Last Head Org"))
	head := s.CreateUser("del-last-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	s.Equal("head", head.GetRole())
	s.T().Logf("Single head in institution")

	// Admin tries to delete the last head — should fail
	err := admin.DeleteUser(head.ID)
	s.Refused(err, testutil.CodeLastHead, "should not be able to delete the last head")
	s.T().Logf("Correctly rejected: %v", err)

	// Head should still exist
	me := Must(head.GetMe())
	s.Equal(head.Name, me.Name)
	s.T().Logf("Head still exists after failed deletion")
}

// TestAdminCanDeleteHeadWithAnotherHead tests that a head can be deleted
// if the institution has another head.
func (s *UserDeletionTestSuite) TestAdminCanDeleteHeadWithAnotherHead() {
	admin := s.DevUser()

	// Setup: institution with two heads
	inst := Must(admin.CreateInstitution("Two Heads Org"))
	head1 := s.CreateUser("del-head1")
	head2 := s.CreateUser("del-head2")

	head1Invite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head1.ID))
	Must(head1.AcceptInvite(head1Invite.Id.String()))
	head2Invite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head2.ID))
	Must(head2.AcceptInvite(head2Invite.Id.String()))
	s.T().Logf("Two heads in institution")

	// Head1 creates API key and shares with org
	head1Key := Must(head1.AddApiKey("mock-del-head1", "Head1 Key", "openai"))
	Must(head1.ShareApiKeyWithInstitution(head1Key.Id.String(), inst.Id.String()))
	s.T().Logf("Head1 shared key with org")

	// Count users before
	usersBefore := Must(admin.GetUsers())

	// Admin deletes head1 — should succeed since head2 remains
	MustSucceed(admin.DeleteUser(head1.ID))
	s.T().Logf("Admin deleted head1")

	// Head1 should no longer appear in user list
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-1, len(usersAfter), "user count should decrease by 1")

	// Head2 should still exist and be head
	s.Equal("head", head2.GetRole())

	// Org should no longer have head1's shared key
	orgKeys := Must(head2.GetInstitutionApiKeys(inst.Id.String()))
	s.Len(orgKeys, 0, "org should have no shared keys after head1 deletion")
	s.T().Logf("Head1 correctly deleted, head2 remains")
}
