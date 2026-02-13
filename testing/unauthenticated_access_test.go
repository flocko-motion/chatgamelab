package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// UnauthenticatedAccessTestSuite tests that unauthenticated users (no auth token at all)
// cannot access protected endpoints. This is a sanity check for the auth middleware.
type UnauthenticatedAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUnauthenticatedAccessSuite(t *testing.T) {
	s := &UnauthenticatedAccessTestSuite{}
	s.SuiteName = "Unauthenticated Access Tests"
	suite.Run(t, s)
}

// TestUnauthenticatedCannotListUsers verifies that an unauthenticated client
// cannot access GET /api/users.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListUsers() {
	pub := s.Public()
	pub.FailGet("users")
	s.T().Logf("Correctly denied unauthenticated access to user list")
}

// TestUnauthenticatedCannotGetCurrentUser verifies that an unauthenticated client
// cannot access GET /api/users/me.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotGetCurrentUser() {
	pub := s.Public()
	pub.FailGet("users/me")
	s.T().Logf("Correctly denied unauthenticated access to current user")
}

// TestUnauthenticatedCannotGetUserByID verifies that an unauthenticated client
// cannot access GET /api/users/{id}.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotGetUserByID() {
	// Create a user so we have a valid ID to try
	user := s.CreateUser("unauth-target")

	pub := s.Public()
	pub.FailGet("users/" + user.ID)
	s.T().Logf("Correctly denied unauthenticated access to user profile")
}

// TestUnauthenticatedCannotListOwnGames verifies that an unauthenticated client
// cannot use the "own" filter on GET /api/games.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListOwnGames() {
	pub := s.Public()
	pub.FailGet("games?filter=own")
	s.T().Logf("Correctly denied unauthenticated access to own games filter")
}

// TestUnauthenticatedCannotCreateGame verifies that an unauthenticated client
// cannot create a game via POST /api/games/new.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotCreateGame() {
	pub := s.Public()
	pub.FailPost("games/new", map[string]interface{}{"name": "Unauthorized Game"})
	s.T().Logf("Correctly denied unauthenticated game creation")
}

// TestUnauthenticatedCannotListApiKeys verifies that an unauthenticated client
// cannot access GET /api/apikeys.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListApiKeys() {
	pub := s.Public()
	pub.FailGet("apikeys")
	s.T().Logf("Correctly denied unauthenticated access to API keys")
}

// TestUnauthenticatedCannotListInstitutions verifies that an unauthenticated client
// cannot access GET /api/institutions.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListInstitutions() {
	pub := s.Public()
	pub.FailGet("institutions")
	s.T().Logf("Correctly denied unauthenticated access to institutions")
}

// TestUnauthenticatedCannotListWorkshops verifies that an unauthenticated client
// cannot access GET /api/workshops.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListWorkshops() {
	pub := s.Public()
	pub.FailGet("workshops")
	s.T().Logf("Correctly denied unauthenticated access to workshops")
}

// TestUnauthenticatedCannotCreateSession verifies that an unauthenticated client
// cannot create a game session via POST /api/games/{id}/sessions.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotCreateSession() {
	// Create a user and game so we have a valid game ID
	user := s.CreateUser("unauth-sess")
	Must(user.AddApiKey("mock-unauth-sess", "Key", "mock"))
	game := Must(user.UploadGame("alien-first-contact"))

	pub := s.Public()
	pub.FailPost("games/"+game.ID.String()+"/sessions", nil)
	s.T().Logf("Correctly denied unauthenticated session creation")
}

// TestUnauthenticatedCannotDeleteGame verifies that an unauthenticated client
// cannot delete a game.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotDeleteGame() {
	user := s.CreateUser("unauth-del")
	game := Must(user.UploadGame("alien-first-contact"))

	pub := s.Public()
	// PublicClient doesn't have Delete, so use FailGet on a non-existent endpoint
	// to verify auth is required. We test via POST to a protected endpoint instead.
	pub.FailPost("games/"+game.ID.String(), map[string]interface{}{"name": "Hacked"})
	s.T().Logf("Correctly denied unauthenticated game update")
}

// TestUnauthenticatedCannotListGames verifies that unauthenticated users
// cannot list any games, not even public ones.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListGames() {
	pub := s.Public()
	pub.FailGet("games")
	s.T().Logf("Correctly denied unauthenticated access to games list")
}

// TestUnauthenticatedCannotListPublicGames verifies that unauthenticated users
// cannot list public games either — no platform access without auth.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListPublicGames() {
	pub := s.Public()
	pub.FailGet("games?filter=public")
	s.T().Logf("Correctly denied unauthenticated access to public games")
}

// TestUnauthenticatedCannotGetGameByID verifies that unauthenticated users
// cannot access a specific game by ID.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotGetGameByID() {
	user := s.CreateUser("unauth-game")
	game := Must(user.UploadGame("alien-first-contact"))

	pub := s.Public()
	pub.FailGet("games/" + game.ID.String())
	s.T().Logf("Correctly denied unauthenticated access to game by ID")
}

// TestUnauthenticatedCanAccessStatus verifies that the status endpoint
// is accessible without authentication (sanity check).
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCanAccessStatus() {
	pub := s.Public()
	var result map[string]interface{}
	pub.MustGet("status", &result)
	s.Equal("running", result["status"])
	s.T().Logf("Status endpoint accessible without auth")
}
