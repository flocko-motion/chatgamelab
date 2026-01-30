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
	game := Must(s.clientAlice.UploadGame("stone-collector-de"))
	s.T().Logf("Created and uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create game session
	sessionResponse := Must(s.clientAlice.CreateGameSession(game.ID.String(), apiKeyShare.ID, "gpt-5.2"))
	s.T().Logf("Created game session: %s", sessionResponse.ID)

	// Log initial message
	if len(sessionResponse.Messages) > 0 {
		initialMsg := sessionResponse.Messages[0]
		log.Printf("\n=================================================================================================\n")
		log.Printf("Initial Message:")
		log.Printf("  Analytics: %s", functional.MaybeToString(initialMsg.URLAnalytics, "nil"))
		log.Printf("  PromptStatusUpdate: %s", functional.MaybeToString(initialMsg.PromptStatusUpdate, "nil"))
		log.Printf("  PromptExpandStory: %s", functional.MaybeToString(initialMsg.PromptExpandStory, "nil"))
		log.Printf("  PromptImageGeneration: %s", functional.MaybeToString(initialMsg.PromptImageGeneration, "nil"))
		log.Printf("  ResponseRaw: %s", functional.MaybeToString(initialMsg.ResponseRaw, "nil"))
		log.Printf("  AI: %s", initialMsg.Message)
	}

	playerActions := []string{
		"Ich sammle 5 Steine",
		"Ich sammle 3 Steine",
		"Ich sammle 2 Steine",
	}
	for i, playerAction := range playerActions {
		msg1 := Must(s.clientAlice.SendGameMessage(sessionResponse.ID.String(), playerAction))
		log.Printf("\n=================================================================================================\n")
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
