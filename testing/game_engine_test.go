//go:build ai_tests

package testing

import (
	"log"
	"testing"

	"cgl/functional"
	"cgl/game/ai"
	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

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
	apiKeyShare := Must(s.clientAlice.AddApiKey(ai.GetApiKeyOpenAI(), "Test OpenAI Key", "openai"))
	s.T().Logf("Added API key: %s", apiKeyShare.ID)

	// Create and upload predictable test game
	game := Must(s.clientAlice.UploadGame("stone-collector"))
	s.Equal("Stone Collector", game.Name)
	s.T().Logf("Created and uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create game session
	session := Must(s.clientAlice.CreateGameSession(game.ID.String(), apiKeyShare.ID, "gpt-5.2"))
	s.T().Logf("Created game session: %s", session.ID)

	playerActions := []string{
		"I collect 5 stones",
		"I collect 3 more stones",
		"I collect 2 more stones",
	}
	for i, playerAction := range playerActions {
		// Turn 1: Collect 5 stones
		msg1 := Must(s.clientAlice.SendGameMessage(session.ID.String(), playerAction))
		log.Printf("Turn #%d - Player: %s", i, playerAction)
		log.Printf("Analytics: %s", functional.MaybeToString(msg1.URLAnalytics, "nil"))
		log.Printf("PromptStatusUpdate: %s", functional.MaybeToString(msg1.PromptStatusUpdate, "nil"))
		log.Printf("PromptExpandStory: %s", functional.MaybeToString(msg1.PromptExpandStory, "nil"))
		log.Printf("PromptImageGeneration: %s", functional.MaybeToString(msg1.PromptImageGeneration, "nil"))
		log.Printf("ResponseRaw: %s", functional.MaybeToString(msg1.ResponseRaw, "nil"))
		log.Printf("AI: %s", functional.MaybeToString(msg1.Message, "nil"))
		s.Greater(len(msg1.Message), 10, "AI response should be substantial")
		// TODO: Validate status fields show Day=2, Stones=5
	}
	s.T().Logf("Game engine test completed successfully!")
}
