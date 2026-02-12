package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopApiKeyTestSuite tests the workshop API key lifecycle:
// sharing keys with orgs, setting workshop keys, participant access, and cascade cleanup.
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
// 3. Participant joins workshop and can see API key is available
// 4. Head deletes the API key (cascade) — workshop key is cleared automatically
// 5. Participant can no longer see an available API key
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyLifecycle() {
	admin := s.DevUser()

	// Setup: create institution, head user, and invite as head
	inst := Must(admin.CreateInstitution("Key Lifecycle Org"))
	head := s.CreateUser("ws-head")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.Equal("head", head.GetRole())
	s.T().Logf("Head joined institution")

	// Head adds an API key
	keyShare := Must(head.AddApiKey("mock-ws-key", "Workshop Key", "mock"))
	s.NotEmpty(keyShare.ID)
	s.T().Logf("Head added API key: shareID=%s, apiKeyID=%s", keyShare.ID, keyShare.ApiKeyID)

	// Head shares the key with the institution
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	s.NotEmpty(orgShare.ID)
	s.T().Logf("Head shared key with institution: orgShareID=%s", orgShare.ID)

	// Head creates a workshop
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Key Test Workshop"))
	s.NotEmpty(workshop.ID)
	s.T().Logf("Head created workshop: %s", workshop.ID)

	// Head sets the org share as the workshop default API key
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	updatedWs := Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.NotNil(updatedWs.DefaultApiKeyShareID, "workshop should have a default API key")
	s.T().Logf("Workshop key set to orgShare: %s", orgShareIDStr)

	// Head uploads a game (needed for API key status check)
	game := Must(head.UploadGame("alien-first-contact"))
	s.NotEmpty(game.ID)
	s.T().Logf("Head uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode")

	// Verify head can see API key is available for the game
	headAvailable := Must(head.GetApiKeyStatus(game.ID.String()))
	s.True(headAvailable, "head should have API key available via workshop key")
	s.T().Logf("Head: API key available = %v", headAvailable)

	// Create a participant via anonymous workshop invite
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	s.NotNil(participantInvite.InviteToken)
	s.T().Logf("Created participant invite with token")

	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	s.NotNil(participantResp.AuthToken)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined workshop: %s (ID: %s)", participant.Name, participant.ID)

	// Participant can see API key is available (resolved via workshop key)
	participantAvailable := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.True(participantAvailable, "participant should have API key available via workshop key")
	s.T().Logf("Participant: API key available = %v", participantAvailable)

	// Head deletes the API key (cascade: removes key + all shares including org share)
	MustSucceed(head.DeleteApiKey(keyShare.ID.String(), true))
	s.T().Logf("Head deleted API key (cascade)")

	// Workshop key should be cleared automatically (the share it referenced was deleted)
	clearedWs := Must(head.GetWorkshop(wsIDStr))
	s.Nil(clearedWs.DefaultApiKeyShareID, "workshop key should be cleared after API key deletion")
	s.T().Logf("Workshop key correctly cleared after API key deletion")

	// Participant can no longer see an available API key
	participantAvailableAfter := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(participantAvailableAfter, "participant should NOT have API key available after key deletion")
	s.T().Logf("Participant: API key available after deletion = %v", participantAvailableAfter)
}

// TestWorkshopKeyRemovedViaOrgShareDeletion tests that removing the org share
// (without deleting the underlying key) also clears the workshop key.
func (s *WorkshopApiKeyTestSuite) TestWorkshopKeyRemovedViaOrgShareDeletion() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Share Removal Org"))
	head := s.CreateUser("ws-head-share")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head adds API key and shares with org
	keyShare := Must(head.AddApiKey("mock-ws-key-2", "Workshop Key 2", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	s.T().Logf("Head added key and shared with org: orgShareID=%s", orgShare.ID)

	// Head creates workshop and sets org share as workshop key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Share Removal Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Workshop key set")

	// Head uploads a game
	game := Must(head.UploadGame("arctic-expedition"))
	s.T().Logf("Head uploaded game: %s", game.Name)

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Create participant
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant can see API key is available
	available := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.True(available, "participant should have API key available")
	s.T().Logf("Participant: API key available = %v", available)

	// Head removes ONLY the org share (not the underlying key)
	// This is a non-cascade delete of the org share
	MustSucceed(head.DeleteApiKey(orgShare.ID.String(), false))
	s.T().Logf("Head removed org share (non-cascade)")

	// Workshop key should be cleared (the share it referenced was deleted)
	clearedWs := Must(head.GetWorkshop(wsIDStr))
	s.Nil(clearedWs.DefaultApiKeyShareID, "workshop key should be cleared after org share removal")
	s.T().Logf("Workshop key correctly cleared after org share removal")

	// Participant can no longer see an available API key
	availableAfter := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(availableAfter, "participant should NOT have API key available after org share removal")
	s.T().Logf("Participant: API key available after share removal = %v", availableAfter)

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

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Head and staff joined institution")

	// Head creates key, shares with org, creates workshop, sets workshop key
	headKey := Must(head.AddApiKey("mock-head-ow", "Head Key", "mock"))
	headOrgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Overwrite Workshop"))
	wsIDStr := workshop.ID.String()
	headShareIDStr := headOrgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &headShareIDStr))
	s.T().Logf("Head set workshop key")

	// Verify workshop has head's key
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareID)
	s.Equal(headOrgShare.ID, *ws.DefaultApiKeyShareID)
	s.T().Logf("Workshop key is head's share: %s", headOrgShare.ID)

	// Staff creates their own key, shares with org
	staffKey := Must(staff.AddApiKey("mock-staff-ow", "Staff Key", "mock"))
	staffOrgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.ID.String(), inst.ID.String()))
	s.T().Logf("Staff shared key with org: %s", staffOrgShare.ID)

	// Staff overwrites the workshop key with their own
	staffShareIDStr := staffOrgShare.ID.String()
	Must(staff.SetWorkshopApiKey(wsIDStr, &staffShareIDStr))
	s.T().Logf("Staff overwrote workshop key")

	// Verify workshop now has staff's key
	ws = Must(staff.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareID)
	s.Equal(staffOrgShare.ID, *ws.DefaultApiKeyShareID)
	s.T().Logf("Workshop key is now staff's share: %s", staffOrgShare.ID)
}

// TestColleagueCanRemoveOrgShare tests that any head/staff in the org can remove
// API key shares from colleagues, and that this clears the workshop key if it referenced that share.
func (s *WorkshopApiKeyTestSuite) TestColleagueCanRemoveOrgShare() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Colleague Share Org"))
	head := s.CreateUser("ws-head-col")
	staff := s.CreateUser("ws-staff-col")

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Head and staff joined institution")

	// Staff creates key, shares with org, sets as workshop key
	staffKey := Must(staff.AddApiKey("mock-staff-col", "Staff Key", "mock"))
	staffOrgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.ID.String(), inst.ID.String()))
	workshop := Must(staff.CreateWorkshop(inst.ID.String(), "Colleague Workshop"))
	wsIDStr := workshop.ID.String()
	staffShareIDStr := staffOrgShare.ID.String()
	Must(staff.SetWorkshopApiKey(wsIDStr, &staffShareIDStr))
	s.T().Logf("Staff set workshop key: %s", staffOrgShare.ID)

	// Verify workshop has staff's key
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareID)
	s.T().Logf("Workshop key set to staff's share")

	// Head (colleague) removes staff's org share
	MustSucceed(head.DeleteApiKey(staffOrgShare.ID.String(), false))
	s.T().Logf("Head removed staff's org share")

	// Workshop key should be cleared (the share it referenced was deleted)
	ws = Must(head.GetWorkshop(wsIDStr))
	s.Nil(ws.DefaultApiKeyShareID, "workshop key should be cleared after colleague removed the org share")
	s.T().Logf("Workshop key correctly cleared")

	// Staff's underlying key still exists (only the org share was removed)
	staffKeys := Must(staff.GetApiKeys())
	s.Len(staffKeys.ApiKeys, 1, "staff should still have 1 API key")
	s.T().Logf("Staff still has the underlying API key")
}

// TestDeleteWorkshopCleansUpParticipants tests that deleting a workshop also
// deletes the anonymous participant accounts that were created for it.
func (s *WorkshopApiKeyTestSuite) TestDeleteWorkshopCleansUpParticipants() {
	admin := s.DevUser()

	// Setup: institution with head
	inst := Must(admin.CreateInstitution("Cleanup Org"))
	head := s.CreateUser("ws-head-del")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head creates workshop
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Deletable Workshop"))
	wsIDStr := workshop.ID.String()
	s.T().Logf("Created workshop: %s", wsIDStr)

	// Create two participants via anonymous invites
	invite1 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	participant1 := s.CreateUserWithToken(*resp1.AuthToken)
	s.T().Logf("Participant 1 joined: %s (ID: %s)", participant1.Name, participant1.ID)

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	participant2 := s.CreateUserWithToken(*resp2.AuthToken)
	s.T().Logf("Participant 2 joined: %s (ID: %s)", participant2.Name, participant2.ID)

	// Verify participants exist (they can call GetMe)
	me1 := Must(participant1.GetMe())
	s.Equal(participant1.Name, me1.Name)
	me2 := Must(participant2.GetMe())
	s.Equal(participant2.Name, me2.Name)
	s.T().Logf("Both participants verified")

	// Count users before deletion
	usersBefore := Must(admin.GetUsers())
	s.T().Logf("Users before deletion: %d", len(usersBefore))

	// Head deletes the workshop
	MustSucceed(head.DeleteWorkshop(wsIDStr))
	s.T().Logf("Workshop deleted")

	// Participants should no longer exist (their accounts were deleted)
	_, err = participant1.GetMe()
	s.Error(err, "participant 1 should no longer exist after workshop deletion")
	s.T().Logf("Participant 1 correctly deleted")

	_, err = participant2.GetMe()
	s.Error(err, "participant 2 should no longer exist after workshop deletion")
	s.T().Logf("Participant 2 correctly deleted")

	// Count users after deletion — should be 2 fewer
	usersAfter := Must(admin.GetUsers())
	s.Equal(len(usersBefore)-2, len(usersAfter), "should have 2 fewer users after workshop deletion")
	s.T().Logf("Users after deletion: %d (removed %d)", len(usersAfter), len(usersBefore)-len(usersAfter))
}
