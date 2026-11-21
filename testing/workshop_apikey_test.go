package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopApiKeyTestSuite tests the workshop API key lifecycle:
// sharing keys with orgs, setting workshop keys, guest access, and cascade cleanup.
type WorkshopApiKeyTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopApiKeySuite(t *testing.T) {
	s := &WorkshopApiKeyTestSuite{}
	s.SuiteName = "Workshop API Key Tests"
	suite.Run(t, s)
}

// TestWorkshopKeyLifecycle tests the full lifecycle:
// 1. Head creates API key, shares with org
// 2. Head creates workshop, sets shared key as workshop key
// 3. Guest joins workshop and can see API key is available
// 4. Head deletes the API key (cascade) — workshop key is cleared automatically
// 5. Guest can no longer see an available API key
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyLifecycle() {
	admin := s.DevUser()

	// Setup: create institution, head user, and invite as head
	inst := Must(admin.CreateInstitution("Key Lifecycle Org"))
	head := s.CreateUser("ws-head")
	invite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.Id.String()))
	s.Equal("head", head.GetRole())
	s.T().Logf("Head joined institution")

	// Head adds an API key
	keyShare := Must(head.AddApiKey("mock-ws-key", "Workshop Key", "openai"))
	s.NotEmpty(keyShare.Id)
	s.T().Logf("Head added API key: shareID=%s, apiKeyID=%s", keyShare.Id, keyShare.ApiKeyId)

	// Head shares the key with the institution
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))
	s.NotEmpty(orgShare.Id)
	s.T().Logf("Head shared key with institution: orgShareID=%s", orgShare.Id)

	// Head creates a workshop
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Key Test Workshop"))
	s.NotEmpty(workshop.Id)
	s.T().Logf("Head created workshop: %s", workshop.Id)

	// Head sets the org share as the workshop default API key
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	updatedWs := Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.NotNil(updatedWs.DefaultApiKeyShareId, "workshop should have a default API key")
	s.T().Logf("Workshop key set to orgShare: %s", orgShareIDStr)

	// Head uploads a game (needed for API key status check)
	game := Must(head.UploadGame("alien-first-contact"))
	s.NotEmpty(game.Id)
	s.T().Logf("Head uploaded game: %s (ID: %s)", game.Name, game.Id)

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode")

	// Verify head can see API key is available for the game
	headAvailable := Must(head.GetApiKeyStatus(game.Id.String()))
	s.True(headAvailable, "head should have API key available via workshop key")
	s.T().Logf("Head: API key available = %v", headAvailable)

	// Create a Guest through the workshop invite link
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	s.Require().NotNil(guestInvite.InviteToken)
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop: %s (ID: %s)", guest.Name, guest.ID)

	// Guest can see API key is available (resolved via workshop key)
	guestAvailable := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.True(guestAvailable, "guest should have API key available via workshop key")
	s.T().Logf("Guest: API key available = %v", guestAvailable)

	// Head deletes the API key (cascade: removes key + all shares including org share)
	MustSucceed(head.DeleteApiKey(keyShare.Id.String(), true))
	s.T().Logf("Head deleted API key (cascade)")

	// Workshop key should be cleared automatically (the share it referenced was deleted)
	clearedWs := Must(head.GetWorkshop(wsIDStr))
	s.Nil(clearedWs.DefaultApiKeyShareId, "workshop key should be cleared after API key deletion")
	s.T().Logf("Workshop key correctly cleared after API key deletion")

	// Guest can no longer see an available API key
	guestAvailableAfter := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.False(guestAvailableAfter, "guest should NOT have API key available after key deletion")
	s.T().Logf("Guest: API key available after deletion = %v", guestAvailableAfter)
}

// TestWorkshopKeyRemovedViaOrgShareDeletion tests that removing the org share
// (without deleting the underlying key) also clears the workshop key.
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyRemovedViaOrgShareDeletion() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Share Removal Org"))
	head := s.CreateUser("ws-head-share")
	invite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.Id.String()))
	s.T().Logf("Head joined institution")

	// Head adds API key and shares with org
	keyShare := Must(head.AddApiKey("mock-ws-key-2", "Workshop Key 2", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))
	s.T().Logf("Head added key and shared with org: orgShareID=%s", orgShare.Id)

	// Head creates workshop and sets org share as workshop key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Share Removal Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Workshop key set")

	// Head uploads a game
	game := Must(head.UploadGame("arctic-expedition"))
	s.T().Logf("Head uploaded game: %s", game.Name)

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Create guest
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined: %s", guest.Name)

	// Guest can see API key is available
	available := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.True(available, "guest should have API key available")
	s.T().Logf("Guest: API key available = %v", available)

	// Head removes ONLY the org share (not the underlying key)
	// This is a non-cascade delete of the org share
	MustSucceed(head.DeleteApiKey(orgShare.Id.String(), false))
	s.T().Logf("Head removed org share (non-cascade)")

	// Workshop key should be cleared (the share it referenced was deleted)
	clearedWs := Must(head.GetWorkshop(wsIDStr))
	s.Nil(clearedWs.DefaultApiKeyShareId, "workshop key should be cleared after org share removal")
	s.T().Logf("Workshop key correctly cleared after org share removal")

	// Guest can no longer see an available API key
	availableAfter := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.False(availableAfter, "guest should NOT have API key available after org share removal")
	s.T().Logf("Guest: API key available after share removal = %v", availableAfter)

	// The underlying key still exists (head's personal share)
	keys := Must(head.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "head should still have 1 API key")
	s.T().Logf("Head still has the underlying API key")
}

// TestWorkshopKeyOverwriteByColleague tests that another head/staff in the same org
// can overwrite the workshop API key with their own shared key.
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyOverwriteByColleague() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Overwrite Org"))
	head := s.CreateUser("ws-head-ow")
	staff := s.CreateUser("ws-staff-ow")

	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.T().Logf("Head and staff joined institution")

	// Head creates key, shares with org, creates workshop, sets workshop key
	headKey := Must(head.AddApiKey("mock-head-ow", "Head Key", "openai"))
	headOrgShare := Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), inst.Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Overwrite Workshop"))
	wsIDStr := workshop.Id.String()
	headShareIDStr := headOrgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &headShareIDStr))
	s.T().Logf("Head set workshop key")

	// Verify workshop has head's key
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareId)
	s.Equal(headOrgShare.Id, *ws.DefaultApiKeyShareId)
	s.T().Logf("Workshop key is head's share: %s", headOrgShare.Id)

	// Staff creates their own key, shares with org
	staffKey := Must(staff.AddApiKey("mock-staff-ow", "Staff Key", "openai"))
	staffOrgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.Id.String(), inst.Id.String()))
	s.T().Logf("Staff shared key with org: %s", staffOrgShare.Id)

	// Staff overwrites the workshop key with their own
	staffShareIDStr := staffOrgShare.Id.String()
	Must(staff.SetWorkshopApiKey(wsIDStr, &staffShareIDStr))
	s.T().Logf("Staff overwrote workshop key")

	// Verify workshop now has staff's key
	ws = Must(staff.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareId)
	s.Equal(staffOrgShare.Id, *ws.DefaultApiKeyShareId)
	s.T().Logf("Workshop key is now staff's share: %s", staffOrgShare.Id)
}

// TestColleagueCanRemoveOrgShare tests that any head/staff in the org can remove
// API key shares from colleagues, and that this clears the workshop key if it referenced that share.
func (s *WorkshopApiKeyTestSuite) TestColleagueCanRemoveOrgShare() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Colleague Share Org"))
	head := s.CreateUser("ws-head-col")
	staff := s.CreateUser("ws-staff-col")

	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.T().Logf("Head and staff joined institution")

	// Staff creates key, shares with org, sets as workshop key
	staffKey := Must(staff.AddApiKey("mock-staff-col", "Staff Key", "openai"))
	staffOrgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.Id.String(), inst.Id.String()))
	workshop := Must(staff.CreateWorkshop(inst.Id.String(), "Colleague Workshop"))
	wsIDStr := workshop.Id.String()
	staffShareIDStr := staffOrgShare.Id.String()
	Must(staff.SetWorkshopApiKey(wsIDStr, &staffShareIDStr))
	s.T().Logf("Staff set workshop key: %s", staffOrgShare.Id)

	// Verify workshop has staff's key
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareId)
	s.T().Logf("Workshop key set to staff's share")

	// Head (colleague) removes staff's org share
	MustSucceed(head.DeleteApiKey(staffOrgShare.Id.String(), false))
	s.T().Logf("Head removed staff's org share")

	// Workshop key should be cleared (the share it referenced was deleted)
	ws = Must(head.GetWorkshop(wsIDStr))
	s.Nil(ws.DefaultApiKeyShareId, "workshop key should be cleared after colleague removed the org share")
	s.T().Logf("Workshop key correctly cleared")

	// Staff's underlying key still exists (only the org share was removed)
	staffKeys := Must(staff.GetApiKeys())
	s.Len(staffKeys.ApiKeys, 1, "staff should still have 1 API key")
	s.T().Logf("Staff still has the underlying API key")
}

// TestWorkshopKeyAvailableToAllRoles verifies that all roles that should have
// access to a workshop key get it resolved for a game:
//   - head (in workshop mode)
//   - staff (in workshop mode)
//   - individual (in workshop mode)
//   - guest (always in workshop context, no mode switch needed)
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyAvailableToAllRoles() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Session Creation Org"))
	head := s.CreateUser("ws-session-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	s.Equal("head", head.GetRole())

	staff := s.CreateUser("ws-session-staff")
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.Equal("staff", staff.GetRole())

	// Individual: not in the institution, joins workshop via invite
	individual := s.CreateUser("ws-session-individual")
	s.Equal("individual", individual.GetRole())

	// Head adds a working mock key and shares with org
	mockKey := Must(head.AddApiKey("mock-session-ws-key", "Workshop Session Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(mockKey.Id.String(), inst.Id.String()))
	s.T().Logf("Head shared key with org: orgShareID=%s", orgShare.Id)

	// Head creates workshop and sets the org share as workshop key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Session Test Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Workshop key set")

	// Upload a game and make it public so all roles can access it
	game := Must(head.UploadGame("alien-first-contact"))
	game.Public = true
	Must(head.UpdateGame(game.Id.String(), game))
	s.T().Logf("Head uploaded game (public): %s (ID: %s)", game.Name, game.Id)

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode")

	// Staff enters workshop mode
	Must(staff.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Staff entered workshop mode")

	// Individual joins workshop via its invite link (enters workshop mode,
	// keeping their role)
	individualInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	MustSucceed(individual.AcceptWorkshopInviteByToken(*individualInvite.InviteToken))
	s.T().Logf("Individual joined workshop via invite")

	// Guest joins workshop anonymously (always in workshop context)
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop: %s", guest.Name)

	// All four roles must get the workshop key resolved
	type roleCase struct {
		user *testutil.UserClient
		role string
	}
	cases := []roleCase{
		{head, "head"},
		{staff, "staff"},
		{individual, "individual"},
		{guest, "guest"},
	}

	for _, tc := range cases {
		available := Must(tc.user.GetApiKeyStatus(game.Id.String()))
		s.True(available, "%s should have API key available via workshop key", tc.role)
		s.T().Logf("%s: API key available = %v", tc.role, available)
	}
}

// TestDeleteWorkshopCleansUpGuests tests that deleting a workshop also
// deletes the Guests that joined it.
func (s *WorkshopApiKeyTestSuite) TestDeleteWorkshopCleansUpGuests() {
	admin := s.DevUser()

	// Setup: institution with head
	inst := Must(admin.CreateInstitution("Cleanup Org"))
	head := s.CreateUser("ws-head-del")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	s.T().Logf("Head joined institution")

	// Head creates workshop
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Deletable Workshop"))
	wsIDStr := workshop.Id.String()
	s.T().Logf("Created workshop: %s", wsIDStr)

	// Create two guests via anonymous invites
	invite1 := Must(head.CreateWorkshopInvite(wsIDStr))
	guest1 := Must(s.JoinWorkshop(*invite1.InviteToken))
	s.T().Logf("Guest 1 joined: %s (ID: %s)", guest1.Name, guest1.ID)

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr))
	guest2 := Must(s.JoinWorkshop(*invite2.InviteToken))
	s.T().Logf("Guest 2 joined: %s (ID: %s)", guest2.Name, guest2.ID)

	// Verify guests exist (they can call GetMe)
	me1 := Must(guest1.GetMe())
	s.Equal(guest1.Name, me1.Name)
	me2 := Must(guest2.GetMe())
	s.Equal(guest2.Name, me2.Name)
	s.T().Logf("Both guests verified")

	// Count users before deletion
	usersBefore := Must(admin.GetUsers())
	s.T().Logf("Users before deletion: %d", len(usersBefore))

	// Head deletes the workshop
	MustSucceed(head.DeleteWorkshop(wsIDStr))
	s.T().Logf("Workshop deleted")

	// Guests no longer exist: their accounts are gone, so their
	// sessions are refused
	_, err := guest1.WhoAmI()
	s.Forbidden(err, "guest 1 should no longer exist after workshop deletion")

	_, err = guest2.WhoAmI()
	s.Forbidden(err, "guest 2 should no longer exist after workshop deletion")

	// Count users after deletion — should be 2 fewer
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-2, len(usersAfter), "should have 2 fewer users after workshop deletion")
	s.T().Logf("Users after deletion: %d (removed %d)", len(usersAfter), len(usersBefore)-len(usersAfter))
}
