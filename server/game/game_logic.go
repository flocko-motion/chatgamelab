package game

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/game/ai"
	"cgl/game/templates"
	"cgl/obj"

	"github.com/google/uuid"
)

// CreateSession creates a new game session for a user.
// If shareID is uuid.Nil, the user's default API key share will be used.
// If model is empty, the platform's default model will be used.
// Returns *obj.HTTPError (which implements the standard error interface) for client-facing errors with appropriate status codes.
func CreateSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, shareID uuid.UUID, aiModel string) (*obj.GameSession, *obj.HTTPError) {
	// TODO: resolving keys is more complex - we also have sponsored public keys, workshop keys, institution keys... so we need more logic to figure out which key to use
	// For now, we'll just use the provided share or default, but in the future we should implement proper key resolution logic

	// Resolve share: use provided share, or fall back to user's default
	if shareID == uuid.Nil {
		defaultShareID, err := db.GetUserDefaultApiKeyShare(ctx, userID)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get default API key: " + err.Error()}
		}
		if defaultShareID == nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "No API key share provided and no default set. Use 'apikey default <share-id>' to set a default."}
		}
		shareID = *defaultShareID
	}

	// Get the share and check if user is directly included
	share, err := db.GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 404, Message: "API key share not found: " + err.Error()}
	}

	// Check if user is directly included in the share (not via workshop/institution for now)
	if share.User == nil || share.User.ID != userID {
		return nil, &obj.HTTPError{StatusCode: 403, Message: "You don't have direct access to this API key share"}
	}

	// Get the game
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found: " + err.Error()}
	}

	// Parse game template to get system message
	systemMessage, err := templates.GetTemplate(game)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get game template: " + err.Error()}
	}

	// Get AI platform and resolve model
	aiPlatform, aiModel, err := ai.GetAiPlatform(share.ApiKey.Platform, aiModel)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 400, Message: err.Error()}
	}

	// Create session object
	session := obj.GameSession{
		ID:              uuid.New(),
		GameID:          game.ID,
		GameName:        game.Name,
		GameDescription: game.Description,
		UserID:          userID,
		ApiKeyID:        share.ApiKey.ID,
		ApiKey:          share.ApiKey,
		AiPlatform:      share.ApiKey.Platform,
		AiModel:         aiModel,
		ImageStyle:      game.ImageStyle,
		StatusFields:    game.StatusFields,
		AiSession:       "{}",
	}

	// Persist to database
	if err := db.CreateGameSession(ctx, &session); err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create session: " + err.Error()}
	}

	if err := aiPlatform.InitGameSession(&session, systemMessage); err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to initialize session: " + err.Error()}
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
	platform, _, err := ai.GetAiPlatform(session.AiPlatform, session.AiModel)
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
