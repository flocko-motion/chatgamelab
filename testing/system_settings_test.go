package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// SystemSettingsTestSuite tests admin system settings management
type SystemSettingsTestSuite struct {
	testutil.BaseSuite
}

func TestSystemSettingsSuite(t *testing.T) {
	s := &SystemSettingsTestSuite{}
	s.SuiteName = "System Settings Tests"
	suite.Run(t, s)
}

// TestAdminsCanSetAndReplaceFreeUseApiKey tests that multiple admins can
// set, replace, and clear the system free-use API key, and that non-admins are denied.
func (s *SystemSettingsTestSuite) TestAdminsCanSetAndReplaceFreeUseApiKey() {
	// Use preseeded admin as admin1
	admin1 := s.DevUser()

	// Create a second admin
	admin2 := s.CreateUser("admin2")
	s.Role(admin2, "admin")
	s.Equal("admin", admin2.GetRole())
	s.T().Logf("Created admin2 with admin role")

	// Both admins add a mock API key
	admin1Key := Must(admin1.AddApiKey("mock-key-admin1", "Admin1 Mock Key", "mock"))
	s.NotEmpty(admin1Key.ID)
	s.NotEmpty(admin1Key.ApiKeyID)
	s.T().Logf("Admin1 added API key: shareID=%s, apiKeyID=%s", admin1Key.ID, admin1Key.ApiKeyID)

	admin2Key := Must(admin2.AddApiKey("mock-key-admin2", "Admin2 Mock Key", "mock"))
	s.NotEmpty(admin2Key.ID)
	s.NotEmpty(admin2Key.ApiKeyID)
	s.T().Logf("Admin2 added API key: shareID=%s, apiKeyID=%s", admin2Key.ID, admin2Key.ApiKeyID)

	// Verify initial state: no free-use key set
	settings := Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyID, "initially no free-use key should be set")
	s.T().Logf("Initial settings: no free-use key")

	// Admin1 sets his key as the system free-use key
	admin1KeyIDStr := admin1Key.ApiKeyID.String()
	settings = Must(admin1.SetSystemFreeUseApiKey(&admin1KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID, "free-use key should be set")
	s.Equal(admin1Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.Equal("Admin1 Mock Key", settings.FreeUseApiKeyName)
	s.Equal("mock", settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin1 set free-use key to his own key")

	// Admin2 can see the free-use key is set (admins see full details)
	settings = Must(admin2.GetSystemSettings())
	s.NotNil(settings.FreeUseApiKeyID, "admin2 should see free-use key")
	s.Equal(admin1Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.T().Logf("Admin2 can see admin1's free-use key in settings")

	// Admin2 replaces it with his own key
	admin2KeyIDStr := admin2Key.ApiKeyID.String()
	settings = Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID, "free-use key should be set to admin2's key")
	s.Equal(admin2Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.Equal("Admin2 Mock Key", settings.FreeUseApiKeyName)
	s.Equal("mock", settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin2 replaced free-use key with his own key")

	// Admin1 sees admin2's key is now the free-use key
	settings = Must(admin1.GetSystemSettings())
	s.NotNil(settings.FreeUseApiKeyID)
	s.Equal(admin2Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.Equal("Admin2 Mock Key", settings.FreeUseApiKeyName)
	s.T().Logf("Admin1 sees admin2's key is now the free-use key")

	// Admin2 clears the free-use key
	settings = Must(admin2.SetSystemFreeUseApiKey(nil))
	s.Nil(settings.FreeUseApiKeyID, "free-use key should be cleared")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin2 cleared the free-use key")

	// Non-admin cannot set free-use key
	regularUser := s.CreateUser("regular-user")
	s.Equal("individual", regularUser.GetRole())
	Fail(regularUser.SetSystemFreeUseApiKey(&admin1KeyIDStr))
	s.T().Logf("Non-admin correctly denied setting free-use key")
}

// TestDeleteApiKeyWhileSetAsFreeUse tests that an admin can always delete
// their own API key even when it is currently set as the system free-use key.
// The system settings should be automatically cleaned up (free-use key cleared).
func (s *SystemSettingsTestSuite) TestDeleteApiKeyWhileSetAsFreeUse() {
	admin1 := s.DevUser()

	// Create a second admin
	admin2 := s.CreateUser("admin2-delete-test")
	s.Role(admin2, "admin")

	// Admin1 adds a key and sets it as free-use
	admin1Key := Must(admin1.AddApiKey("mock-key-delete-test", "Key To Delete", "mock"))
	admin1KeyIDStr := admin1Key.ApiKeyID.String()
	settings := Must(admin1.SetSystemFreeUseApiKey(&admin1KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID)
	s.Equal(admin1Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.T().Logf("Admin1 set free-use key: %s", admin1Key.ApiKeyID)

	// Admin1 deletes the key (cascade) — should succeed even though it's the free-use key
	MustSucceed(admin1.DeleteApiKey(admin1Key.ID.String(), true))
	s.T().Logf("Admin1 deleted API key that was set as free-use key")

	// Verify system settings were cleaned up: free-use key should be nil
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyID, "free-use key should be cleared after key deletion")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("System settings correctly cleared after key deletion")

	// Admin2 also sees the free-use key is cleared
	settings = Must(admin2.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyID, "admin2 should also see free-use key cleared")
	s.T().Logf("Admin2 confirms free-use key is cleared")

	// Now test the same with admin2's key
	admin2Key := Must(admin2.AddApiKey("mock-key-admin2-delete", "Admin2 Key To Delete", "mock"))
	admin2KeyIDStr := admin2Key.ApiKeyID.String()
	settings = Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID)
	s.Equal(admin2Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.T().Logf("Admin2 set free-use key: %s", admin2Key.ApiKeyID)

	// Admin2 deletes the key — should succeed and clean up settings
	MustSucceed(admin2.DeleteApiKey(admin2Key.ID.String(), true))
	s.T().Logf("Admin2 deleted API key that was set as free-use key")

	// Verify cleanup
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyID, "free-use key should be cleared after admin2's key deletion")
	s.T().Logf("System settings correctly cleared after admin2's key deletion")
}

// TestHeadCannotModifySystemSettings verifies that a user with the "head" role
// cannot modify system settings — only admins should have write access.
// Note: GET /api/system/settings is open to all authenticated users (sensitive
// fields are stripped for non-admins).
func (s *SystemSettingsTestSuite) TestHeadCannotModifySystemSettings() {
	admin := s.DevUser()

	// Create an institution and invite a user as head
	inst := Must(admin.CreateInstitution("test-inst-settings"))
	head := s.CreateUser("head-settings")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))
	s.Equal("head", head.GetRole())
	s.T().Logf("Created head user in institution")

	// Head cannot read system settings
	_, err := head.GetSystemSettings()
	s.Error(err, "head should not be able to read system settings")
	s.T().Logf("Head correctly denied reading system settings")

	// Head cannot set free-use key
	adminKey := Must(admin.AddApiKey("mock-key-head-test", "Admin Key", "mock"))
	adminKeyIDStr := adminKey.ApiKeyID.String()
	_, err = head.SetSystemFreeUseApiKey(&adminKeyIDStr)
	s.Error(err, "head should not be able to set free-use key")
	s.T().Logf("Head correctly denied setting free-use key")

	// Head cannot clear free-use key
	_, err = head.SetSystemFreeUseApiKey(nil)
	s.Error(err, "head should not be able to clear free-use key")
	s.T().Logf("Head correctly denied clearing free-use key")
}

// TestFreeUseKeyClearedWhenAdminRoleRemoved tests that when an admin who set
// the system free-use API key loses their admin role, the free-use key is
// automatically cleared from system settings.
func (s *SystemSettingsTestSuite) TestFreeUseKeyClearedWhenAdminRoleRemoved() {
	admin1 := s.DevUser()

	// Create a second admin who will have their role removed
	admin2 := s.CreateUser("admin2-role-removal")
	s.Role(admin2, "admin")
	s.Equal("admin", admin2.GetRole())
	s.T().Logf("Created admin2 with admin role")

	// Admin2 adds a key and sets it as the system free-use key
	admin2Key := Must(admin2.AddApiKey("mock-key-role-test", "Admin2 Role Test Key", "mock"))
	admin2KeyIDStr := admin2Key.ApiKeyID.String()
	settings := Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyID)
	s.Equal(admin2Key.ApiKeyID, *settings.FreeUseApiKeyID)
	s.T().Logf("Admin2 set free-use key: %s", admin2Key.ApiKeyID)

	// Admin1 removes admin2's admin role — user becomes individual
	MustSucceed(admin1.RemoveUserRole(admin2.ID))
	s.Equal("individual", admin2.GetRole())
	s.T().Logf("Admin1 removed admin2's admin role")

	// The free-use key should be cleared because admin2 is no longer an admin
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyID, "free-use key should be cleared when the key owner loses admin role")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("System free-use key correctly cleared after admin role removal")
}
