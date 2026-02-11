package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// InstitutionDeletionTestSuite tests that admins can delete institutions
// with full cascade cleanup, and that non-admins cannot.
type InstitutionDeletionTestSuite struct {
	testutil.BaseSuite
}

func TestInstitutionDeletionSuite(t *testing.T) {
	s := &InstitutionDeletionTestSuite{}
	s.SuiteName = "Institution Deletion Tests"
	suite.Run(t, s)
}

// TestAdminCanDeleteInstitutionWithCascade tests that an admin can delete an institution
// and all its workshops, participants, member users, games, and API keys are cleaned up.
func (s *InstitutionDeletionTestSuite) TestAdminCanDeleteInstitutionWithCascade() {
	admin := s.DevUser()

	// Setup: institution + head + staff + workshop + participant + games + API keys
	inst := Must(admin.CreateInstitution("Cascade Del Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("cd-head")
	staff := s.CreateUser("cd-staff")

	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Head creates API key and shares with org
	headKey := Must(head.AddApiKey("mock-cd-head", "Head Key", "mock"))
	Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), instIDStr))
	s.T().Logf("Head shared key with org")

	// Create workshop with participant
	workshop := Must(head.CreateWorkshop(instIDStr, "Cascade Del Workshop"))
	wsIDStr := workshop.ID.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant creates a game
	game := Must(participant.UploadGame("alien-first-contact"))
	s.T().Logf("Participant created game: %s (ID: %s)", game.Name, game.ID)

	// Count users before
	usersBefore := Must(admin.GetUsers())
	s.T().Logf("Users before deletion: %d", len(usersBefore))

	// Admin deletes the institution
	MustSucceed(admin.DeleteInstitution(instIDStr))
	s.T().Logf("Admin deleted institution")

	// Institution should no longer exist
	institutions := Must(admin.ListInstitutions())
	for _, i := range institutions {
		s.NotEqual(inst.ID, i.ID, "institution should be deleted")
	}

	// Only participant should be deleted; head and staff survive
	usersAfter := Must(admin.GetUsers())
	s.T().Logf("Users after deletion: %d", len(usersAfter))
	s.Equal(len(usersBefore)-1, len(usersAfter), "only participant should be removed")

	// Head and staff should still exist
	headMe := Must(head.GetMe())
	s.Equal(head.Name, headMe.Name, "head should still exist")
	staffMe := Must(staff.GetMe())
	s.Equal(staff.Name, staffMe.Name, "staff should still exist")
	s.T().Logf("Head and staff survived institution deletion")

	// Participant's game should be gone
	games := Must(admin.ListGames())
	for _, g := range games {
		s.NotEqual(game.ID, g.ID, "participant's game should be deleted")
	}
	s.T().Logf("Institution deleted: participant + game removed, head + staff survived")
}

// TestAdminDeleteOrgMultipleUsersAndWorkshops tests deleting an org with 2 heads, 2 staff,
// 2 workshops each with 2 participants. Participants should be deleted, heads/staff should survive.
func (s *InstitutionDeletionTestSuite) TestAdminDeleteOrgMultipleUsersAndWorkshops() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Multi User Org"))
	instIDStr := inst.ID.String()

	// Create 2 heads and 2 staff
	head1 := s.CreateUser("mu-head1")
	head2 := s.CreateUser("mu-head2")
	staff1 := s.CreateUser("mu-staff1")
	staff2 := s.CreateUser("mu-staff2")

	h1Invite := Must(admin.InviteToInstitution(instIDStr, "head", head1.ID))
	Must(head1.AcceptInvite(h1Invite.ID.String()))
	h2Invite := Must(admin.InviteToInstitution(instIDStr, "head", head2.ID))
	Must(head2.AcceptInvite(h2Invite.ID.String()))
	s1Invite := Must(head1.InviteToInstitution(instIDStr, "staff", staff1.ID))
	Must(staff1.AcceptInvite(s1Invite.ID.String()))
	s2Invite := Must(head1.InviteToInstitution(instIDStr, "staff", staff2.ID))
	Must(staff2.AcceptInvite(s2Invite.ID.String()))
	s.T().Logf("2 heads + 2 staff joined")

	// Create 2 workshops, each with 2 participants
	ws1 := Must(head1.CreateWorkshop(instIDStr, "MU Workshop 1"))
	ws2 := Must(head1.CreateWorkshop(instIDStr, "MU Workshop 2"))

	var participants []*testutil.UserClient
	for _, ws := range []struct{ id string }{{ws1.ID.String()}, {ws2.ID.String()}} {
		for i := 0; i < 2; i++ {
			invite := Must(head1.CreateWorkshopInvite(ws.id, string(obj.RoleParticipant)))
			resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
			s.NoError(err)
			p := s.CreateUserWithToken(*resp.AuthToken)
			participants = append(participants, p)
		}
	}
	s.T().Logf("4 participants joined across 2 workshops")

	// Participants create games
	game1 := Must(participants[0].UploadGame("alien-first-contact"))
	game2 := Must(participants[2].UploadGame("alien-first-contact"))
	s.T().Logf("Participants created games: %s, %s", game1.ID, game2.ID)

	// Staff creates API key and shares with org
	staffKey := Must(staff1.AddApiKey("mock-mu-staff", "Staff Key", "mock"))
	Must(staff1.ShareApiKeyWithInstitution(staffKey.ID.String(), instIDStr))
	s.T().Logf("Staff shared key with org")

	// Count users before
	usersBefore := Must(admin.GetUsers())
	s.T().Logf("Users before: %d", len(usersBefore))

	// Admin deletes the institution
	MustSucceed(admin.DeleteInstitution(instIDStr))
	s.T().Logf("Admin deleted institution")

	// Institution gone
	institutions := Must(admin.ListInstitutions())
	for _, i := range institutions {
		s.NotEqual(inst.ID, i.ID, "institution should be deleted")
	}

	// 4 participants deleted, 4 non-participants (2 heads + 2 staff) survive
	usersAfter := Must(admin.GetUsers())
	s.T().Logf("Users after: %d", len(usersAfter))
	s.Equal(len(usersBefore)-4, len(usersAfter), "only 4 participants should be removed")

	// Verify all heads and staff still exist
	Must(head1.GetMe())
	Must(head2.GetMe())
	Must(staff1.GetMe())
	Must(staff2.GetMe())
	s.T().Logf("All 4 non-participants survived")

	// Verify participant games are gone
	games := Must(admin.ListGames())
	for _, g := range games {
		s.NotEqual(game1.ID, g.ID, "participant game1 should be deleted")
		s.NotEqual(game2.ID, g.ID, "participant game2 should be deleted")
	}
	s.T().Logf("All participant games deleted, all non-participants survived")
}

// TestHeadCannotDeleteInstitution tests that a head cannot delete their own institution.
func (s *InstitutionDeletionTestSuite) TestHeadCannotDeleteInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head No Del Org"))
	head := s.CreateUser("hnd-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head tries to delete — should fail
	err := head.DeleteInstitution(inst.ID.String())
	s.Error(err, "head should not be able to delete institution")
	s.T().Logf("Head correctly rejected: %v", err)

	// Institution should still exist
	institutions := Must(admin.ListInstitutions())
	found := false
	for _, i := range institutions {
		if i.ID == inst.ID {
			found = true
			break
		}
	}
	s.True(found, "institution should still exist")
}

// TestStaffCannotDeleteInstitution tests that a staff member cannot delete the institution.
func (s *InstitutionDeletionTestSuite) TestStaffCannotDeleteInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff No Del Org"))
	head := s.CreateUser("snd-head")
	staff := s.CreateUser("snd-staff")

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Staff tries to delete — should fail
	err := staff.DeleteInstitution(inst.ID.String())
	s.Error(err, "staff should not be able to delete institution")
	s.T().Logf("Staff correctly rejected: %v", err)
}

// TestIndividualCannotDeleteInstitution tests that an individual user cannot delete any institution.
func (s *InstitutionDeletionTestSuite) TestIndividualCannotDeleteInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Ind No Del Org"))
	individual := s.CreateUser("ind-no-del")

	// Individual tries to delete — should fail
	err := individual.DeleteInstitution(inst.ID.String())
	s.Error(err, "individual should not be able to delete institution")
	s.T().Logf("Individual correctly rejected: %v", err)
}
