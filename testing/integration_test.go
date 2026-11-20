package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// BasicTestSuite contains basic API tests
type BasicTestSuite struct {
	testutil.BaseSuite
}

// TestBasicSuite runs the basic test suite
func TestBasicSuite(t *testing.T) {
	s := &BasicTestSuite{}
	s.SuiteName = "Basic API Tests"
	suite.Run(t, s)
}

// TestStatusEndpoint tests schemaf's status endpoint
func (s *BasicTestSuite) TestStatusEndpoint() {
	status := Must(s.Public().Status())

	s.Equal("ok", status["status"])
	s.NotEmpty(status["uptime"])
}

func (s *BasicTestSuite) TestCreateGame() {
	alice := s.CreateUser("alice", "alice@example.com")

	game := Must(alice.CreateGame("Test Game"))

	s.NotEmpty(game.Id, "game ID should not be empty")
	s.Equal("Test Game", game.Name)
}

func (s *BasicTestSuite) TestListGames() {
	bob := s.CreateUser("bob", "bob@example.com")
	Must(bob.CreateGame("Game for List Test"))

	games := Must(bob.ListGames())

	s.NotEmpty(games, "should have at least one game")
}

func (s *BasicTestSuite) TestUnauthorizedAccess() {
	_, err := s.Public().CreateGame("Unauthorized Game")
	s.Unauthorized(err)
}

func (s *BasicTestSuite) TestGetGameByID() {
	charlie := s.CreateUser("charlie", "charlie@example.com")
	created := Must(charlie.CreateGame("Game to Retrieve"))

	retrieved := Must(charlie.GetGameByID(created.Id.String()))

	s.Equal(created.Id, retrieved.Id)
	s.Equal("Game to Retrieve", retrieved.Name)
}
