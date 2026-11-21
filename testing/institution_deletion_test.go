package testing

import (
	"testing"

	"cgl/testing/testutil"

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
// and all its workshops, guests, member users, games, and API keys are cleaned up.
func (s *InstitutionDeletionTestSuite) TestAdminCanDeleteInstitutionWithCascade() {
	admin := s.DevUser()

	// Setup: institution + head + staff + workshop + guest + games + API keys
	inst := Must(admin.CreateInstitution("Cascade Del Org"))
	instIDStr := inst.Id.String()

	head := s.CreateUser("cd-head")
	staff := s.CreateUser("cd-staff")

	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Head creates API key and shares with org
	headKey := Must(head.AddApiKey("mock-cd-head", "Head Key", "openai"))
	Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), instIDStr))
	s.T().Logf("Head shared key with org")

	// Create workshop with guest
	workshop := Must(head.CreateWorkshop(instIDStr, "Cascade Del Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))
	s.T().Logf("Guest joined: %s", guest.Name)

	// Guest creates a game
	game := Must(guest.UploadGame("alien-first-contact"))
	s.T().Logf("Guest created game: %s (ID: %s)", game.Name, game.Id)

	// Count users before
	usersBefore := Must(admin.GetUsers())
	s.T().Logf("Users before deletion: %d", len(usersBefore))

	// Admin deletes the institution
	MustSucceed(admin.DeleteInstitution(instIDStr))
	s.T().Logf("Admin deleted institution")

	// Institution should no longer exist
	institutions := Must(admin.ListInstitutions())
	for _, i := range institutions {
		s.NotEqual(inst.Id, i.Id, "institution should be deleted")
	}

	// Only guest should be deleted; head and staff survive
	usersAfter := Must(admin.GetUsers())
	s.T().Logf("Users after deletion: %d", len(usersAfter))
	s.Equal(len(usersBefore)-1, len(usersAfter), "only guest should be removed")

	// Head and staff should still exist with individual roles
	headMe := Must(head.GetMe())
	s.Equal(head.Name, headMe.Name, "head should still exist")
	s.Require().NotNil(headMe.Role, "head should have a role")
	s.Equal("individual", headMe.Role.Role, "head should become individual")
	staffMe := Must(staff.GetMe())
	s.Equal(staff.Name, staffMe.Name, "staff should still exist")
	s.Require().NotNil(staffMe.Role, "staff should have a role")
	s.Equal("individual", staffMe.Role.Role, "staff should become individual")
	s.T().Logf("Head and staff survived institution deletion with individual roles")

	// Guest's game should be gone
	games := Must(admin.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "guest's game should be deleted")
	}
	s.T().Logf("Institution deleted: guest + game removed, head + staff survived")
}

// TestAdminDeleteOrgMultipleUsersAndWorkshops tests deleting an org with 2 heads, 2 staff,
// 2 workshops each with 2 guests. Guests should be deleted, heads/staff should survive.
func (s *InstitutionDeletionTestSuite) TestAdminDeleteOrgMultipleUsersAndWorkshops() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Multi User Org"))
	instIDStr := inst.Id.String()

	// Create 2 heads and 2 staff
	head1 := s.CreateUser("mu-head1")
	head2 := s.CreateUser("mu-head2")
	staff1 := s.CreateUser("mu-staff1")
	staff2 := s.CreateUser("mu-staff2")

	h1Invite := Must(admin.InviteToInstitution(instIDStr, "head", head1.ID))
	Must(head1.AcceptInvite(h1Invite.Id.String()))
	h2Invite := Must(admin.InviteToInstitution(instIDStr, "head", head2.ID))
	Must(head2.AcceptInvite(h2Invite.Id.String()))
	s1Invite := Must(head1.InviteToInstitution(instIDStr, "staff", staff1.ID))
	Must(staff1.AcceptInvite(s1Invite.Id.String()))
	s2Invite := Must(head1.InviteToInstitution(instIDStr, "staff", staff2.ID))
	Must(staff2.AcceptInvite(s2Invite.Id.String()))
	s.T().Logf("2 heads + 2 staff joined")

	// Create 2 workshops, each with 2 guests
	ws1 := Must(head1.CreateWorkshop(instIDStr, "MU Workshop 1"))
	ws2 := Must(head1.CreateWorkshop(instIDStr, "MU Workshop 2"))

	var guests []*testutil.UserClient
	for _, ws := range []struct{ id string }{{ws1.Id.String()}, {ws2.Id.String()}} {
		for i := 0; i < 2; i++ {
			invite := Must(head1.CreateWorkshopInvite(ws.id))
			p := Must(s.JoinWorkshop(*invite.InviteToken))
			guests = append(guests, p)
		}
	}
	s.T().Logf("4 guests joined across 2 workshops")

	// Guests create games
	game1 := Must(guests[0].UploadGame("alien-first-contact"))
	game2 := Must(guests[2].UploadGame("alien-first-contact"))
	s.T().Logf("Guests created games: %s, %s", game1.Id, game2.Id)

	// Staff creates API key and shares with org
	staffKey := Must(staff1.AddApiKey("mock-mu-staff", "Staff Key", "openai"))
	Must(staff1.ShareApiKeyWithInstitution(staffKey.Id.String(), instIDStr))
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
		s.NotEqual(inst.Id, i.Id, "institution should be deleted")
	}

	// 4 guests deleted, 4 non-guests (2 heads + 2 staff) survive
	usersAfter := Must(admin.GetUsers())
	s.T().Logf("Users after: %d", len(usersAfter))
	s.Equal(len(usersBefore)-4, len(usersAfter), "only 4 guests should be removed")

	// Verify all heads and staff still exist
	Must(head1.GetMe())
	Must(head2.GetMe())
	Must(staff1.GetMe())
	Must(staff2.GetMe())
	s.T().Logf("All 4 non-guests survived")

	// Verify guest games are gone
	games := Must(admin.ListGames())
	for _, g := range games {
		s.NotEqual(game1.Id, g.Id, "guest game1 should be deleted")
		s.NotEqual(game2.Id, g.Id, "guest game2 should be deleted")
	}
	s.T().Logf("All guest games deleted, all non-guests survived")
}

// TestDeleteInstitutionUnlinksMemberGamesDeletesGuestGames tests that when an institution
// is deleted, guest games are removed but head/staff games are preserved (unlinked from workshop).
func (s *InstitutionDeletionTestSuite) TestDeleteInstitutionUnlinksMemberGamesDeletesGuestGames() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Game Unlink Org"))
	instIDStr := inst.Id.String()

	head := s.CreateUser("gu-head")
	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	staff := s.CreateUser("gu-staff")
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Create workshop with a guest
	workshop := Must(head.CreateWorkshop(instIDStr, "Game Unlink Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	// Head creates a game (member game — should be preserved)
	headGame := Must(head.UploadGame("alien-first-contact"))
	s.T().Logf("Head created game: %s (ID: %s)", headGame.Name, headGame.Id)

	// Staff creates a game (member game — should be preserved)
	staffGame := Must(staff.UploadGame("alien-first-contact"))
	s.T().Logf("Staff created game: %s (ID: %s)", staffGame.Name, staffGame.Id)

	// Guest creates a game (should be deleted)
	guestGame := Must(guest.UploadGame("alien-first-contact"))
	s.T().Logf("Guest created game: %s (ID: %s)", guestGame.Name, guestGame.Id)

	// Delete the institution
	MustSucceed(admin.DeleteInstitution(instIDStr))
	s.T().Logf("Admin deleted institution")

	// Head's game should still exist (unlinked from workshop)
	fetchedHeadGame, err := head.GetGameByID(headGame.Id.String())
	s.NoError(err, "head's game should survive institution deletion")
	s.Equal(headGame.Id, fetchedHeadGame.Id)
	s.Nil(fetchedHeadGame.WorkshopId, "head's game should be unlinked from workshop")
	s.T().Logf("Head's game survived and is unlinked")

	// Staff's game should still exist (unlinked from workshop)
	fetchedStaffGame, err := staff.GetGameByID(staffGame.Id.String())
	s.NoError(err, "staff's game should survive institution deletion")
	s.Equal(staffGame.Id, fetchedStaffGame.Id)
	s.Nil(fetchedStaffGame.WorkshopId, "staff's game should be unlinked from workshop")
	s.T().Logf("Staff's game survived and is unlinked")

	// Guest's game should be gone (guest user deleted)
	_, err = admin.GetGameByID(guestGame.Id.String())
	s.NotFound(err, "guest's game should be deleted with institution")
	s.T().Logf("Guest's game correctly deleted")

	// Head and staff should now have individual roles (not left without a role)
	headMe := Must(head.GetMe())
	s.Require().NotNil(headMe.Role, "head should still have a role after institution deletion")
	s.Equal("individual", headMe.Role.Role, "head should become individual after institution deletion")
	s.Nil(headMe.Role.Institution, "head should have no institution after institution deletion")
	s.T().Logf("Head has individual role")

	staffMe := Must(staff.GetMe())
	s.Require().NotNil(staffMe.Role, "staff should still have a role after institution deletion")
	s.Equal("individual", staffMe.Role.Role, "staff should become individual after institution deletion")
	s.Nil(staffMe.Role.Institution, "staff should have no institution after institution deletion")
	s.T().Logf("Staff has individual role")
}

// TestDeleteWorkshopUnlinksMemberGames tests that deleting a single workshop
// unlinks member games and deletes guest games.
func (s *InstitutionDeletionTestSuite) TestDeleteWorkshopUnlinksMemberGames() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("WS Del Org"))
	instIDStr := inst.Id.String()

	head := s.CreateUser("wsd-head")
	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(instIDStr, "WS Del Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	// Head creates a game in the workshop context
	headGame := Must(head.UploadGame("alien-first-contact"))
	s.T().Logf("Head created game: %s", headGame.Id)

	// Guest creates a game
	guestGame := Must(guest.UploadGame("alien-first-contact"))
	s.T().Logf("Guest created game: %s", guestGame.Id)

	// Delete the workshop (not the institution)
	MustSucceed(head.DeleteWorkshop(wsIDStr))
	s.T().Logf("Head deleted workshop")

	// Head's game should still exist (unlinked)
	fetchedHeadGame, err := head.GetGameByID(headGame.Id.String())
	s.NoError(err, "head's game should survive workshop deletion")
	s.Equal(headGame.Id, fetchedHeadGame.Id)
	s.Nil(fetchedHeadGame.WorkshopId, "head's game should be unlinked from workshop")
	s.T().Logf("Head's game survived and is unlinked")

	// Guest's game should be gone
	_, err = admin.GetGameByID(guestGame.Id.String())
	s.NotFound(err, "guest's game should be deleted with workshop")
	s.T().Logf("Guest's game correctly deleted")
}

// TestHeadCannotDeleteInstitution tests that a head cannot delete their own institution.
func (s *InstitutionDeletionTestSuite) TestHeadCannotDeleteInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head No Del Org"))
	head := s.CreateUser("hnd-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head tries to delete — should fail
	err := head.DeleteInstitution(inst.Id.String())
	s.Forbidden(err, "head should not be able to delete institution")
	s.T().Logf("Head correctly rejected: %v", err)

	// Institution should still exist
	institutions := Must(admin.ListInstitutions())
	found := false
	for _, i := range institutions {
		if i.Id == inst.Id {
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

	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Staff tries to delete — should fail
	err := staff.DeleteInstitution(inst.Id.String())
	s.Forbidden(err, "staff should not be able to delete institution")
	s.T().Logf("Staff correctly rejected: %v", err)
}

// TestIndividualCannotDeleteInstitution tests that an individual user cannot delete any institution.
func (s *InstitutionDeletionTestSuite) TestIndividualCannotDeleteInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Ind No Del Org"))
	individual := s.CreateUser("ind-no-del")

	// Individual tries to delete — should fail: an organisation they are not
	// in does not exist for them
	err := individual.DeleteInstitution(inst.Id.String())
	s.NotFound(err, "individual should not be able to delete institution")
	s.T().Logf("Individual correctly rejected: %v", err)
}
