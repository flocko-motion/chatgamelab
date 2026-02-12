package testing

import (
	"cgl/api/routes"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ApiKeyIsolationTestSuite tests that no user can ever see another user's
// API keys, regardless of role combination.
type ApiKeyIsolationTestSuite struct {
	testutil.BaseSuite
}

func TestApiKeyIsolationSuite(t *testing.T) {
	s := &ApiKeyIsolationTestSuite{}
	s.SuiteName = "API Key Isolation Tests"
	suite.Run(t, s)
}

// assertNoForeignKeys checks that none of the API keys in the response
// belong to the forbidden user.
func (s *ApiKeyIsolationTestSuite) assertNoForeignKeys(resp routes.ApiKeysResponse, forbiddenUserName string, forbiddenKeyNames []string) {
	s.T().Helper()
	forbidden := make(map[string]bool)
	for _, name := range forbiddenKeyNames {
		forbidden[name] = true
	}
	for _, key := range resp.ApiKeys {
		s.False(forbidden[key.Name], "should not see %s's key %q in ApiKeys list", forbiddenUserName, key.Name)
	}
	for _, share := range resp.Shares {
		if share.ApiKey != nil {
			s.False(forbidden[share.ApiKey.Name], "should not see %s's key %q in Shares list", forbiddenUserName, share.ApiKey.Name)
		}
	}
}

// TestAdminCannotSeeOtherAdminKeys verifies admin cannot see another admin's keys.
func (s *ApiKeyIsolationTestSuite) TestAdminCannotSeeOtherAdminKeys() {
	admin := s.DevUser()
	Must(admin.AddApiKey("mock-admin-secret", "Admin Secret Key", "mock"))

	admin2 := s.CreateUser("iso-admin2")
	s.Role(admin2, "admin")
	Must(admin2.AddApiKey("mock-admin2-secret", "Admin2 Secret Key", "mock"))

	// Admin2 lists keys — should NOT see admin's key
	resp := Must(admin2.GetApiKeys())
	s.assertNoForeignKeys(resp, "admin", []string{"Admin Secret Key"})
	s.T().Logf("Admin2 sees %d keys, %d shares — none from admin", len(resp.ApiKeys), len(resp.Shares))

	// Admin lists keys — should NOT see admin2's key
	resp = Must(admin.GetApiKeys())
	s.assertNoForeignKeys(resp, "admin2", []string{"Admin2 Secret Key"})
	s.T().Logf("Admin sees %d keys, %d shares — none from admin2", len(resp.ApiKeys), len(resp.Shares))
}

// TestHeadCannotSeeIndividualKeys verifies head cannot see an individual user's keys.
func (s *ApiKeyIsolationTestSuite) TestHeadCannotSeeIndividualKeys() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Key Iso Org"))
	head := s.CreateUser("iso-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	Must(head.AddApiKey("mock-head-key", "Head Key", "mock"))

	individual := s.CreateUser("iso-individual")
	Must(individual.AddApiKey("mock-ind-secret", "Individual Secret Key", "mock"))

	// Head lists keys — should NOT see individual's key
	resp := Must(head.GetApiKeys())
	s.assertNoForeignKeys(resp, "individual", []string{"Individual Secret Key"})
	s.T().Logf("Head sees %d keys — none from individual", len(resp.ApiKeys))

	// Individual lists keys — should NOT see head's key
	resp = Must(individual.GetApiKeys())
	s.assertNoForeignKeys(resp, "head", []string{"Head Key"})
	s.T().Logf("Individual sees %d keys — none from head", len(resp.ApiKeys))
}

// TestStaffCannotSeeHeadKeys verifies staff cannot see head's keys
// even though they are in the same institution.
func (s *ApiKeyIsolationTestSuite) TestStaffCannotSeeHeadKeys() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff Key Org"))
	head := s.CreateUser("iso-head2")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	Must(head.AddApiKey("mock-head2-key", "Head2 Secret Key", "mock"))

	staff := s.CreateUser("iso-staff")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	Must(staff.AddApiKey("mock-staff-key", "Staff Secret Key", "mock"))

	// Staff lists keys — should NOT see head's key
	resp := Must(staff.GetApiKeys())
	s.assertNoForeignKeys(resp, "head", []string{"Head2 Secret Key"})
	s.T().Logf("Staff sees %d keys — none from head", len(resp.ApiKeys))

	// Head lists keys — should NOT see staff's key
	resp = Must(head.GetApiKeys())
	s.assertNoForeignKeys(resp, "staff", []string{"Staff Secret Key"})
	s.T().Logf("Head sees %d keys — none from staff", len(resp.ApiKeys))
}

// TestAdminCannotSeeHeadKeys verifies admin cannot see a head's keys.
func (s *ApiKeyIsolationTestSuite) TestAdminCannotSeeHeadKeys() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Admin Key Org"))
	head := s.CreateUser("iso-head3")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	Must(head.AddApiKey("mock-head3-key", "Head3 Secret Key", "mock"))

	// Admin lists keys — should NOT see head's key
	resp := Must(admin.GetApiKeys())
	s.assertNoForeignKeys(resp, "head", []string{"Head3 Secret Key"})
	s.T().Logf("Admin sees %d keys — none from head", len(resp.ApiKeys))
}

// TestCannotFetchOtherUsersKeyByID verifies that a user cannot fetch
// another user's API key share by guessing the share ID.
func (s *ApiKeyIsolationTestSuite) TestCannotFetchOtherUsersKeyByID() {
	userA := s.CreateUser("iso-usera")
	keyA := Must(userA.AddApiKey("mock-usera-key", "UserA Key", "mock"))

	userB := s.CreateUser("iso-userb")
	Must(userB.AddApiKey("mock-userb-key", "UserB Key", "mock"))

	// UserB tries to fetch UserA's key share by ID — should fail
	var info routes.ApiKeyInfoResponse
	err := userB.Get("apikeys/"+keyA.ID.String(), &info)
	s.Error(err, "user should not be able to fetch another user's API key by share ID")
	s.T().Logf("Correctly denied: %v", err)
}

// TestUserSeesOnlyOwnKeys is a sanity check that a user can see their own keys.
func (s *ApiKeyIsolationTestSuite) TestUserSeesOnlyOwnKeys() {
	user := s.CreateUser("iso-self")
	Must(user.AddApiKey("mock-self-key", "My Key", "mock"))

	resp := Must(user.GetApiKeys())
	s.Equal(1, len(resp.ApiKeys), "user should see exactly 1 key")
	s.Equal("My Key", resp.ApiKeys[0].Name)
	s.T().Logf("User sees own key: %s", resp.ApiKeys[0].Name)
}
