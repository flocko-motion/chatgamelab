//go:build ai_tests

package testing

import (
	"log"
	"testing"

	"cgl/functional"
	"cgl/game/ai"
	"cgl/obj"
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

func (s *GameEngineTestSuite) TestGamePlaythroughOpenai() {
	apiKeyShare := Must(s.clientAlice.AddApiKey(ai.GetApiKeyOpenAI(), "Test OpenAI Key", "openai"))
	s.T().Logf("Added API key: %s", apiKeyShare.ID)
	s.GamePlaythrough(apiKeyShare)
}

func (s *GameEngineTestSuite) TestGamePlaythroughMistral() {
	apiKeyShare := Must(s.clientAlice.AddApiKey(ai.GetApiKeyMistral(), "Test Mistral Key", "mistral"))
	s.T().Logf("Added API key: %s", apiKeyShare.ID)
	s.GamePlaythrough(apiKeyShare)
}

// GamePlaythrough tests the complete game engine workflow:
// - Add API key
// - Upload game from YAML
// - Create game session
// - Send prompt and receive AI response
// - Validate status field updates
func (s *GameEngineTestSuite) GamePlaythrough(apiKeyShare obj.ApiKeyShare) {

	// Set preferred language to French
	err := s.clientAlice.SetUserLanguage("fr")
	s.Require().NoError(err, "Failed to set language to French")
	s.T().Logf("Set language preference to French")

	// Verify language was set
	me := Must(s.clientAlice.GetMe())
	s.Equal("fr", me.Language, "Language should be set to French")
	s.T().Logf("Verified language preference: %s", me.Language)

	// Create and upload predictable test game
	game := Must(s.clientAlice.UploadGame("stone-collector-de"))
	s.T().Logf("Created and uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create game session - game should be auto-translated to French
	sessionResponse := Must(s.clientAlice.CreateGameSession(game.ID.String()))
	s.T().Logf("Created game session: %s", sessionResponse.ID)

	// Log initial message
	s.Require().NotEmpty(sessionResponse.Messages, "Session should have initial message")
	initialMsg := sessionResponse.Messages[0]
	log.Printf("\n=================================================================================================\n")
	log.Printf("Initial Message (should be in FRENCH):")
	log.Printf("  Analytics: %s", functional.MaybeToString(initialMsg.URLAnalytics, "nil"))
	log.Printf("  PromptStatusUpdate: %s", functional.MaybeToString(initialMsg.PromptStatusUpdate, "nil"))
	log.Printf("  PromptExpandStory: %s", functional.MaybeToString(initialMsg.PromptExpandStory, "nil"))
	log.Printf("  PromptImageGeneration: %s", functional.MaybeToString(initialMsg.PromptImageGeneration, "nil"))
	log.Printf("  ResponseRaw: %s", functional.MaybeToString(initialMsg.ResponseRaw, "nil"))
	log.Printf("  AI: %s", initialMsg.Message)

	// Verify token usage from session creation (includes theme + translation + initial action)
	s.Require().NotNil(initialMsg.TokenUsage, "Initial message should have token usage")
	s.Greater(initialMsg.TokenUsage.InputTokens, 0, "Initial message should have input tokens > 0")
	s.Greater(initialMsg.TokenUsage.OutputTokens, 0, "Initial message should have output tokens > 0")
	s.Greater(initialMsg.TokenUsage.TotalTokens, 0, "Initial message should have total tokens > 0")
	log.Printf("  TokenUsage: input=%d, output=%d, total=%d", initialMsg.TokenUsage.InputTokens, initialMsg.TokenUsage.OutputTokens, initialMsg.TokenUsage.TotalTokens)

	// Player actions in French (game is translated to French)
	playerActions := []string{
		"Je ramasse 5 pierres",
		"Je ramasse 3 pierres",
		"Je ramasse 2 pierres",
	}
	messageLens := []int{}
	for i, playerAction := range playerActions {
		msg1, err := s.clientAlice.SendGameMessage(sessionResponse.ID.String(), playerAction)
		s.Require().NoError(err, "SendGameMessage failed for action #%d: %s", i, playerAction)
		log.Printf("\n=================================================================================================\n")
		log.Printf("Turn #%d - Player: %s", i, playerAction)
		log.Printf("Analytics: %s", functional.MaybeToString(msg1.URLAnalytics, "nil"))
		log.Printf("PromptStatusUpdate: %s", functional.MaybeToString(msg1.PromptStatusUpdate, "nil"))
		log.Printf("PromptExpandStory: %s", functional.MaybeToString(msg1.PromptExpandStory, "nil"))
		log.Printf("PromptImageGeneration: %s", functional.MaybeToString(msg1.PromptImageGeneration, "nil"))
		log.Printf("ResponseRaw: %s", functional.MaybeToString(msg1.ResponseRaw, "nil"))
		log.Printf("AI Story Len=%d (should be in FRENCH): %s", len(msg1.Message), functional.MaybeToString(msg1.Message, "nil"))
		s.Greater(len(msg1.Message), 10, "AI response should be substantial")
		messageLens = append(messageLens, len(msg1.Message))

		// Verify token usage for each action
		s.Require().NotNil(msg1.TokenUsage, "Action response should have token usage")
		s.Greater(msg1.TokenUsage.InputTokens, 0, "Action should have input tokens > 0")
		s.Greater(msg1.TokenUsage.OutputTokens, 0, "Action should have output tokens > 0")
		s.Greater(msg1.TokenUsage.TotalTokens, 0, "Action should have total tokens > 0")
		log.Printf("TokenUsage: input=%d, output=%d, total=%d", msg1.TokenUsage.InputTokens, msg1.TokenUsage.OutputTokens, msg1.TokenUsage.TotalTokens)
	}

	// --- Session resume: simulate player closing browser and returning later ---
	log.Printf("\n=================================================================================================\n")
	log.Printf("Session resume: loading all messages (simulating browser reload)")

	resumed := Must(s.clientAlice.GetGameSession(sessionResponse.ID.String()))
	s.Require().NotNil(resumed.GameSession, "Resumed session should not be nil")
	s.Equal(sessionResponse.ID, resumed.ID, "Session ID should match")

	// Expect: 1 initial game message + (N player actions * 2: player + game response)
	expectedMessageCount := 2 + len(playerActions)*2
	s.Require().Len(resumed.Messages, expectedMessageCount,
		"Should have %d messages: 1 initial + %d player actions + %d AI responses",
		expectedMessageCount, len(playerActions), len(playerActions))

	// Message[0] should be the initial game response (system message triggers this)
	s.Equal("system", resumed.Messages[0].Type, "First message should be a game response (from system prompt)")
	s.Equal("game", resumed.Messages[1].Type, "Second message should be a game start scenario")
	s.Equal("player", resumed.Messages[2].Type, "Second message should be a player input")

	// Log all messages and verify prompt fields on game-type messages
	for i, msg := range resumed.Messages {
		log.Printf("  Message[%d] type=%-6s seq=%d len=%d", i, msg.Type, msg.Seq, len(msg.Message))

		if msg.Type == "game" {
			s.Require().NotNil(msg.PromptExpandStory, "Message[%d] PromptExpandStory should not be nil", i)
			s.Greater(len(*msg.PromptExpandStory), 0, "Message[%d] PromptExpandStory should not be empty", i)

			s.Require().NotNil(msg.PromptStatusUpdate, "Message[%d] PromptStatusUpdate should not be nil", i)
			s.Greater(len(*msg.PromptStatusUpdate), 0, "Message[%d] PromptStatusUpdate should not be empty", i)

			s.Require().NotNil(msg.PromptResponseSchema, "Message[%d] PromptResponseSchema should not be nil", i)
			s.Greater(len(*msg.PromptResponseSchema), 0, "Message[%d] PromptResponseSchema should not be empty", i)

			s.Require().NotNil(msg.PromptImageGeneration, "Message[%d] PromptImageGeneration should not be nil", i)
			s.Greater(len(*msg.PromptImageGeneration), 0, "Message[%d] PromptImageGeneration should not be empty", i)
		}
	}

	log.Printf("Game engine test completed successfully! Review terminal output to verify French translation.")
	log.Printf("Message lengths: %v", messageLens)
}
