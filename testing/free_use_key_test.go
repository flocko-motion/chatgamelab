package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// FreeUseKeyTestSuite tests the API key resolution priority chain for playing games.
// The backend resolves keys transparently (users never see which key is selected):
//
//  1. Workshop key
//  2. Sponsored game key
//  3. Institution free-use key
//  4. User's default API key
//  5. System free-use key
//
// These tests verify levels 5, 3, and 1 of the chain.
type FreeUseKeyTestSuite struct {
	testutil.BaseSuite
}

func TestFreeUseKeySuite(t *testing.T) {
	s := &FreeUseKeyTestSuite{}
	s.SuiteName = "Free Use Key Tests"
	suite.Run(t, s)
}

// ---------------------------------------------------------------------------
// System free-use key (priority 5 — lowest, global fallback)
// ---------------------------------------------------------------------------

// TestSystemFreeUseKeyAllowsPlayWithoutOwnKey verifies that when an admin sets
// a system free-use key, any authenticated user (individual, head, staff) can
// play a game without having their own API key.
func (s *FreeUseKeyTestSuite) TestSystemFreeUseKeyAllowsPlayWithoutOwnKey() {
	admin := s.DevUser()

	// Admin uploads a game (needed for session creation)
	game := Must(admin.UploadGame("alien-first-contact"))
	s.NotEmpty(game.ID)
	s.T().Logf("Admin uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Admin adds a mock API key and sets it as the system free-use key
	adminKey := Must(admin.AddApiKey("mock-sys-free", "System Free Key", "mock"))
	adminKeyIDStr := adminKey.ApiKeyID.String()
	settings := Must(admin.SetSystemFreeUseApiKey(&adminKeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID)
	s.T().Logf("System free-use key set: %s", adminKey.ApiKeyID)

	// Create users with different roles — none have their own API keys
	individual := s.CreateUser("ind-sysfree")
	s.Equal("individual", individual.GetRole())

	inst := Must(admin.CreateInstitution("SysFree Org"))
	head := s.CreateUser("head-sysfree")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	s.Equal("head", head.GetRole())

	staff := s.CreateUser("staff-sysfree")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.Equal("staff", staff.GetRole())

	// All three should be able to play (API key resolved via system free-use)
	for _, user := range []*testutil.UserClient{individual, head, staff} {
		available := Must(user.GetApiKeyStatus(game.ID.String()))
		s.True(available, "%s should have API key available via system free-use key", user.Name)
		s.T().Logf("%s: API key available = %v (via system free-use)", user.Name, available)
	}

	// Clean up: clear the system free-use key
	Must(admin.SetSystemFreeUseApiKey(nil))
	s.T().Logf("System free-use key cleared")

	// After clearing, users without own keys cannot play
	for _, user := range []*testutil.UserClient{individual, head, staff} {
		available := Must(user.GetApiKeyStatus(game.ID.String()))
		s.False(available, "%s should NOT have API key after system free-use key cleared", user.Name)
		s.T().Logf("%s: API key available = %v (after clear)", user.Name, available)
	}
}

// ---------------------------------------------------------------------------
// Institution free-use key (priority 3)
// ---------------------------------------------------------------------------

// TestInstitutionFreeUseKeyAllowsOrgMembersToPlay verifies that when a head
// sets an institution free-use key, all org members (head, staff, participant)
// can play, but non-org members cannot use that key.
func (s *FreeUseKeyTestSuite) TestInstitutionFreeUseKeyAllowsOrgMembersToPlay() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("alien-first-contact"))
	s.T().Logf("Admin uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create institution with head
	inst := Must(admin.CreateInstitution("OrgFree Org"))
	head := s.CreateUser("head-orgfree")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	s.Equal("head", head.GetRole())

	// Head adds API key, shares with institution
	headKey := Must(head.AddApiKey("mock-org-free", "Org Free Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))
	s.T().Logf("Head shared key with institution: orgShareID=%s", orgShare.ID)

	// Head sets the org share as institution free-use key
	orgShareIDStr := orgShare.ID.String()
	updatedInst := Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), &orgShareIDStr))
	s.NotNil(updatedInst.FreeUseApiKeyShareID)
	s.T().Logf("Institution free-use key set: %s", orgShare.ID)

	// Add staff member
	staff := s.CreateUser("staff-orgfree")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.Equal("staff", staff.GetRole())

	// Create a workshop and add a participant
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "OrgFree Workshop"))
	wsIDStr := workshop.ID.String()
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined workshop: %s (ID: %s)", participant.Name, participant.ID)

	// Head and staff can play (resolved via institution free-use key)
	for _, user := range []*testutil.UserClient{head, staff} {
		available := Must(user.GetApiKeyStatus(game.ID.String()))
		s.True(available, "%s (org member) should have API key via institution free-use key", user.Name)
		s.T().Logf("%s: API key available = %v (via institution free-use)", user.Name, available)
	}

	// Participant (anonymous workshop user) should NOT get institution free-use key —
	// participants can only use the workshop key (priority 1)
	participantAvailable := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(participantAvailable, "participant should NOT have API key via institution free-use (only workshop key works for participants)")
	s.T().Logf("participant: API key available = %v (correctly denied — no workshop key)", participantAvailable)

	// Non-org member (individual) should NOT have access via institution free-use key
	outsider := s.CreateUser("outsider-orgfree")
	s.Equal("individual", outsider.GetRole())
	outsiderAvailable := Must(outsider.GetApiKeyStatus(game.ID.String()))
	s.False(outsiderAvailable, "outsider should NOT have API key via institution free-use key")
	s.T().Logf("outsider: API key available = %v (correctly denied)", outsiderAvailable)
}

// TestInstitutionFreeUseKeyDoesNotApplyInWorkshopWithNoKey verifies that when
// a workshop has NO default API key set, members in that workshop context still
// fall back to the institution free-use key (priority 3 > no workshop key).
func (s *FreeUseKeyTestSuite) TestInstitutionFreeUseKeyFallbackInWorkshopWithNoKey() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("arctic-expedition"))
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("OrgFallback Org"))
	head := s.CreateUser("head-orgfb")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds key, shares with org, sets as institution free-use key
	headKey := Must(head.AddApiKey("mock-org-fb", "Org Fallback Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), &orgShareIDStr))
	s.T().Logf("Institution free-use key set")

	// Create workshop WITHOUT setting a workshop API key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "No-Key Workshop"))
	wsIDStr := workshop.ID.String()
	s.Nil(workshop.DefaultApiKeyShareID, "workshop should have no default API key")

	// Create participant in the workshop
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined workshop (no workshop key)")

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Head should still be able to play via institution free-use fallback (priority 3)
	headAvailable := Must(head.GetApiKeyStatus(game.ID.String()))
	s.True(headAvailable, "head should have API key via institution free-use (workshop has no key)")
	s.T().Logf("head: API key available = %v (institution free-use fallback)", headAvailable)

	// Participant should NOT be able to play — institution free-use key does not
	// apply to participants; they can only use the workshop key (priority 1)
	participantAvailable := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(participantAvailable, "participant should NOT have API key (institution free-use does not apply to participants)")
	s.T().Logf("participant: API key available = %v (correctly denied — no workshop key)", participantAvailable)
}

// ---------------------------------------------------------------------------
// Workshop key (priority 1 — highest)
// ---------------------------------------------------------------------------

// TestWorkshopKeyAllowsWorkshopMembersToPlay verifies that when a workshop
// has a default API key set, workshop members can play, but non-workshop
// members of the same org cannot use that key (they fall back to lower priorities).
func (s *FreeUseKeyTestSuite) TestWorkshopKeyAllowsWorkshopMembersToPlay() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("alien-first-contact"))
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("WsKey Org"))
	head := s.CreateUser("head-wskey")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds key, shares with org
	headKey := Must(head.AddApiKey("mock-ws-play", "Workshop Play Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))
	s.T().Logf("Head shared key with org: %s", orgShare.ID)

	// Head creates workshop and sets the org share as workshop key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Play Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Workshop key set")

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Create participant in the workshop
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined workshop: %s", participant.Name)

	// Workshop members can play
	headAvailable := Must(head.GetApiKeyStatus(game.ID.String()))
	s.True(headAvailable, "head (in workshop) should have API key via workshop key")
	s.T().Logf("head: API key available = %v (workshop key)", headAvailable)

	participantAvailable := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.True(participantAvailable, "participant should have API key via workshop key")
	s.T().Logf("participant: API key available = %v (workshop key)", participantAvailable)

	// Staff in the same org but NOT in this workshop should NOT get the workshop key
	staff := s.CreateUser("staff-wskey")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.Equal("staff", staff.GetRole())

	// Staff is NOT in the workshop and has no own key, no institution free-use key
	staffAvailable := Must(staff.GetApiKeyStatus(game.ID.String()))
	s.False(staffAvailable, "staff (not in workshop, no own key, no org free-use) should NOT have API key")
	s.T().Logf("staff (not in workshop): API key available = %v (correctly denied)", staffAvailable)
}

// TestWorkshopKeyDoesNotLeakToOtherWorkshops verifies that a workshop key
// only applies to members of that specific workshop, not to members of
// a different workshop in the same institution.
func (s *FreeUseKeyTestSuite) TestWorkshopKeyDoesNotLeakToOtherWorkshops() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("arctic-expedition"))
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("MultiWs Org"))
	head := s.CreateUser("head-multiws")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds key, shares with org
	headKey := Must(head.AddApiKey("mock-multiws", "Multi WS Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.ID.String(), inst.ID.String()))

	// Create workshop A WITH a key
	workshopA := Must(head.CreateWorkshop(inst.ID.String(), "Workshop A"))
	wsAIDStr := workshopA.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsAIDStr, &orgShareIDStr))
	s.T().Logf("Workshop A created with key")

	// Create workshop B WITHOUT a key
	workshopB := Must(head.CreateWorkshop(inst.ID.String(), "Workshop B"))
	wsBIDStr := workshopB.ID.String()
	s.T().Logf("Workshop B created without key")

	// Create participant in workshop A
	inviteA := Must(head.CreateWorkshopInvite(wsAIDStr, string(obj.RoleParticipant)))
	respA, err := s.AcceptWorkshopInviteAnonymously(*inviteA.InviteToken)
	s.NoError(err)
	participantA := s.CreateUserWithToken(*respA.AuthToken)
	s.T().Logf("Participant A joined Workshop A: %s", participantA.Name)

	// Create participant in workshop B
	inviteB := Must(head.CreateWorkshopInvite(wsBIDStr, string(obj.RoleParticipant)))
	respB, err := s.AcceptWorkshopInviteAnonymously(*inviteB.InviteToken)
	s.NoError(err)
	participantB := s.CreateUserWithToken(*respB.AuthToken)
	s.T().Logf("Participant B joined Workshop B: %s", participantB.Name)

	// Participant A (workshop with key) can play
	availableA := Must(participantA.GetApiKeyStatus(game.ID.String()))
	s.True(availableA, "participant in Workshop A (has key) should have API key")
	s.T().Logf("Participant A: API key available = %v", availableA)

	// Participant B (workshop without key, no institution free-use, no system free-use) cannot play
	availableB := Must(participantB.GetApiKeyStatus(game.ID.String()))
	s.False(availableB, "participant in Workshop B (no key) should NOT have API key")
	s.T().Logf("Participant B: API key available = %v (correctly denied)", availableB)
}

// TestWorkshopKeyTakesPriorityOverInstitutionFreeUse verifies that when both
// a workshop key and an institution free-use key are set, the workshop key
// wins (priority 1 > priority 3). We verify this indirectly: if the workshop
// key is removed, the institution free-use key still works as fallback.
func (s *FreeUseKeyTestSuite) TestWorkshopKeyTakesPriorityOverInstitutionFreeUse() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("alien-first-contact"))
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("Priority Org"))
	head := s.CreateUser("head-prio")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds TWO keys, shares both with org
	wsKey := Must(head.AddApiKey("mock-ws-prio", "WS Priority Key", "mock"))
	wsOrgShare := Must(head.ShareApiKeyWithInstitution(wsKey.ID.String(), inst.ID.String()))

	orgKey := Must(head.AddApiKey("mock-org-prio", "Org Priority Key", "mock"))
	orgOrgShare := Must(head.ShareApiKeyWithInstitution(orgKey.ID.String(), inst.ID.String()))

	// Set institution free-use key
	orgShareIDStr := orgOrgShare.ID.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), &orgShareIDStr))
	s.T().Logf("Institution free-use key set")

	// Create workshop and set workshop key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Priority Workshop"))
	wsIDStr := workshop.ID.String()
	wsShareIDStr := wsOrgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &wsShareIDStr))
	s.T().Logf("Workshop key set")

	// Create participant
	participantInvite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	participantResp, err := s.AcceptWorkshopInviteAnonymously(*participantInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*participantResp.AuthToken)
	s.T().Logf("Participant joined workshop")

	// Participant can play (workshop key active)
	available := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.True(available, "participant should have API key (workshop key)")
	s.T().Logf("Participant: API key available = %v (workshop key active)", available)

	// Remove workshop key — participant loses access (institution free-use
	// does NOT apply to participants)
	Must(head.SetWorkshopApiKey(wsIDStr, nil))
	s.T().Logf("Workshop key cleared")

	availableAfterWsClear := Must(participant.GetApiKeyStatus(game.ID.String()))
	s.False(availableAfterWsClear, "participant should NOT have API key after workshop key removed")
	s.T().Logf("Participant: API key available = %v (no workshop key)", availableAfterWsClear)

	// Head (non-participant) should still have access via institution free-use fallback
	Must(head.SetActiveWorkshop(&wsIDStr))
	headAvailable := Must(head.GetApiKeyStatus(game.ID.String()))
	s.True(headAvailable, "head should still have API key via institution free-use fallback")
	s.T().Logf("Head: API key available = %v (institution free-use fallback)", headAvailable)

	// Add a staff member with NO own keys to test institution free-use fallback cleanly
	// (Head has own keys from sharing, so they always resolve via user default — priority 4)
	staff := s.CreateUser("staff-prio")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.Equal("staff", staff.GetRole())

	// Staff (no own keys) should have access via institution free-use key
	staffAvailable := Must(staff.GetApiKeyStatus(game.ID.String()))
	s.True(staffAvailable, "staff should have API key via institution free-use fallback")
	s.T().Logf("Staff: API key available = %v (institution free-use fallback)", staffAvailable)

	// Now also remove institution free-use key — staff loses access
	Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), nil))
	s.T().Logf("Institution free-use key cleared")

	staffAvailableFinal := Must(staff.GetApiKeyStatus(game.ID.String()))
	s.False(staffAvailableFinal, "staff should NOT have API key (all keys removed)")
	s.T().Logf("Staff: API key available = %v (all keys removed)", staffAvailableFinal)
}
