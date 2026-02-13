package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ApiKeyCascadeOrgShareTestSuite tests that deleting an API key properly cleans up
// all institution-level references (org shares, free-use key).
type ApiKeyCascadeOrgShareTestSuite struct {
	testutil.BaseSuite
}

func TestApiKeyCascadeOrgShareSuite(t *testing.T) {
	s := &ApiKeyCascadeOrgShareTestSuite{}
	s.SuiteName = "API Key Cascade Org Share Tests"
	suite.Run(t, s)
}

// setupHeadWithOrgAndKey creates a user who is head of an institution, adds an API key,
// and shares it with the institution.
// Returns (user, institutionID, gameID, selfShareID, orgShareID).
func (s *ApiKeyCascadeOrgShareTestSuite) setupHeadWithOrgAndKey(prefix string) (*testutil.UserClient, string, string, string) {
	admin := s.DevUser()
	user := s.CreateUser(prefix)

	// Create institution and make user the head
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	instIDStr := inst.ID.String()
	invite := Must(admin.InviteToInstitution(instIDStr, "head", user.ID))
	Must(user.AcceptInvite(invite.ID.String()))
	s.Equal("head", user.GetRole())

	// Add API key
	keyShare := Must(user.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
	selfShareID := keyShare.ID.String()

	// Share key with institution
	orgShare := Must(user.ShareApiKeyWithInstitution(selfShareID, instIDStr))
	s.NotEmpty(orgShare.ID)

	return user, instIDStr, selfShareID, orgShare.ID.String()
}

// TestDeleteKeyRemovesOrgShare verifies that cascade-deleting an API key removes
// the institution share. After re-creating a new key, the org should have no shares.
func (s *ApiKeyCascadeOrgShareTestSuite) TestDeleteKeyRemovesOrgShare() {
	user, instIDStr, selfShareID, _ := s.setupHeadWithOrgAndKey("cascade-org")

	// Verify org has the shared key
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 API key")
	orgShareCount := 0
	for _, sh := range keys.Shares {
		if sh.Institution != nil && sh.Institution.ID.String() == instIDStr {
			orgShareCount++
		}
	}
	s.Equal(1, orgShareCount, "should have 1 org share before deletion")
	s.T().Logf("Org has 1 share before deletion")

	// Delete the API key (cascade)
	MustSucceed(user.DeleteApiKey(selfShareID, true))
	s.T().Logf("Deleted API key (cascade)")

	// Verify no keys and no org shares remain
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 0, "should have no API keys after deletion")
	s.Len(keys.Shares, 0, "should have no shares after deletion")
	s.T().Logf("No keys or shares remain after deletion")

	// Re-create a new key
	newKey := Must(user.AddApiKey("mock-cascade-org-new", "New Key", "mock"))
	s.NotEmpty(newKey.ID)
	s.T().Logf("Created new key: shareID=%s", newKey.ID)

	// Verify the new key does NOT have an org share
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 API key after re-creation")
	orgShareCount = 0
	for _, sh := range keys.Shares {
		if sh.Institution != nil && sh.Institution.ID.String() == instIDStr {
			orgShareCount++
		}
	}
	s.Equal(0, orgShareCount, "new key should NOT have an org share — old shares must be fully cleaned up")
	s.T().Logf("New key has no org share — cleanup was correct")
}

// TestDeleteKeyWithFreeUseKeyClearsInstitutionRef verifies that cascade-deleting
// an API key that is set as the institution's free-use key clears the institution reference.
func (s *ApiKeyCascadeOrgShareTestSuite) TestDeleteKeyWithFreeUseKeyClearsInstitutionRef() {
	user, instIDStr, selfShareID, orgShareID := s.setupHeadWithOrgAndKey("cascade-freeuse")

	// Set the org share as institution free-use key
	Must(user.SetInstitutionFreeUseApiKey(instIDStr, &orgShareID))
	s.T().Logf("Set org share as institution free-use key")

	// Verify institution has free-use key set
	inst := Must(user.GetInstitution(instIDStr))
	s.NotNil(inst.FreeUseApiKeyShareID, "institution should have free-use key set")
	s.T().Logf("Institution free-use key: %s", *inst.FreeUseApiKeyShareID)

	// Delete the API key (cascade) — should clear institution free-use key reference
	MustSucceed(user.DeleteApiKey(selfShareID, true))
	s.T().Logf("Deleted API key (cascade)")

	// Verify institution free-use key is cleared
	inst = Must(user.GetInstitution(instIDStr))
	s.Nil(inst.FreeUseApiKeyShareID, "institution free-use key should be cleared after key deletion")
	s.T().Logf("Institution free-use key correctly cleared")

	// Verify no keys remain
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 0, "should have no API keys")
	s.Len(keys.Shares, 0, "should have no shares")
	s.T().Logf("All keys and shares cleaned up")
}

// TestDeleteKeyWithFreeUseKeyThenRecreate verifies the full cycle:
// create key → share with org → set as free-use → delete key → re-create key →
// verify org has no shares and free-use key is cleared.
func (s *ApiKeyCascadeOrgShareTestSuite) TestDeleteKeyWithFreeUseKeyThenRecreate() {
	user, instIDStr, selfShareID, orgShareID := s.setupHeadWithOrgAndKey("cascade-recreate")

	// Set the org share as institution free-use key
	Must(user.SetInstitutionFreeUseApiKey(instIDStr, &orgShareID))

	// Delete the API key (cascade)
	MustSucceed(user.DeleteApiKey(selfShareID, true))
	s.T().Logf("Deleted API key with free-use reference")

	// Re-create a new key and share with org
	newKey := Must(user.AddApiKey("mock-cascade-recreate-new", "New Key", "mock"))
	newShareID := newKey.ID.String()
	s.T().Logf("Created new key: shareID=%s", newShareID)

	// Verify institution free-use key is cleared (not pointing to old share)
	inst := Must(user.GetInstitution(instIDStr))
	s.Nil(inst.FreeUseApiKeyShareID, "institution free-use key should be nil after old key deletion")
	s.T().Logf("Institution free-use key is nil — correct")

	// Verify new key has no org share (only self-share)
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 API key")
	orgShareCount := 0
	for _, sh := range keys.Shares {
		if sh.Institution != nil && sh.Institution.ID.String() == instIDStr {
			orgShareCount++
		}
	}
	s.Equal(0, orgShareCount, "new key should not inherit old org shares")
	s.T().Logf("New key correctly has no org shares")
}
