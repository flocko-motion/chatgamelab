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

// TestMultiUserScenario demonstrates how to test with multiple users
func (s *MultiUserTestSuite) TestMultiUserScenario() {
	// Create multiple users
	alice := s.CreateUser("alice", "alice@example.com")
	bob := s.CreateUser("bob", "bob@example.com")

	// Alice creates a game
	var aliceGame obj.Game
	alice.MustPost("games/new", map[string]interface{}{
		"name": "Alice's Adventure",
	}, &aliceGame)
	s.NotEmpty(aliceGame.ID)
	s.T().Logf("Alice created game: %s", aliceGame.ID)

	// Bob creates a different game
	var bobGame obj.Game
	bob.MustPost("games/new", map[string]interface{}{
		"name": "Bob's Quest",
	}, &bobGame)
	s.NotEmpty(bobGame.ID)
	s.T().Logf("Bob created game: %s", bobGame.ID)

	// Both users can list games and see both
	var aliceGames []obj.Game
	alice.MustGet("games", &aliceGames)
	s.GreaterOrEqual(len(aliceGames), 2, "Alice should see at least 2 games")

	var bobGames []obj.Game
	bob.MustGet("games", &bobGames)
	s.GreaterOrEqual(len(bobGames), 2, "Bob should see at least 2 games")

	// Alice can retrieve her own game
	var retrievedGame obj.Game
	alice.MustGet("games/"+aliceGame.ID.String(), &retrievedGame)
	s.Equal("Alice's Adventure", retrievedGame.Name)

	// Bob can also retrieve Alice's game (public access)
	var bobViewOfAliceGame obj.Game
	bob.MustGet("games/"+aliceGame.ID.String(), &bobViewOfAliceGame)
	s.Equal("Alice's Adventure", bobViewOfAliceGame.Name)
}

// TestUserRegistry demonstrates creating users in the registry
func (s *MultiUserTestSuite) TestUserRegistry() {
	// Create a user
	dave := s.CreateUser("dave", "dave@example.com")

	var game obj.Game
	dave.MustPost("games/new", map[string]interface{}{
		"name": "Dave's Game",
	}, &game)

	s.T().Logf("Dave (ID: %s) created game: %s", dave.ID, game.ID)
}

// TestReuseUser shows how users are isolated per test
func (s *MultiUserTestSuite) TestReuseUser() {
	// This will fail if dave doesn't exist
	// In a real scenario, you'd use CreateUser which returns existing user if found
	dave := s.CreateUser("dave", "dave@example.com")

	var games []obj.Game
	dave.MustGet("games", &games)

	s.T().Logf("Dave can list %d games", len(games))
}
