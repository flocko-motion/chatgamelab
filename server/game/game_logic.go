package game

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/game/ai"
	"cgl/game/stream"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// CreateSession creates a new game session for a user.
// If shareID is uuid.Nil, the user's default API key share will be used.
// If model is empty, the platform's default model will be used.
// Returns *obj.HTTPError (which implements the standard error interface) for client-facing errors with appropriate status codes.
func CreateSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, shareID uuid.UUID, aiModel string) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	log.Debug("creating session", "user_id", userID, "game_id", gameID, "share_id", shareID, "ai_model", aiModel)

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
	log.Debug("resolving API key share", "share_id", shareID)
	share, err := db.GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		log.Debug("API key share not found", "share_id", shareID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "API key share not found: " + err.Error()}
	}

	// Check if user is directly included in the share (not via workshop/institution for now)
	if share.User == nil || share.User.ID != userID {
		return nil, nil, &obj.HTTPError{StatusCode: 403, Message: "You don't have direct access to this API key share"}
	}

	// Get the game
	log.Debug("loading game", "game_id", gameID)
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("game not found", "game_id", gameID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found: " + err.Error()}
	}

	// Parse game template to get system message
	log.Debug("parsing game template", "game_id", gameID, "game_name", game.Name)
	systemMessage, err := GetTemplate(game)
	if err != nil {
		log.Debug("failed to get game template", "game_id", gameID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get game template: " + err.Error()}
	}

	// Get AI platform and resolve model
	log.Debug("resolving AI platform", "platform", share.ApiKey.Platform, "requested_model", aiModel)
	_, aiModel, err = ai.GetAiPlatform(share.ApiKey.Platform, aiModel)
	if err != nil {
		log.Debug("failed to get AI platform", "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 400, Message: err.Error()}
	}
	log.Debug("AI platform resolved", "platform", share.ApiKey.Platform, "model", aiModel)

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
	log.Debug("persisting session to database")
	session, err = db.CreateGameSession(ctx, session)
	if err != nil {
		log.Debug("failed to create session in DB", "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create session: " + err.Error()}
	}
	log.Debug("session created", "session_id", session.ID)

	// First action is a system message containing the game instructions
	log.Debug("executing initial system action", "session_id", session.ID)
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)
	return session, response, httpErr
}

// DoSessionAction orchestrates the AI response generation:
// 1. ExecuteAction (blocking) - gets structured JSON with plotOutline, statusFields, imagePrompt
// 2. ExpandStory (async/streaming) - expands plotOutline to full narrative, streams to client
// 3. GenerateImage (async/streaming) - generates image from imagePrompt, streams partial images
// Returns immediately after ExecuteAction with the response containing plotOutline.
// Client connects to SSE to receive streamed text and image updates.
func DoSessionAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (response *obj.GameSessionMessage, httpErr *obj.HTTPError) {
	const failedAction = "failed doing session action"
	log.Debug("starting session action", "session_id", session.ID, "action_type", action.Type)

	if session == nil {
		log.Error("session action failed: session is nil")
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: session is nil", failedAction)}
	}
	if session.ApiKey == nil {
		log.Error("session action failed: no API key", "session_id", session.ID)
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s %s: session has no api key object", failedAction, session.ID)}
	}

	log.Debug("getting AI platform", "platform", session.AiPlatform, "model", session.AiModel)
	platform, _, err := ai.GetAiPlatform(session.AiPlatform, session.AiModel)
	if err != nil {
		log.Debug("failed to get AI platform", "session_id", session.ID, "error", err)
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s %s: %v", failedAction, session.ID, err)}
	}

	// Store player/system action message (skip for system messages which are just prompts)
	if action.Type == obj.GameSessionMessageTypePlayer {
		log.Debug("storing player action message", "session_id", session.ID)
		if _, err = db.CreateGameSessionMessage(ctx, session.UserID, action); err != nil {
			log.Debug("failed to store player action", "session_id", session.ID, "error", err)
			return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: failed to store player action: %v", failedAction, err)}
		}
	}

	// Create placeholder message with Stream=true (client will connect to SSE)
	log.Debug("creating streaming message", "session_id", session.ID)
	response, err = db.CreateStreamingMessage(ctx, session.UserID, session.ID, obj.GameSessionMessageTypeGame)
	if err != nil {
		log.Debug("failed to create streaming message", "session_id", session.ID, "error", err)
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: failed to create streaming message: %v", failedAction, err)}
	}
	log.Debug("streaming message created", "message_id", response.ID)

	// Create stream for SSE with ImageSaver to persist image before signaling done
	responseStream := stream.Get().Create(ctx, response, func(messageID uuid.UUID, imageData []byte) error {
		return db.UpdateGameSessionMessageImage(context.Background(), messageID, imageData)
	})

	// Phase 1: ExecuteAction (blocking) - get structured JSON with plotOutline, statusFields, imagePrompt
	log.Debug("executing AI action", "session_id", session.ID, "message_id", response.ID)
	if err = platform.ExecuteAction(ctx, session, action, response); err != nil {
		log.Debug("ExecuteAction failed", "session_id", session.ID, "error", err)
		responseStream.SendError(err.Error())
		response.Type = "error"
		response.Message = err.Error()
		response.Stream = false
		_ = db.UpdateGameSessionMessage(ctx, *response)
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: ExecuteAction failed: %v", failedAction, err)}
	}
	log.Debug("ExecuteAction completed", "session_id", session.ID, "has_image_prompt", response.ImagePrompt != nil)

	// Save the structured response (plotOutline in Message, statusFields, imagePrompt)
	// This is returned to client immediately
	_ = db.UpdateGameSessionMessage(ctx, *response)

	// Phase 2 & 3: Run ExpandStory and GenerateImage in parallel (async)
	log.Debug("starting async story expansion and image generation", "session_id", session.ID)
	go func() {
		log.Debug("starting ExpandStory", "session_id", session.ID, "message_id", response.ID)
		// ExpandStory streams text and updates response.Message with full narrative
		if err := platform.ExpandStory(context.Background(), session, response, responseStream); err != nil {
			log.Warn("ExpandStory failed", "session_id", session.ID, "error", err)
		} else {
			log.Debug("ExpandStory completed", "session_id", session.ID, "message_length", len(response.Message))
		}

		// Update DB with full text (replaces plotOutline)
		response.Stream = false
		if err := db.UpdateGameSessionMessage(context.Background(), *response); err != nil {
			log.Warn("failed to update message after ExpandStory", "session_id", session.ID, "error", err)
		}

		// Update session with new AI state (response IDs for conversation continuity)
		if err := db.UpdateGameSessionAiSession(context.Background(), session.ID, session.AiSession); err != nil {
			log.Warn("failed to update session AI state", "session_id", session.ID, "error", err)
		}
	}()

	go func() {
		log.Debug("starting GenerateImage", "session_id", session.ID, "message_id", response.ID)
		// GenerateImage streams partial images and updates response.Image with final
		// Note: Image is saved to DB inside stream.SendImage when isDone=true
		if err := platform.GenerateImage(context.Background(), session, response, responseStream); err != nil {
			log.Warn("GenerateImage failed", "session_id", session.ID, "error", err)
		} else {
			log.Debug("GenerateImage completed", "session_id", session.ID, "image_size", len(response.Image))
		}
	}()

	return response, nil
}
