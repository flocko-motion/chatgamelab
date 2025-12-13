package ai

import (
	"cgl/game/ai/mock"
	"cgl/game/ai/openai"
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
	// ExecuteAction executes an action and fills in the response message
	// The msg parameter is pre-created with ID set; this function fills Message, StatusFields, ImagePrompt
	// For system messages (first call), the action.Message contains the system prompt/instructions
	// Caller handles streaming and DB persistence
	ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, msg *obj.GameSessionMessage) error
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
