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
	// Mistral,
	Mock,
}

func IsValidApiKeyPlatform(platform string) bool {
	return slices.Contains(ApiKeyPlatforms, platform)
}

type AiPlatform interface {
	GetPlatformInfo() obj.AiPlatform

	// ExecuteAction - blocking, returns structured JSON (plotOutline in Message, statusFields, imagePrompt)
	// For system messages (first call), action.Message contains the system prompt/instructions
	ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage) error

	// ExpandStory - async/streaming, expands plotOutline to full narrative text
	// Streams text chunks to responseStream, updates response.Message with full text when done
	ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error

	// GenerateImage - async/streaming, generates image from response.ImagePrompt
	// Streams partial images to responseStream, updates response.Image with final image when done
	GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error

	// Translate - blocking, translates a set of language files to a target language
	// Returns the translated JSON as a stringified object
	Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, error)

	// ListModels - blocking, retrieves all available models from the platform API
	ListModels(ctx context.Context, apiKey string) ([]obj.AiModel, error)
}

func GetAiPlatformInfos() []obj.AiPlatform {
	return []obj.AiPlatform{
		functional.First(getAiPlatform(OpenAi)).GetPlatformInfo(),
		functional.First(getAiPlatform(Mistral)).GetPlatformInfo(),
		functional.First(getAiPlatform(Mock)).GetPlatformInfo(),
	}
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

// GetAiPlatform returns the AI platform and resolves the model.
// If model is empty, returns the platform's default model.
// Returns error if platform is unknown.
// Note: Does not validate model names - allows any model string for flexibility with dev tools.
func GetAiPlatform(platformName, model string) (AiPlatform, string, error) {
	platform, err := getAiPlatform(platformName)
	if err != nil {
		return nil, "", err
	}

	// If no model specified, use the first hardcoded model as default
	if model == "" {
		info := platform.GetPlatformInfo()
		if len(info.Models) == 0 {
			return nil, "", fmt.Errorf("no models available for platform %s", info.Name)
		}
		model = info.Models[0].ID
	}

	// Return platform and model without validation
	// This allows dev tools to use any model name, including new ones not in hardcoded list
	return platform, model, nil
}
