package testing

import (
	"cgl/obj"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// MultiUserTestSuite contains multi-user collaboration tests
// Each suite gets its own fresh Docker environment
type MultiUserTestSuite struct {
	testutil.BaseSuite
}

// TestMultiUserSuite runs the multi-user test suite
func TestMultiUserSuite(t *testing.T) {
	s := &MultiUserTestSuite{}
	s.SuiteName = "Multi-User Tests"
	suite.Run(t, s)
}

// TestUserManagement tests basic user operations
func (s *MultiUserTestSuite) TestUserManagement() {
	// List users - should have 1 (default dev user)
	admin := s.CreateUser("admin")
	s.Role(admin, "admin")

	var initialUsers []obj.User
	admin.MustGet("users", &initialUsers)
	s.Equal(1, len(initialUsers), "should have 1 user (dev user)")
	s.T().Logf("Initial users: %d", len(initialUsers))

	// Create admin user (already created above)
	s.Equal("admin", admin.Name)
	s.Equal("admin@test.local", admin.Email)

	// Create normal user
	alice := s.CreateUser("alice")
	s.Equal("alice", alice.Name)
	s.Equal("alice@test.local", alice.Email)

	// List users again - should have 3 now (dev + admin + alice)
	var allUsers []obj.User
	admin.MustGet("users", &allUsers)
	s.Equal(3, len(allUsers), "should have 3 users (dev + admin + alice)")

	// Verify user names
	userNames := make([]string, len(allUsers))
	for i, user := range allUsers {
		userNames[i] = user.Name
	}
	s.Contains(userNames, "admin")
	s.Contains(userNames, "alice")

	s.T().Logf("All users: %v", userNames)
}
