package ai

import (
	"cgl/obj"

	"github.com/google/uuid"
)

var ApiKeyPlatforms = []string{"openai", "mistral", "mock"}

type AiPlatform interface {
	InitGameSession(session *obj.GameSession) (err error)
	ExecuteAction(session *obj.GameSession, action obj.GameActionInput) (response *obj.GameActionOutput, err error)
}

// NewSession creates a new session object, but doesn't store it in the db or push it to any ai platform.
func NewSession(game obj.Game, userID uuid.UUID, apiKeyID uuid.UUID) *obj.GameSession {

	return &obj.GameSession{
		ID:              uuid.New(),
		GameID:          game.ID,
		GameName:        game.Name,
		GameDescription: game.Description,
		UserID:          userID,
		ApiKeyID:        apiKeyID,
		ImageStyle:      game.ImageStyle,
		StatusFields:    parseStatusFields(game.StatusFields),
		ModelSession:    "{}",
	}
}

func parseStatusFields(statusFieldsJSON string) []obj.StatusField {
	// TODO: parse JSON status fields from game config
	return []obj.StatusField{}
}
