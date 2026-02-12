//go:build ai_tests

package testing

import (
	"log"
	"os"
	"os/exec"
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

// TestAudioPlaythroughOpenai tests the audio output feature with the "max" quality tier.
// Uses OpenAI's gpt-4o-mini-tts to generate audio narration alongside text and image.
func (s *GameEngineTestSuite) TestAudioPlaythroughOpenai() {
	// Add OpenAI API key
	apiKeyShare := Must(s.clientAlice.AddApiKey(ai.GetApiKeyOpenAI(), "Test OpenAI Audio Key", "openai"))
	s.T().Logf("Added API key: %s", apiKeyShare.ID)

	// Set quality tier to "max" (= high text model + audio)
	err := s.clientAlice.SetUserAiQualityTier("max")
	s.Require().NoError(err, "Failed to set AI quality tier to max")
	s.T().Logf("Set AI quality tier to max")

	// Upload a simple test game
	game := Must(s.clientAlice.UploadGame("stone-collector"))
	s.T().Logf("Created and uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Create game session and consume the initial SSE stream (text + image + audio)
	sessionResponse, initialStream, err := s.clientAlice.CreateGameSessionWithStream(game.ID.String())
	s.Require().NoError(err, "CreateGameSessionWithStream failed")
	s.T().Logf("Created game session: %s (model: %s, platform: %s)", sessionResponse.ID, sessionResponse.AiModel, sessionResponse.AiPlatform)

	// Verify session was created with "max" tier
	s.Equal("max", sessionResponse.AiModel, "Session should use max tier")
	s.Equal("openai", sessionResponse.AiPlatform, "Session should use openai platform")

	// Validate initial message has text + audio
	s.Require().NotEmpty(sessionResponse.Messages, "Session should have messages")
	// Find the game-type message (index 1, after system message)
	var initialMsg *obj.GameSessionMessage
	for i := range sessionResponse.Messages {
		if sessionResponse.Messages[i].Type == "game" {
			initialMsg = &sessionResponse.Messages[i]
			break
		}
	}
	s.Require().NotNil(initialMsg, "Should have a game-type initial message")
	s.Greater(len(initialMsg.Message), 10, "Initial message should have substantial text")
	s.True(initialMsg.HasImage, "Initial message should have HasImage=true (max tier on OpenAI)")
	s.True(initialMsg.HasAudio, "Initial message should have HasAudio=true (max tier on OpenAI)")
	log.Printf("Initial message (len=%d): %s", len(initialMsg.Message), initialMsg.Message)

	// Validate audio on initial message
	s.Require().NotNil(initialStream, "Initial stream result should not be nil")
	s.validateAudioStream(initialStream, "initial message")

	// Validate audio is persisted in DB and accessible via replay endpoint
	s.validateAudioFromDB(initialMsg.ID.String(), initialStream, "initial message")

	// Send a player action and consume the SSE stream (text + image + audio)
	msg, streamResult, err := s.clientAlice.SendGameMessageWithStream(sessionResponse.ID.String(), "I collect 3 stones")
	s.Require().NoError(err, "SendGameMessageWithStream failed")

	// Validate text and capability flags
	s.Greater(len(msg.Message), 10, "AI response text should be substantial")
	s.True(msg.HasImage, "Action response should have HasImage=true (max tier on OpenAI)")
	s.True(msg.HasAudio, "Action response should have HasAudio=true (max tier on OpenAI)")
	log.Printf("AI response (len=%d): %s", len(msg.Message), msg.Message)

	// Validate audio on player action response
	s.Require().NotNil(streamResult, "Stream result should not be nil")
	s.validateAudioStream(streamResult, "action response")
	s.validateAudioFromDB(msg.ID.String(), streamResult, "action response")

	log.Printf("Audio playthrough test completed successfully!")
}

// validateAudioStream checks that audio data was received via SSE and has valid MP3 header
func (s *GameEngineTestSuite) validateAudioStream(result *testutil.StreamResult, label string) {
	s.T().Helper()
	s.Greater(len(result.AudioData), 0, "%s: audio data should be received via SSE stream", label)
	log.Printf("%s: audio data received: %d bytes", label, len(result.AudioData))

	s.Require().GreaterOrEqual(len(result.AudioData), 2, "%s: audio data should be at least 2 bytes", label)
	s.Equal(byte(0xFF), result.AudioData[0], "%s: audio should start with MP3 sync byte 0xFF", label)
	s.Equal(byte(0xFB), result.AudioData[1]&0xFB, "%s: audio should have MP3 frame header", label)
}

// validateAudioFromDB checks that audio is persisted and accessible via the replay endpoint
func (s *GameEngineTestSuite) validateAudioFromDB(messageID string, streamResult *testutil.StreamResult, label string) {
	s.T().Helper()
	audioFromDB, err := s.clientAlice.GetMessageAudio(messageID)
	s.Require().NoError(err, "%s: GetMessageAudio should succeed", label)
	s.Greater(len(audioFromDB), 0, "%s: audio from DB should not be empty", label)
	s.Equal(byte(0xFF), audioFromDB[0], "%s: audio from DB should start with MP3 sync byte 0xFF", label)
	log.Printf("%s: audio from DB: %d bytes (matches stream: %v)", label, len(audioFromDB), len(audioFromDB) == len(streamResult.AudioData))

	playAudio(audioFromDB, label)
}

// playAudio writes audio bytes to a temp file and plays them via paplay (blocking).
// Silently skips if paplay is not available.
func playAudio(data []byte, label string) {
	paplay, err := exec.LookPath("paplay")
	if err != nil {
		return
	}
	tmpFile, err := os.CreateTemp("", "cgl-audio-*.mp3")
	if err != nil {
		return
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return
	}
	tmpFile.Close()
	log.Printf("%s: playing audio via paplay...", label)
	cmd := exec.Command(paplay, tmpFile.Name())
	if err := cmd.Run(); err != nil {
		log.Printf("%s: paplay failed: %v", label, err)
	}
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

	// Determine expected capabilities from the platform
	expectImage := apiKeyShare.ApiKey.Platform == "openai" // OpenAI supports images, Mistral does not
	expectAudio := false                                   // standard tiers never have audio

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
		s.Equal(expectImage, msg1.HasImage, "Turn #%d: HasImage should be %v for platform %s", i, expectImage, apiKeyShare.ApiKey.Platform)
		s.Equal(expectAudio, msg1.HasAudio, "Turn #%d: HasAudio should be %v", i, expectAudio)
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

			// Verify capability flags are persisted and loaded correctly
			s.Equal(expectImage, msg.HasImage, "Message[%d] HasImage should be %v (persisted)", i, expectImage)
			s.Equal(expectAudio, msg.HasAudio, "Message[%d] HasAudio should be %v (persisted)", i, expectAudio)
		}
	}

	log.Printf("Game engine test completed successfully! Review terminal output to verify French translation.")
	log.Printf("Message lengths: %v", messageLens)
}
