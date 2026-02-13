package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ApiKeyTestSuite tests API key management: defaults, sharing, deletion
type ApiKeyTestSuite struct {
	testutil.BaseSuite
}

func TestApiKeySuite(t *testing.T) {
	s := &ApiKeyTestSuite{}
	s.SuiteName = "API Key Tests"
	suite.Run(t, s)
}

// TestDefaultApiKeySwitching tests that a user can add API keys and switch the default.
func (s *ApiKeyTestSuite) TestDefaultApiKeySwitching() {
	user := s.CreateUser("key-switcher")

	// Add first key — should become default automatically
	key1 := Must(user.AddApiKey("mock-key-1", "First Key", "mock"))
	s.NotEmpty(key1.ID)
	s.NotEmpty(key1.ApiKeyID)
	s.T().Logf("Added first key: shareID=%s", key1.ID)

	// Verify it's the default
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1)
	s.True(keys.ApiKeys[0].IsDefault, "first key should be default")
	s.T().Logf("First key is default")

	// Add second key — should NOT be default
	key2 := Must(user.AddApiKey("mock-key-2", "Second Key", "mock"))
	s.NotEmpty(key2.ID)
	s.T().Logf("Added second key: shareID=%s", key2.ID)

	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 2)
	for _, k := range keys.ApiKeys {
		if k.ID == key1.ApiKeyID {
			s.True(k.IsDefault, "first key should still be default")
		} else {
			s.False(k.IsDefault, "second key should not be default")
		}
	}
	s.T().Logf("Second key is not default")

	// Switch default to second key
	Must(user.SetDefaultApiKey(key2.ID.String()))
	s.T().Logf("Switched default to second key")

	// Verify via GetApiKeys
	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key2.ApiKeyID {
			s.True(k.IsDefault, "second key should now be default")
		} else {
			s.False(k.IsDefault, "first key should no longer be default")
		}
	}
	s.T().Logf("Default correctly switched to second key")

	// Switch back to first key
	Must(user.SetDefaultApiKey(key1.ID.String()))
	s.T().Logf("Switched default back to first key")

	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key1.ApiKeyID {
			s.True(k.IsDefault, "first key should be default again")
		} else {
			s.False(k.IsDefault, "second key should not be default")
		}
	}
	s.T().Logf("Default correctly switched back to first key")
}

// TestMistralKeyLifecycle tests adding a Mistral key, setting it as default, and deleting it.
// Reproduces "share not found" bug when setting a non-mock key as default.
// This test was added, because Mistral keys didn't work (error - share not found) while openai keys did
func (s *ApiKeyTestSuite) TestMistralKeyLifecycle() {
	user := s.CreateUser("mistral-user")

	// Add a Mistral key
	key := Must(user.AddApiKey("test-mistral-key-123", "My Mistral", "mistral"))
	s.NotEmpty(key.ID, "share ID should not be empty")
	s.NotEmpty(key.ApiKeyID, "api key ID should not be empty")
	s.T().Logf("Added Mistral key: shareID=%s, apiKeyID=%s", key.ID, key.ApiKeyID)

	// Verify it's listed and is the default (first key)
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 key")
	s.Equal("mistral", keys.ApiKeys[0].Platform, "platform should be mistral")
	s.True(keys.ApiKeys[0].IsDefault, "first key should be default")
	s.T().Logf("Mistral key is default")

	// Add a second key (mock) so we can switch defaults
	key2 := Must(user.AddApiKey("test-mock-key-456", "My Mock", "mock"))
	s.T().Logf("Added Mock key: shareID=%s", key2.ID)

	// Switch default to mock key
	Must(user.SetDefaultApiKey(key2.ID.String()))
	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key2.ApiKeyID {
			s.True(k.IsDefault, "mock key should be default")
		} else {
			s.False(k.IsDefault, "mistral key should not be default")
		}
	}
	s.T().Logf("Switched default to mock key")

	// Switch default back to Mistral key — this is where "share not found" was reported
	Must(user.SetDefaultApiKey(key.ID.String()))
	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key.ApiKeyID {
			s.True(k.IsDefault, "mistral key should be default again")
		} else {
			s.False(k.IsDefault, "mock key should not be default")
		}
	}
	s.T().Logf("Switched default back to Mistral key")

	// Delete Mistral key
	MustSucceed(user.DeleteApiKey(key.ID.String(), true))
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 key remaining")
	s.Equal(key2.ApiKeyID, keys.ApiKeys[0].ID, "remaining key should be mock")
	s.T().Logf("Deleted Mistral key, mock key remains")

	// Delete mock key
	MustSucceed(user.DeleteApiKey(key2.ID.String(), true))
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 0, "should have no keys remaining")
	s.T().Logf("All keys removed")
}

// TestApiKeyTrimWhitespace verifies that leading and trailing whitespace is trimmed
// from the API key value, name, and platform when creating a key.
func (s *ApiKeyTestSuite) TestApiKeyTrimWhitespace() {
	user := s.CreateUser("trim-ws")

	// Create a key with whitespace in the key value and name
	key := Must(user.AddApiKey("  mock-trim-key  ", "  Trimmed Key  ", "  mock  "))
	s.NotEmpty(key.ID)
	s.T().Logf("Created key with whitespace: shareID=%s", key.ID)

	// Verify the key was created successfully (platform was trimmed and recognized)
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 key")

	apiKey := keys.ApiKeys[0]

	// Name should be trimmed
	s.Equal("Trimmed Key", apiKey.Name, "key name should be trimmed")

	// Platform should be trimmed
	s.Equal("mock", apiKey.Platform, "platform should be trimmed")

	// KeyShortened is the last 6 chars of the stored key — if trimmed correctly,
	// "mock-trim-key" → shortened "im-key" (no trailing spaces)
	s.Equal("im-key", apiKey.KeyShortened, "key value should be trimmed (no trailing whitespace in shortened)")
	s.T().Logf("Key stored correctly: name=%q, platform=%q, shortened=%q", apiKey.Name, apiKey.Platform, apiKey.KeyShortened)

	// Clean up
	MustSucceed(user.DeleteApiKey(key.ID.String(), true))
}

// TestApiKeyTrimWhitespaceOnlyFails verifies that whitespace-only values are rejected.
func (s *ApiKeyTestSuite) TestApiKeyTrimWhitespaceOnlyFails() {
	user := s.CreateUser("trim-empty")

	// Key with whitespace-only value should fail
	_, err := user.AddApiKey("   ", "Some Name", "mock")
	s.Error(err, "whitespace-only key value should be rejected")
	s.T().Logf("Whitespace-only key correctly rejected: %v", err)

	// Key with whitespace-only platform should fail
	_, err = user.AddApiKey("mock-key-123", "Some Name", "   ")
	s.Error(err, "whitespace-only platform should be rejected")
	s.T().Logf("Whitespace-only platform correctly rejected: %v", err)
}

// TestApiKeyShareWithOrgAndCleanup tests the full lifecycle: add keys, join org,
// share with org, switch defaults, remove shared key, remove all keys.
func (s *ApiKeyTestSuite) TestApiKeyShareWithOrgAndCleanup() {
	admin := s.DevUser()
	user := s.CreateUser("org-key-user")

	// User adds two API keys, first becomes default
	key1 := Must(user.AddApiKey("mock-org-key-1", "Personal Key 1", "mock"))
	key2 := Must(user.AddApiKey("mock-org-key-2", "Personal Key 2", "mock"))
	s.T().Logf("User added two keys: key1=%s, key2=%s", key1.ID, key2.ID)

	// Verify key1 is default
	keys := Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 2)
	for _, k := range keys.ApiKeys {
		if k.ID == key1.ApiKeyID {
			s.True(k.IsDefault, "first key should be default")
		}
	}

	// Admin creates institution and invites user as head
	inst := Must(admin.CreateInstitution("Key Test Org"))
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", user.ID))
	Must(user.AcceptInvite(invite.ID.String()))
	s.Equal("head", user.GetRole())
	s.T().Logf("User became head of institution")

	// User shares key1 with the institution
	orgShare := Must(user.ShareApiKeyWithInstitution(key1.ID.String(), inst.ID.String()))
	s.NotEmpty(orgShare.ID)
	s.T().Logf("Shared key1 with institution: orgShareID=%s", orgShare.ID)

	// Verify shares exist
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 2, "should still have 2 API keys")
	s.True(len(keys.Shares) >= 3, "should have at least 3 shares (2 self + 1 org)")
	s.T().Logf("User has %d keys and %d shares", len(keys.ApiKeys), len(keys.Shares))

	// Switch default to key2
	Must(user.SetDefaultApiKey(key2.ID.String()))
	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key2.ApiKeyID {
			s.True(k.IsDefault, "key2 should be default after switch")
		} else {
			s.False(k.IsDefault, "key1 should not be default")
		}
	}
	s.T().Logf("Switched default to key2")

	// Switch default back to key1
	Must(user.SetDefaultApiKey(key1.ID.String()))
	keys = Must(user.GetApiKeys())
	for _, k := range keys.ApiKeys {
		if k.ID == key1.ApiKeyID {
			s.True(k.IsDefault, "key1 should be default again")
		} else {
			s.False(k.IsDefault, "key2 should not be default")
		}
	}
	s.T().Logf("Switched default back to key1")

	// Delete key1 (which is shared with org) — cascade should remove key + all shares
	MustSucceed(user.DeleteApiKey(key1.ID.String(), true))
	s.T().Logf("Deleted key1 (was shared with org)")

	// Verify key1 is gone, only key2 remains
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 1, "should have 1 key remaining")
	s.Equal(key2.ApiKeyID, keys.ApiKeys[0].ID, "remaining key should be key2")
	s.T().Logf("Key1 removed, key2 remains")

	// Delete key2 — user should have no keys left
	MustSucceed(user.DeleteApiKey(key2.ID.String(), true))
	keys = Must(user.GetApiKeys())
	s.Len(keys.ApiKeys, 0, "should have no keys remaining")
	s.T().Logf("All keys removed")
}
