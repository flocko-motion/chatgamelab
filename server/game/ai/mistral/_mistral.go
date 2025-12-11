package mistral

import (
	"cgl/obj"
	"context"
)

type MistralPlatform struct{}

func (p *MistralPlatform) InitGameSession(session *obj.GameSession) error {
	// TODO: implement Mistral session initialization
	return nil
}

func (p *MistralPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameActionInput) (*obj.GameActionOutput, error) {
	// TODO: implement Mistral action execution
	return &obj.GameActionOutput{}, nil
}
