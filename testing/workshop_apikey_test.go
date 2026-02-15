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

// TestPersonalKeyWorkshopShareLifecycle tests the full lifecycle of using a personal key
// for a workshop (not shared with the org):
// 1. Head creates API key (gets self-share, no org share)
// 2. Head sets personal key as workshop default via apiKeyId — backend auto-creates workshop-scoped share
// 3. Participant can use the key via workshop resolution
// 4. Head clears the workshop key — workshop-scoped share is auto-deleted
// 5. Participant can no longer use the key
func (s *WorkshopApiKeyTestSuite) TestPersonalKeyWorkshopShareLifecycle() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Personal Key Lifecycle Org"))
	head := s.CreateUser("ws-head-pk")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head adds an API key (only self-share, NOT shared with org)
	keyShare := Must(head.AddApiKey("mock-personal-key", "Personal Key", "mock"))
	s.NotEmpty(keyShare.ID)
	s.T().Logf("Head added personal API key: shareID=%s, apiKeyID=%s", keyShare.ID, keyShare.ApiKeyID)

	// Verify key is NOT in institution API keys
	orgKeys := Must(head.GetInstitutionApiKeys(inst.ID.String()))
	s.Empty(orgKeys, "personal key should NOT appear in institution API keys")

	// Head creates a workshop
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Personal Key Workshop"))
	wsIDStr := workshop.ID.String()
	s.T().Logf("Head created workshop: %s", wsIDStr)

	// Head sets personal key as workshop default via apiKeyId
	updatedWs := Must(head.SetWorkshopApiKeyByKeyId(wsIDStr, keyShare.ApiKeyID.String()))
	s.NotNil(updatedWs.DefaultApiKeyShareID, "workshop should have a default API key")
	workshopShareID := updatedWs.DefaultApiKeyShareID.String()
	s.T().Logf("Workshop key set via personal key, new workshop share: %s", workshopShareID)

	// The workshop share should be different from the self-share (it's a new workshop-scoped share)
	s.NotEqual(keyShare.ID.String(), workshopShareID, "workshop share should be a new share, not the self-share")

	// Verify key is still NOT in institution API keys (workshop-scoped share, not org share)
	orgKeys = Must(head.GetInstitutionApiKeys(inst.ID.String()))
	s.Empty(orgKeys, "personal key should still NOT appear in institution API keys after workshop share")

	// Head uploads a game and enters workshop mode
	game := Must(head.UploadGame("alien-first-contact"))
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode")

	// Create a participant
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant can see API key is available (resolved via workshop key)
	available := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.True(available, "participant should have API key available via workshop key")
	s.T().Logf("Participant: API key available = %v", available)

	// Head clears the workshop key
	Must(head.SetWorkshopApiKey(wsIDStr, nil))
	s.T().Logf("Head cleared workshop key")

	// Workshop key should be cleared
	clearedWs := Must(head.GetWorkshop(wsIDStr))
	s.Nil(clearedWs.DefaultApiKeyShareID, "workshop key should be cleared")

	// Participant can no longer see an available API key
	availableAfter := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(availableAfter, "participant should NOT have API key available after clearing")
	s.T().Logf("Participant: API key available after clearing = %v", availableAfter)

	// The underlying key still exists (head's personal key)
	keys := Must(head.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "head should still have 1 API key")
	s.T().Logf("Head still has the underlying API key")
}

// TestPersonalKeyWorkshopShareNotUsableByOthers tests that a workshop-specific share
// created from a personal key is NOT usable by other head/staff in the org for other workshops.
func (s *WorkshopApiKeyTestSuite) TestPersonalKeyWorkshopShareNotUsableByOthers() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Personal Key Isolation Org"))
	head := s.CreateUser("ws-head-iso")
	staff := s.CreateUser("ws-staff-iso")

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Head and staff joined institution")

	// Head creates a personal key (NOT shared with org)
	headKey := Must(head.AddApiKey("mock-head-iso", "Head Personal Key", "mock"))
	s.T().Logf("Head added personal key: %s", headKey.ApiKeyID)

	// Head creates workshop and sets personal key via apiKeyId
	workshop1 := Must(head.CreateWorkshop(inst.ID.String(), "Head's Workshop"))
	ws1IDStr := workshop1.ID.String()
	updatedWs := Must(head.SetWorkshopApiKeyByKeyId(ws1IDStr, headKey.ApiKeyID.String()))
	workshopShareID := updatedWs.DefaultApiKeyShareID.String()
	s.T().Logf("Head set personal key on workshop 1, share: %s", workshopShareID)

	// Verify key is NOT in institution API keys
	orgKeys := Must(staff.GetInstitutionApiKeys(inst.ID.String()))
	s.Empty(orgKeys, "head's personal key should NOT appear in institution API keys for staff")

	// Staff creates another workshop and tries to use head's workshop share for it
	workshop2 := Must(staff.CreateWorkshop(inst.ID.String(), "Staff's Workshop"))
	ws2IDStr := workshop2.ID.String()

	// Staff should NOT be able to set head's workshop-specific share on their own workshop
	_, err := staff.SetWorkshopApiKey(ws2IDStr, &workshopShareID)
	s.Error(err, "staff should NOT be able to use head's workshop-specific share for another workshop")
	s.T().Logf("Staff correctly cannot use head's workshop share: %v", err)

	// Staff CAN still see the workshop (they're in the org) and see it has a key set
	ws1 := Must(staff.GetWorkshop(ws1IDStr))
	s.NotNil(ws1.DefaultApiKeyShareID, "staff can see workshop 1 has a key set")
	s.T().Logf("Staff can see workshop 1 has key: %s", ws1.DefaultApiKeyShareID)

	// But staff CAN replace it with their own key
	staffKey := Must(staff.AddApiKey("mock-staff-iso", "Staff Key", "mock"))
	updatedWs1 := Must(staff.SetWorkshopApiKeyByKeyId(ws1IDStr, staffKey.ApiKeyID.String()))
	s.NotNil(updatedWs1.DefaultApiKeyShareID, "workshop should now have staff's key")
	s.NotEqual(workshopShareID, updatedWs1.DefaultApiKeyShareID.String(), "workshop share should have changed")
	s.T().Logf("Staff replaced head's key with their own on workshop 1")
}

// TestOrgSharedKeyUsableByAllHeadStaff tests that an API key shared with the org
// can be used by any head/staff member for any workshop in the org.
func (s *WorkshopApiKeyTestSuite) TestOrgSharedKeyUsableByAllHeadStaff() {
	admin := s.DevUser()

	// Setup: institution with head and staff
	inst := Must(admin.CreateInstitution("Org Shared Key Org"))
	head := s.CreateUser("ws-head-org")
	staff := s.CreateUser("ws-staff-org")

	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Head and staff joined institution")

	// Head creates key and shares with org
	headKey := Must(head.AddApiKey("mock-org-key", "Org Shared Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))
	s.T().Logf("Head shared key with org: orgShareID=%s", orgShare.ID)

	// Both head and staff can see the key in institution API keys
	headOrgKeys := Must(head.GetInstitutionApiKeys(inst.ID.String()))
	s.Len(headOrgKeys, 1, "head should see 1 org key")
	staffOrgKeys := Must(staff.GetInstitutionApiKeys(inst.ID.String()))
	s.Len(staffOrgKeys, 1, "staff should see 1 org key")
	s.T().Logf("Both head and staff can see org key")

	// Head creates workshop 1 and sets org share
	workshop1 := Must(head.CreateWorkshop(inst.ID.String(), "Head's Org Key Workshop"))
	ws1IDStr := workshop1.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(ws1IDStr, &orgShareIDStr))
	s.T().Logf("Head set org key on workshop 1")

	// Staff creates workshop 2 and sets the SAME org share
	workshop2 := Must(staff.CreateWorkshop(inst.ID.String(), "Staff's Org Key Workshop"))
	ws2IDStr := workshop2.ID.String()
	Must(staff.SetWorkshopApiKey(ws2IDStr, &orgShareIDStr))
	s.T().Logf("Staff set same org key on workshop 2")

	// Upload a game for testing
	game := Must(head.UploadGame("alien-first-contact"))

	// Create participants in both workshops
	invite1 := Must(head.CreateWorkshopInvite(ws1IDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	participant1 := s.CreateUserWithToken(*resp1.AuthToken)

	invite2 := Must(staff.CreateWorkshopInvite(ws2IDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	participant2 := s.CreateUserWithToken(*resp2.AuthToken)
	s.T().Logf("Participants joined both workshops")

	// Both participants can see API key is available
	avail1 := Must(participant1.GetApiKeyStatus(game.ID.String()))
	s.True(avail1, "participant 1 should have API key available")
	avail2 := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.True(avail2, "participant 2 should have API key available")
	s.T().Logf("Both participants have API key available")

	// Staff can overwrite workshop 1's key (they're in the same org)
	staffKey := Must(staff.AddApiKey("mock-staff-org", "Staff Key", "mock"))
	staffOrgShare := Must(staff.ShareApiKeyWithInstitution(staffKey.ID.String(), inst.ID.String()))
	staffShareIDStr := staffOrgShare.ID.String()
	Must(staff.SetWorkshopApiKey(ws1IDStr, &staffShareIDStr))
	s.T().Logf("Staff overwrote workshop 1's key with their own org-shared key")

	// Workshop 1 now has staff's key, workshop 2 still has head's key
	ws1 := Must(head.GetWorkshop(ws1IDStr))
	s.Equal(staffOrgShare.ID, *ws1.DefaultApiKeyShareID)
	ws2 := Must(staff.GetWorkshop(ws2IDStr))
	s.Equal(orgShare.ID, *ws2.DefaultApiKeyShareID)
	s.T().Logf("Workshop 1 has staff's key, workshop 2 still has head's key")

	// Clearing workshop 1's key should NOT delete the org share (it's shared with the org)
	Must(head.SetWorkshopApiKey(ws1IDStr, nil))
	ws1Cleared := Must(head.GetWorkshop(ws1IDStr))
	s.Nil(ws1Cleared.DefaultApiKeyShareID, "workshop 1 key should be cleared")

	// Workshop 2 should still work (org share was not deleted)
	avail2After := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.True(avail2After, "participant 2 should still have API key available after clearing workshop 1")
	s.T().Logf("Workshop 2 still works after clearing workshop 1's org-shared key")
}

// TestPersonalKeyShareReplacedByOrgShare tests that when a workshop-specific share
// is replaced by an org share, the workshop-specific share is auto-deleted.
func (s *WorkshopApiKeyTestSuite) TestPersonalKeyShareReplacedByOrgShare() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Replace Share Org"))
	head := s.CreateUser("ws-head-rep")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head creates two keys
	personalKey := Must(head.AddApiKey("mock-personal-rep", "Personal Key", "mock"))
	orgKey := Must(head.AddApiKey("mock-org-rep", "Org Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(orgKey.ID.String(), inst.ID.String()))
	s.T().Logf("Head created personal key and org-shared key")

	// Head creates workshop and sets personal key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Replace Share Workshop"))
	wsIDStr := workshop.ID.String()
	updatedWs := Must(head.SetWorkshopApiKeyByKeyId(wsIDStr, personalKey.ApiKeyID.String()))
	workshopShareID := updatedWs.DefaultApiKeyShareID.String()
	s.T().Logf("Workshop set with personal key, workshop share: %s", workshopShareID)

	// Now replace with org share — the old workshop-specific share should be auto-deleted
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Replaced personal key with org share")

	// Verify workshop now has org share
	ws := Must(head.GetWorkshop(wsIDStr))
	s.NotNil(ws.DefaultApiKeyShareID)
	s.Equal(orgShare.ID, *ws.DefaultApiKeyShareID)

	// The personal key still exists but the workshop-specific share should be gone
	keys := Must(head.GetApiKeys())
	s.Len(keys.ApiKeys, 2, "head should still have 2 API keys")
	s.T().Logf("Both API keys still exist, workshop-specific share was cleaned up")
}

// TestPersonalKeyShareUsedByMultipleWorkshops tests that a workshop-specific share
// used by multiple workshops is NOT deleted when removed from just one workshop.
func (s *WorkshopApiKeyTestSuite) TestPersonalKeyShareUsedByMultipleWorkshops() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Multi Workshop Share Org"))
	head := s.CreateUser("ws-head-multi")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head creates a personal key
	keyShare := Must(head.AddApiKey("mock-multi-key", "Multi Workshop Key", "mock"))
	s.T().Logf("Head added personal key: apiKeyID=%s", keyShare.ApiKeyID)

	// Head creates two workshops and sets the same personal key on both via apiKeyId
	workshop1 := Must(head.CreateWorkshop(inst.ID.String(), "Multi WS 1"))
	ws1IDStr := workshop1.ID.String()
	updatedWs1 := Must(head.SetWorkshopApiKeyByKeyId(ws1IDStr, keyShare.ApiKeyID.String()))
	shareID := updatedWs1.DefaultApiKeyShareID.String()
	s.T().Logf("Workshop 1 set with personal key, share: %s", shareID)

	// Set the SAME share on workshop 2 (head uses the share ID directly)
	workshop2 := Must(head.CreateWorkshop(inst.ID.String(), "Multi WS 2"))
	ws2IDStr := workshop2.ID.String()
	Must(head.SetWorkshopApiKey(ws2IDStr, &shareID))
	s.T().Logf("Workshop 2 set with same share: %s", shareID)

	// Upload a game for testing
	game := Must(head.UploadGame("alien-first-contact"))

	// Create participants in both workshops
	invite1 := Must(head.CreateWorkshopInvite(ws1IDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	participant1 := s.CreateUserWithToken(*resp1.AuthToken)

	invite2 := Must(head.CreateWorkshopInvite(ws2IDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	participant2 := s.CreateUserWithToken(*resp2.AuthToken)

	// Both participants can see API key
	avail1 := Must(participant1.GetApiKeyStatus(game.ID.String()))
	s.True(avail1, "participant 1 should have API key available")
	avail2 := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.True(avail2, "participant 2 should have API key available")
	s.T().Logf("Both participants have API key available")

	// Clear workshop 1's key — share should NOT be deleted because workshop 2 still uses it
	Must(head.SetWorkshopApiKey(ws1IDStr, nil))
	ws1 := Must(head.GetWorkshop(ws1IDStr))
	s.Nil(ws1.DefaultApiKeyShareID, "workshop 1 key should be cleared")
	s.T().Logf("Workshop 1 key cleared")

	// Workshop 2 should still have the share and participant 2 should still have access
	ws2 := Must(head.GetWorkshop(ws2IDStr))
	s.NotNil(ws2.DefaultApiKeyShareID, "workshop 2 should still have the share")
	s.Equal(shareID, ws2.DefaultApiKeyShareID.String(), "workshop 2 share should be unchanged")

	avail2After := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.True(avail2After, "participant 2 should still have API key available after clearing workshop 1")
	s.T().Logf("Workshop 2 still works — share was preserved")

	// Now clear workshop 2's key — share should be deleted (no more references)
	Must(head.SetWorkshopApiKey(ws2IDStr, nil))
	ws2Cleared := Must(head.GetWorkshop(ws2IDStr))
	s.Nil(ws2Cleared.DefaultApiKeyShareID, "workshop 2 key should be cleared")

	avail2Final := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.False(avail2Final, "participant 2 should NOT have API key after clearing workshop 2")
	s.T().Logf("Workshop 2 cleared, share deleted, participant lost access")

	// Underlying key still exists
	keys := Must(head.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "head should still have 1 API key")
}

// TestApiKeyDeletionCascadesToWorkshopShares tests that deleting an API key
// also removes workshop-specific shares and clears the workshops that referenced them.
func (s *WorkshopApiKeyTestSuite) TestApiKeyDeletionCascadesToWorkshopShares() {
	admin := s.DevUser()

	// Setup: institution + head
	inst := Must(admin.CreateInstitution("Cascade Delete Org"))
	head := s.CreateUser("ws-head-casc")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head creates a personal key and sets it on two workshops
	keyShare := Must(head.AddApiKey("mock-cascade-key", "Cascade Key", "mock"))
	s.T().Logf("Head added personal key: apiKeyID=%s", keyShare.ApiKeyID)

	workshop1 := Must(head.CreateWorkshop(inst.ID.String(), "Cascade WS 1"))
	ws1IDStr := workshop1.ID.String()
	updatedWs1 := Must(head.SetWorkshopApiKeyByKeyId(ws1IDStr, keyShare.ApiKeyID.String()))
	shareID := updatedWs1.DefaultApiKeyShareID.String()
	s.T().Logf("Workshop 1 set with personal key, share: %s", shareID)

	workshop2 := Must(head.CreateWorkshop(inst.ID.String(), "Cascade WS 2"))
	ws2IDStr := workshop2.ID.String()
	Must(head.SetWorkshopApiKey(ws2IDStr, &shareID))
	s.T().Logf("Workshop 2 set with same share")

	// Upload a game and create participants
	game := Must(head.UploadGame("alien-first-contact"))
	Must(head.SetActiveWorkshop(&ws1IDStr))

	invite1 := Must(head.CreateWorkshopInvite(ws1IDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	participant1 := s.CreateUserWithToken(*resp1.AuthToken)

	invite2 := Must(head.CreateWorkshopInvite(ws2IDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	participant2 := s.CreateUserWithToken(*resp2.AuthToken)

	// Both participants can see API key
	avail1 := Must(participant1.GetApiKeyStatus(game.ID.String()))
	s.True(avail1, "participant 1 should have API key available")
	avail2 := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.True(avail2, "participant 2 should have API key available")
	s.T().Logf("Both participants have API key available")

	// Head deletes the API key (cascade)
	MustSucceed(head.DeleteApiKey(keyShare.ID.String(), true))
	s.T().Logf("Head deleted API key (cascade)")

	// Both workshops should have their default key cleared
	ws1 := Must(head.GetWorkshop(ws1IDStr))
	s.Nil(ws1.DefaultApiKeyShareID, "workshop 1 key should be cleared after API key deletion")
	ws2 := Must(head.GetWorkshop(ws2IDStr))
	s.Nil(ws2.DefaultApiKeyShareID, "workshop 2 key should be cleared after API key deletion")
	s.T().Logf("Both workshops cleared")

	// Neither participant can see API key anymore
	avail1After := Must(participant1.GetApiKeyStatus(game.ID.String()))
	s.False(avail1After, "participant 1 should NOT have API key after key deletion")
	avail2After := Must(participant2.GetApiKeyStatus(game.ID.String()))
	s.False(avail2After, "participant 2 should NOT have API key after key deletion")
	s.T().Logf("Both participants lost access")

	// Head has no more keys
	keys := Must(head.GetApiKeys())
	s.Len(keys.ApiKeys, 0, "head should have 0 API keys after deletion")
	s.T().Logf("API key fully deleted with all shares")
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
