package openai

import (
	"cgl/obj"
	"context"
)

type OpenAiPlatform struct{}

func (p *OpenAiPlatform) InitGameSession(session *obj.GameSession) error {
	// TODO: implement OpenAI session initialization
	return nil
}

func (p *OpenAiPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (*obj.GameSessionMessage, error) {
	// TODO: implement OpenAI action execution
	return &obj.GameSessionMessage{}, nil
}
