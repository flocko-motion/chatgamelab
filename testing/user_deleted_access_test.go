package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// UserDeletedAccessTestSuite tests that a deleted user's token is rejected
// for all subsequent API calls.
type UserDeletedAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUserDeletedAccessSuite(t *testing.T) {
	s := &UserDeletedAccessTestSuite{}
	s.SuiteName = "User Deleted Access Tests"
	suite.Run(t, s)
}

// TestDeletedUserCannotListGames verifies that after an admin deletes a user,
// the user's existing token can no longer list games.
func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotListGames() {
	admin := s.DevUser()

	// Create a user and verify they can list games
	user := s.CreateUser("del-access-list")
	_, err := user.ListGames()
	s.NoError(err, "user should be able to list games before deletion")
	s.T().Logf("User can list games before deletion")

	// Admin deletes the user
	MustSucceed(admin.DeleteUser(user.ID))
	s.T().Logf("Admin deleted user %s", user.Name)

	// User's token should now be rejected
	user.FailGet("games", testutil.ErrorContains("401"))
	s.T().Logf("Deleted user correctly rejected when listing games")
}

// TestDeletedUserCannotCreateGame verifies that after an admin deletes a user,
// the user's existing token can no longer create games.
func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotCreateGame() {
	admin := s.DevUser()

	// Create a user and verify they can create a game
	user := s.CreateUser("del-access-create")
	game := Must(user.UploadGame("alien-first-contact"))
	s.NotEmpty(game.ID, "user should be able to create a game before deletion")
	s.T().Logf("User created game %s before deletion", game.ID)

	// Admin deletes the user
	MustSucceed(admin.DeleteUser(user.ID))
	s.T().Logf("Admin deleted user %s", user.Name)

	// User's token should now be rejected for game creation
	user.FailPost("games/new", map[string]string{"name": "Should Fail"}, testutil.ErrorContains("401"))
	s.T().Logf("Deleted user correctly rejected when creating a game")
}

// TestDeletedUserCannotAccessProfile verifies that after an admin deletes a user,
// the user's existing token can no longer access their own profile.
func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotAccessProfile() {
	admin := s.DevUser()

	// Create a user and verify they can access their profile
	user := s.CreateUser("del-access-profile")
	me := Must(user.GetMe())
	s.Equal(user.Name, me.Name, "user should be able to access profile before deletion")
	s.T().Logf("User can access profile before deletion")

	// Admin deletes the user
	MustSucceed(admin.DeleteUser(user.ID))
	s.T().Logf("Admin deleted user %s", user.Name)

	// User's token should now be rejected
	user.FailGet("users/me", testutil.ErrorContains("401"))
	s.T().Logf("Deleted user correctly rejected when accessing profile")
}
