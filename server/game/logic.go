package game

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/game/ai"
	"cgl/obj"

	"github.com/google/uuid"
)

// CreateSession creates a new game session for a user
func CreateSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, apiKeyID uuid.UUID) (*obj.GameSession, error) {
	const failedAction = "failed creating session"
	// Get the game
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get game: %w", failedAction, err)
	}

	// get the api key
	apiKey, err := db.GetApiKeyByID(ctx, &userID, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get api key: %w", failedAction, err)
	}

	// Create session object
	session := obj.GameSession{
		ID:              uuid.New(),
		GameID:          game.ID,
		GameName:        game.Name,
		GameDescription: game.Description,
		UserID:          userID,
		ApiKeyID:        apiKey.ID,
		ApiKey:          apiKey,
		ImageStyle:      game.ImageStyle,
		StatusFields:    game.StatusFields,
		ModelSession:    "{}",
	}

	// Persist to database
	if err := db.CreateGameSession(ctx, &session); err != nil {
		return nil, fmt.Errorf("%s: failed to create session: %w", failedAction, err)
	}

	// Initialize the session
	ai, err := ai.GetAiPlatform(session.ApiKey.Platform)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get ai platform: %w", failedAction, err)
	}
	if err := ai.InitGameSession(&session); err != nil {
		return nil, fmt.Errorf("%s: failed to initialize session: %w", failedAction, err)
	}

	return &session, nil
}

func DoSessionAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (response *obj.GameSessionMessage, err error) {
	const failedAction = "failed doing session action"
	if session == nil {
		return nil, fmt.Errorf("%s: session is nil", failedAction)
	}
	if session.ApiKey == nil {
		return nil, fmt.Errorf("%s %s: session has no api key object", failedAction, session.ID)
	}
	platform, err := ai.GetAiPlatform(session.ApiKey.Platform)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", failedAction, session.ID, err)
	}
	// write action to db
	if _, err := db.CreateGameSessionMessage(ctx, session.UserID, action); err != nil {
		return nil, fmt.Errorf("%s %s: failed to create session action: %w", failedAction, session.ID, err)
	}
	// Execute the action
	actionResult, err := platform.ExecuteAction(ctx, session, action)
	if err != nil {
		return nil, fmt.Errorf("%s %s: failed to execute session action: %w", failedAction, session.ID, err)
	}
	// write action to db
	msgResult, err := db.CreateGameSessionMessage(ctx, session.UserID, obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeStory,
		Message:       actionResult.Message,
		StatusFields:  actionResult.StatusFields,
		ImagePrompt:   actionResult.ImagePrompt,
		Image:         actionResult.Image,
	})
	if err != nil {
		return nil, fmt.Errorf("%s %s: failed to create session action: %w", failedAction, session.ID, err)
	}

	return msgResult, nil

}
