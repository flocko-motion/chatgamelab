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
	InitGameSession(session *obj.GameSession) (err error)
	ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (response *obj.GameSessionMessage, err error)
}

func GetAiPlatform(name string) (AiPlatform, error) {
	const failedAction = "failed getting ai platform"
	switch name {
	case OpenAi:
		return &openai.OpenAiPlatform{}, nil
	// case Mistral:
	// 	return &mistral.MistralPlatform{}, nil
	case Mock:
		return &mock.MockPlatform{}, nil
	default:
		return nil, fmt.Errorf("%s: unknown ai platform: %s", failedAction, name)
	}
}
