package ai

import (
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
}

// GetAiPlatform returns the AI platform and resolves the model.
// If model is empty, returns the platform's default model.
// Returns error if platform is unknown or model is invalid.
func GetAiPlatform(platformName, model string) (AiPlatform, string, error) {
	var platform AiPlatform
	switch platformName {
	case OpenAi:
		platform = &openai.OpenAiPlatform{}
	// case Mistral:
	// 	platform = &mistral.MistralPlatform{}
	case Mock:
		platform = &mock.MockPlatform{}
	default:
		return nil, "", fmt.Errorf("unknown ai platform: %s", platformName)
	}

	info := platform.GetPlatformInfo()

	if model == "" {
		if len(info.Models) == 0 {
			return nil, "", fmt.Errorf("no models available for platform %s", info.Name)
		}
		model = info.Models[0].ID
	} else {
		valid := false
		for _, m := range info.Models {
			if m.ID == model {
				valid = true
				break
			}
		}
		if !valid {
			return nil, "", fmt.Errorf("invalid model '%s' for platform %s", model, info.Name)
		}
	}

	return platform, model, nil
}
