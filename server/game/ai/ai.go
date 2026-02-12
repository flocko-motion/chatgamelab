package ai

import (
	"cgl/functional"
	"cgl/game/ai/mistral"
	"cgl/game/ai/mock"
	"cgl/game/ai/openai"
	"cgl/game/stream"
	"cgl/obj"
	"context"
	"fmt"
	"slices"
)

const (
	OpenAi  = "openai"
	Mistral = "mistral"
	Mock    = "mock"
)

var ApiKeyPlatforms = []string{
	OpenAi,
	Mistral,
	Mock,
}

func IsValidApiKeyPlatform(platform string) bool {
	return slices.Contains(ApiKeyPlatforms, platform)
}

type AiPlatform interface {
	GetPlatformInfo() obj.AiPlatform

	// ExecuteAction - blocking, returns structured JSON (plotOutline in Message, statusFields, imagePrompt)
	// For system messages (first call), action.Message contains the system prompt/instructions
	// gameSchema is the JSON schema enforcing exact status field names, built by the caller.
	ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage, gameSchema map[string]interface{}) (obj.TokenUsage, error)

	// ExpandStory - async/streaming, expands plotOutline to full narrative text
	// Streams text chunks to responseStream, updates response.Message with full text when done
	ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error)

	// GenerateImage - async/streaming, generates image from response.ImagePrompt
	// Streams partial images to responseStream, updates response.Image with final image when done
	GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error

	// GenerateAudio - async/streaming, generates audio narration from text via TTS
	// Streams audio chunks to responseStream, updates response.Audio with final audio when done
	// Only supported on OpenAI (max tier); other platforms return nil (no-op).
	GenerateAudio(ctx context.Context, session *obj.GameSession, text string, responseStream *stream.Stream) ([]byte, error)

	// Translate - blocking, translates a set of language files to a target language
	// Returns the translated JSON as a stringified object and token usage
	Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error)

	// ListModels - blocking, retrieves all available models from the platform API
	ListModels(ctx context.Context, apiKey string) ([]obj.AiModel, error)

	// GenerateTheme - blocking, generates a visual theme JSON based on game description
	// Returns the raw JSON string response from the AI and token usage
	GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error)
}

func GetAiPlatformInfos() []obj.AiPlatform {
	platforms := []obj.AiPlatform{
		functional.First(getAiPlatform(OpenAi)).GetPlatformInfo(),
		functional.First(getAiPlatform(Mistral)).GetPlatformInfo(),
		functional.First(getAiPlatform(Mock)).GetPlatformInfo(),
	}

	// Set SupportsApiKey flag for each platform
	for i := range platforms {
		platforms[i].SupportsApiKey = IsValidApiKeyPlatform(platforms[i].ID)
	}

	return platforms
}

func getAiPlatform(platformName string) (AiPlatform, error) {
	var platform AiPlatform
	switch platformName {
	case OpenAi:
		platform = &openai.OpenAiPlatform{}
	case Mistral:
		platform = &mistral.MistralPlatform{}
	case Mock:
		platform = &mock.MockPlatform{}
	default:
		return nil, fmt.Errorf("unknown ai platform: %s", platformName)
	}
	return platform, nil
}

// GetAiPlatform returns the AI platform for the given platform name.
// Returns error if platform is unknown.
func GetAiPlatform(platformName string) (AiPlatform, error) {
	return getAiPlatform(platformName)
}
