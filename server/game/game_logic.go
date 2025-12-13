package game

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/game/ai"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/obj"

	"github.com/google/uuid"
)

// CreateSession creates a new game session for a user.
// If shareID is uuid.Nil, the user's default API key share will be used.
// If model is empty, the platform's default model will be used.
// Returns *obj.HTTPError (which implements the standard error interface) for client-facing errors with appropriate status codes.
func CreateSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, shareID uuid.UUID, aiModel string) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	// TODO: resolving keys is more complex - we also have sponsored public keys, workshop keys, institution keys... so we need more logic to figure out which key to use
	// For now, we'll just use the provided share or default, but in the future we should implement proper key resolution logic

	// Resolve share: use provided share, or fall back to user's default
	if shareID == uuid.Nil {
		defaultShareID, err := db.GetUserDefaultApiKeyShare(ctx, userID)
		if err != nil {
			return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get default API key: " + err.Error()}
		}
		if defaultShareID == nil {
			return nil, nil, &obj.HTTPError{StatusCode: 400, Message: "No API key share provided and no default set. Use 'apikey default <share-id>' to set a default."}
		}
		shareID = *defaultShareID
	}

	// Get the share and check if user is directly included
	share, err := db.GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "API key share not found: " + err.Error()}
	}

	// Check if user is directly included in the share (not via workshop/institution for now)
	if share.User == nil || share.User.ID != userID {
		return nil, nil, &obj.HTTPError{StatusCode: 403, Message: "You don't have direct access to this API key share"}
	}

	// Get the game
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found: " + err.Error()}
	}

	// Parse game template to get system message
	systemMessage, err := templates.GetTemplate(game)
	if err != nil {
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get game template: " + err.Error()}
	}

	// Get AI platform and resolve model
	_, aiModel, err = ai.GetAiPlatform(share.ApiKey.Platform, aiModel)
	if err != nil {
		return nil, nil, &obj.HTTPError{StatusCode: 400, Message: err.Error()}
	}

	// Create session object
	session := &obj.GameSession{
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
	session, err = db.CreateGameSession(ctx, session)
	if err != nil {
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create session: " + err.Error()}
	}

	// First action is a system message containing the game instructions
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)
	return session, response, httpErr
}

// Spawn async AI call
func executeActionAsync() {

}

// DoSessionAction returns an response that doesn't contain the actual text, but a session message with an id to stream the result in a follow up call via SSE
func DoSessionAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (response *obj.GameSessionMessage, httpErr *obj.HTTPError) {
	const failedAction = "failed doing session action"
	if session == nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: session is nil", failedAction)}
	}
	if session.ApiKey == nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s %s: session has no api key object", failedAction, session.ID)}
	}

	platform, _, err := ai.GetAiPlatform(session.AiPlatform, session.AiModel)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s %s: %w", failedAction, session.ID, err)}
	}

	// Create placeholder message with Stream=true (client will connect to SSE)
	response, err = db.CreateStreamingMessage(ctx, session.UserID, session.ID, obj.GameSessionMessageTypeGame)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: failed to create streaming message: %v", failedAction, err)}
	}

	// Create stream for SSE
	responseStream := stream.Get().Create(ctx, response)

	// Spawn async AI call - result will be available via SSE stream
	go func() {
		var err error
		err = platform.ExecuteAction(context.Background(), session, action, response)
		if err != nil {
			responseStream.SendError(err.Error())
			// Mark message as error type
			response.Type = "error"
			response.Message = err.Error()
			response.Stream = false
			_ = db.UpdateGameSessionMessage(context.Background(), *response)
			return
		}

		// Stream the result
		responseStream.SendText(response.Message)
		responseStream.SendDone()

		// Update the message in DB with final content
		response.Stream = false
		if err = db.UpdateGameSessionMessage(context.Background(), *response); err != nil {
			fmt.Printf("%s: WARNING: failed to update game session message: %v\n", failedAction, err)
		}
	}()

	return response, nil
}
