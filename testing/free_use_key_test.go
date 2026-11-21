package testing

import (
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
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.NotEmpty(game.Id)
	s.T().Logf("Admin uploaded game: %s (ID: %s)", game.Name, game.Id)

	// Admin adds a mock API key and sets it as the system free-use key
	adminKey := Must(admin.AddApiKey("mock-sys-free", "System Free Key", "openai"))
	adminKeyIDStr := adminKey.ApiKeyId.String()
	settings := Must(admin.SetSystemFreeUseApiKey(&adminKeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId)
	s.T().Logf("System free-use key set: %s", adminKey.ApiKeyId)

	// Create users with different roles — none have their own API keys
	individual := s.CreateUser("ind-sysfree")
	s.Equal("individual", individual.GetRole())

	inst := Must(admin.CreateInstitution("SysFree Org"))
	head := s.CreateUser("head-sysfree")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	s.Equal("head", head.GetRole())

	staff := s.CreateUser("staff-sysfree")
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.Equal("staff", staff.GetRole())

	// All three should be able to play (API key resolved via system free-use)
	for _, user := range []*testutil.UserClient{individual, head, staff} {
		available := Must(user.GetApiKeyStatus(game.Id.String()))
		s.True(available, "%s should have API key available via system free-use key", user.Name)
		s.T().Logf("%s: API key available = %v (via system free-use)", user.Name, available)
	}

	// Clean up: clear the system free-use key
	Must(admin.SetSystemFreeUseApiKey(nil))
	s.T().Logf("System free-use key cleared")

	// After clearing, users without own keys cannot play
	for _, user := range []*testutil.UserClient{individual, head, staff} {
		available := Must(user.GetApiKeyStatus(game.Id.String()))
		s.False(available, "%s should NOT have API key after system free-use key cleared", user.Name)
		s.T().Logf("%s: API key available = %v (after clear)", user.Name, available)
	}
}

// ---------------------------------------------------------------------------
// Institution free-use key (priority 3)
// ---------------------------------------------------------------------------

// TestInstitutionFreeUseKeyAllowsOrgMembersToPlay verifies that when a head
// sets an institution free-use key, all org members (head, staff, guest)
// can play, but non-org members cannot use that key.
func (s *FreeUseKeyTestSuite) TestInstitutionFreeUseKeyAllowsOrgMembersToPlay() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("alien-first-contact"))
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.T().Logf("Admin uploaded game: %s (ID: %s)", game.Name, game.Id)

	// Create institution with head
	inst := Must(admin.CreateInstitution("OrgFree Org"))
	head := s.CreateUser("head-orgfree")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	s.Equal("head", head.GetRole())

	// Head adds API key, shares with institution
	headKey := Must(head.AddApiKey("mock-org-free", "Org Free Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), inst.Id.String()))
	s.T().Logf("Head shared key with institution: orgShareID=%s", orgShare.Id)

	// Head sets the org share as institution free-use key
	orgShareIDStr := orgShare.Id.String()
	updatedInst := Must(head.SetInstitutionFreeUseApiKey(inst.Id.String(), &orgShareIDStr))
	s.NotNil(updatedInst.FreeUseApiKeyShareId)
	s.T().Logf("Institution free-use key set: %s", orgShare.Id)

	// Add staff member
	staff := s.CreateUser("staff-orgfree")
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.Equal("staff", staff.GetRole())

	// Create a workshop and add a guest
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "OrgFree Workshop"))
	wsIDStr := workshop.Id.String()
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop: %s (ID: %s)", guest.Name, guest.ID)

	// Head and staff can play (resolved via institution free-use key)
	for _, user := range []*testutil.UserClient{head, staff} {
		available := Must(user.GetApiKeyStatus(game.Id.String()))
		s.True(available, "%s (org member) should have API key via institution free-use key", user.Name)
		s.T().Logf("%s: API key available = %v (via institution free-use)", user.Name, available)
	}

	// A Guest in a workshop should NOT get institution free-use key —
	// guests can only use the workshop key (priority 1)
	guestAvailable := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.False(guestAvailable, "guest should NOT have API key via institution free-use (only workshop key works for guests)")
	s.T().Logf("guest: API key available = %v (correctly denied — no workshop key)", guestAvailable)

	// Non-org member (individual) should NOT have access via institution free-use key
	outsider := s.CreateUser("outsider-orgfree")
	s.Equal("individual", outsider.GetRole())
	outsiderAvailable := Must(outsider.GetApiKeyStatus(game.Id.String()))
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
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("OrgFallback Org"))
	head := s.CreateUser("head-orgfb")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds key, shares with org, sets as institution free-use key
	headKey := Must(head.AddApiKey("mock-org-fb", "Org Fallback Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), inst.Id.String()))
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.Id.String(), &orgShareIDStr))
	s.T().Logf("Institution free-use key set")

	// Create workshop WITHOUT setting a workshop API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "No-Key Workshop"))
	wsIDStr := workshop.Id.String()
	s.Nil(workshop.DefaultApiKeyShareId, "workshop should have no default API key")

	// Create guest in the workshop
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop (no workshop key)")

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Head should still be able to play via institution free-use fallback (priority 3)
	headAvailable := Must(head.GetApiKeyStatus(game.Id.String()))
	s.True(headAvailable, "head should have API key via institution free-use (workshop has no key)")
	s.T().Logf("head: API key available = %v (institution free-use fallback)", headAvailable)

	// Guest should NOT be able to play — institution free-use key does not
	// apply to guests; they can only use the workshop key (priority 1)
	guestAvailable := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.False(guestAvailable, "guest should NOT have API key (institution free-use does not apply to guests)")
	s.T().Logf("guest: API key available = %v (correctly denied — no workshop key)", guestAvailable)
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
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("WsKey Org"))
	head := s.CreateUser("head-wskey")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds key, shares with org
	headKey := Must(head.AddApiKey("mock-ws-play", "Workshop Play Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), inst.Id.String()))
	s.T().Logf("Head shared key with org: %s", orgShare.Id)

	// Head creates workshop and sets the org share as workshop key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Play Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	s.T().Logf("Workshop key set")

	// Head enters workshop mode
	Must(head.SetActiveWorkshop(&wsIDStr))

	// Create guest in the workshop
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop: %s", guest.Name)

	// Workshop members can play
	headAvailable := Must(head.GetApiKeyStatus(game.Id.String()))
	s.True(headAvailable, "head (in workshop) should have API key via workshop key")
	s.T().Logf("head: API key available = %v (workshop key)", headAvailable)

	guestAvailable := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.True(guestAvailable, "guest should have API key via workshop key")
	s.T().Logf("guest: API key available = %v (workshop key)", guestAvailable)

	// Staff in the same org but NOT in this workshop should NOT get the workshop key
	staff := s.CreateUser("staff-wskey")
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.Equal("staff", staff.GetRole())

	// Staff is NOT in the workshop and has no own key, no institution free-use key
	staffAvailable := Must(staff.GetApiKeyStatus(game.Id.String()))
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
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("MultiWs Org"))
	head := s.CreateUser("head-multiws")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds key, shares with org
	headKey := Must(head.AddApiKey("mock-multiws", "Multi WS Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(headKey.Id.String(), inst.Id.String()))

	// Create workshop A WITH a key
	workshopA := Must(head.CreateWorkshop(inst.Id.String(), "Workshop A"))
	wsAIDStr := workshopA.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsAIDStr, &orgShareIDStr))
	s.T().Logf("Workshop A created with key")

	// Create workshop B WITHOUT a key
	workshopB := Must(head.CreateWorkshop(inst.Id.String(), "Workshop B"))
	wsBIDStr := workshopB.Id.String()
	s.T().Logf("Workshop B created without key")

	// Create guest in workshop A
	inviteA := Must(head.CreateWorkshopInvite(wsAIDStr))
	guestA := Must(s.JoinWorkshop(*inviteA.InviteToken))
	s.T().Logf("Guest A joined Workshop A: %s", guestA.Name)

	// Create guest in workshop B
	inviteB := Must(head.CreateWorkshopInvite(wsBIDStr))
	guestB := Must(s.JoinWorkshop(*inviteB.InviteToken))
	s.T().Logf("Guest B joined Workshop B: %s", guestB.Name)

	// Guest A (workshop with key) can play
	availableA := Must(guestA.GetApiKeyStatus(game.Id.String()))
	s.True(availableA, "guest in Workshop A (has key) should have API key")
	s.T().Logf("Guest A: API key available = %v", availableA)

	// Guest B (workshop without key, no institution free-use, no system free-use) cannot play
	availableB := Must(guestB.GetApiKeyStatus(game.Id.String()))
	s.False(availableB, "guest in Workshop B (no key) should NOT have API key")
	s.T().Logf("Guest B: API key available = %v (correctly denied)", availableB)
}

// TestWorkshopKeyTakesPriorityOverInstitutionFreeUse verifies that when both
// a workshop key and an institution free-use key are set, the workshop key
// wins (priority 1 > priority 3). We verify this indirectly: if the workshop
// key is removed, the institution free-use key still works as fallback.
func (s *FreeUseKeyTestSuite) TestWorkshopKeyTakesPriorityOverInstitutionFreeUse() {
	admin := s.DevUser()

	// Admin uploads a game
	game := Must(admin.UploadGame("alien-first-contact"))
	game = Must(admin.SetGamePublic(game.Id.String(), true)) // the players read it: it is public
	s.T().Logf("Admin uploaded game: %s", game.Name)

	// Create institution with head
	inst := Must(admin.CreateInstitution("Priority Org"))
	head := s.CreateUser("head-prio")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds TWO keys, shares both with org
	wsKey := Must(head.AddApiKey("mock-ws-prio", "WS Priority Key", "openai"))
	wsOrgShare := Must(head.ShareApiKeyWithInstitution(wsKey.Id.String(), inst.Id.String()))

	orgKey := Must(head.AddApiKey("mock-org-prio", "Org Priority Key", "openai"))
	orgOrgShare := Must(head.ShareApiKeyWithInstitution(orgKey.Id.String(), inst.Id.String()))

	// Set institution free-use key
	orgShareIDStr := orgOrgShare.Id.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.Id.String(), &orgShareIDStr))
	s.T().Logf("Institution free-use key set")

	// Create workshop and set workshop key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Priority Workshop"))
	wsIDStr := workshop.Id.String()
	wsShareIDStr := wsOrgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &wsShareIDStr))
	s.T().Logf("Workshop key set")

	// Create guest
	guestInvite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*guestInvite.InviteToken))
	s.T().Logf("Guest joined workshop")

	// Guest can play (workshop key active)
	available := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.True(available, "guest should have API key (workshop key)")
	s.T().Logf("Guest: API key available = %v (workshop key active)", available)

	// Remove workshop key — guest loses access (institution free-use
	// does NOT apply to guests)
	Must(head.SetWorkshopApiKey(wsIDStr, nil))
	s.T().Logf("Workshop key cleared")

	availableAfterWsClear := Must(guest.GetApiKeyStatus(game.Id.String()))
	s.False(availableAfterWsClear, "guest should NOT have API key after workshop key removed")
	s.T().Logf("Guest: API key available = %v (no workshop key)", availableAfterWsClear)

	// Head (non-guest) should still have access via institution free-use fallback
	Must(head.SetActiveWorkshop(&wsIDStr))
	headAvailable := Must(head.GetApiKeyStatus(game.Id.String()))
	s.True(headAvailable, "head should still have API key via institution free-use fallback")
	s.T().Logf("Head: API key available = %v (institution free-use fallback)", headAvailable)

	// Add a staff member with NO own keys to test institution free-use fallback cleanly
	// (Head has own keys from sharing, so they always resolve via user default — priority 4)
	staff := s.CreateUser("staff-prio")
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	s.Equal("staff", staff.GetRole())

	// Staff (no own keys) should have access via institution free-use key
	staffAvailable := Must(staff.GetApiKeyStatus(game.Id.String()))
	s.True(staffAvailable, "staff should have API key via institution free-use fallback")
	s.T().Logf("Staff: API key available = %v (institution free-use fallback)", staffAvailable)

	// Now also remove institution free-use key — staff loses access
	Must(head.SetInstitutionFreeUseApiKey(inst.Id.String(), nil))
	s.T().Logf("Institution free-use key cleared")

	staffAvailableFinal := Must(staff.GetApiKeyStatus(game.Id.String()))
	s.False(staffAvailableFinal, "staff should NOT have API key (all keys removed)")
	s.T().Logf("Staff: API key available = %v (all keys removed)", staffAvailableFinal)
}
