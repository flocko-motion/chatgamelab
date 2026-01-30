//go:build ai_tests

package testing

import (
	"os"
	"path/filepath"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

var apiKeyPathOpenai = filepath.Join(os.Getenv("HOME"), ".ai", "openai", "api-keys", "current")

type GameEngineTestSuite struct {
	testutil.BaseSuite
	clientAlice *testutil.UserClient
}

func (s *GameEngineTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.clientAlice = s.CreateUser("alice-game-engine")
	s.T().Logf("Created user: %s", s.clientAlice.Name)
}

func TestGameEngineTestSuite(t *testing.T) {
	suite.Run(t, new(GameEngineTestSuite))
}

// TestGamePlaythrough tests the complete game engine workflow:
// - Create user
// - Add API key
// - Upload game from YAML
// - Create game session
// - Send prompt and receive AI response
// - Validate status field updates
func (s *GameEngineTestSuite) TestGamePlaythrough() {
	apiKeyShare := Must(s.clientAlice.AddApiKey(apiKeyPathOpenai, "Test OpenAI Key", "openai"))
	s.T().Logf("Added API key: %s", apiKeyShare.ID)

	// Create and upload predictable test game
	game := Must(s.clientAlice.UploadGame("stone-collector"))
	s.Equal("Stone Collector", game.Name)
	s.T().Logf("Created and uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create game session
	session := Must(s.clientAlice.CreateGameSession(game.ID.String(), apiKeyShare.ID))
	s.T().Logf("Created game session: %s", session.ID)

	// Turn 1: Collect 5 stones
	msg1 := Must(s.clientAlice.SendGameMessage(session.ID.String(), "I collect 5 stones"))
	s.T().Logf("Turn 1 - Player: I collect 5 stones")
	s.T().Logf("Turn 1 - AI: %s", msg1.Message)
	s.Greater(len(msg1.Message), 10, "AI response should be substantial")
	// TODO: Validate status fields show Day=2, Stones=5

	// Turn 2: Collect 3 more stones
	msg2 := Must(s.clientAlice.SendGameMessage(session.ID.String(), "I collect 3 more stones"))
	s.T().Logf("Turn 2 - Player: I collect 3 more stones")
	s.T().Logf("Turn 2 - AI: %s", msg2.Message)
	s.Greater(len(msg2.Message), 10, "AI response should be substantial")
	// TODO: Validate status fields show Day=3, Stones=8

	// Turn 3: Collect 2 more stones
	msg3 := Must(s.clientAlice.SendGameMessage(session.ID.String(), "I collect 2 more stones"))
	s.T().Logf("Turn 3 - Player: I collect 2 more stones")
	s.T().Logf("Turn 3 - AI: %s", msg3.Message)
	s.Greater(len(msg3.Message), 10, "AI response should be substantial")
	// TODO: Validate status fields show Day=4, Stones=10

	s.T().Logf("Game engine test completed successfully!")
}
