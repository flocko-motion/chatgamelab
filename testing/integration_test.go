package testing

import (
	"cgl/obj"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// BasicTestSuite contains basic API tests
// Each suite gets its own fresh Docker environment
type BasicTestSuite struct {
	testutil.BaseSuite
}

// TestBasicSuite runs the basic test suite
func TestBasicSuite(t *testing.T) {
	s := &BasicTestSuite{}
	s.SuiteName = "Basic API Tests"
	suite.Run(t, s)
}

// TestStatusEndpoint tests the status endpoint
func (s *BasicTestSuite) TestStatusEndpoint() {
	public := s.Public()

	var result map[string]interface{}
	public.MustGet("status", &result)

	s.Equal("running", result["status"])
	s.NotEmpty(result["uptime"])
}

func (s *BasicTestSuite) TestCreateGame() {
	alice := s.CreateUser("alice", "alice@example.com")

	// Create a new game
	payload := map[string]interface{}{
		"name": "Test Game",
	}

	var game obj.Game
	alice.MustPost("games/new", payload, &game)

	// Verify game was created
	s.NotEmpty(game.ID, "game ID should not be empty")
	s.Equal("Test Game", game.Name)

	s.T().Logf("Created game with ID: %s", game.ID)
}

func (s *BasicTestSuite) TestListGames() {
	bob := s.CreateUser("bob", "bob@example.com")

	// Create a game first
	createPayload := map[string]interface{}{
		"name": "Game for List Test",
	}
	var createdGame obj.Game
	bob.MustPost("games/new", createPayload, &createdGame)

	// List games
	var games []obj.Game
	bob.MustGet("games", &games)

	// Should have at least one game
	s.NotEmpty(games, "should have at least one game")

	s.T().Logf("Found %d games", len(games))
}

func (s *BasicTestSuite) TestUnauthorizedAccess() {
	public := s.Public()

	// Try to create game without authentication
	payload := map[string]interface{}{
		"name": "Unauthorized Game",
	}

	// Expect error with any message
	public.FailPost("games/new", payload)
}

func (s *BasicTestSuite) TestGetGameByID() {
	charlie := s.CreateUser("charlie", "charlie@example.com")

	// Create a game
	createPayload := map[string]interface{}{
		"name": "Game to Retrieve",
	}
	var createdGame obj.Game
	charlie.MustPost("games/new", createPayload, &createdGame)
	s.NotEmpty(createdGame.ID)

	// Get the game by ID
	var retrievedGame obj.Game
	charlie.MustGet("games/"+createdGame.ID.String(), &retrievedGame)

	s.Equal(createdGame.ID, retrievedGame.ID)
	s.Equal("Game to Retrieve", retrievedGame.Name)
}
