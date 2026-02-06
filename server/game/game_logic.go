package game

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cgl/db"
	"cgl/functional"
	"cgl/game/ai"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// extractAIErrorCode tries to extract an error code from an AI error message.
// Uses simple keyword matching to identify common OpenAI error types.
func extractAIErrorCode(err error) string {
	if err == nil {
		return ""
	}
	errStr := strings.ToLower(err.Error())

	// Map error keywords to our error codes
	switch {
	case strings.Contains(errStr, "invalid_api_key"):
		return obj.ErrCodeInvalidApiKey
	case strings.Contains(errStr, "billing_not_active"):
		return obj.ErrCodeBillingNotActive
	case strings.Contains(errStr, "organization_verification_required"):
		return obj.ErrCodeOrgVerificationRequired
	case strings.Contains(errStr, "rate_limit") || strings.Contains(errStr, "rate limit"):
		return obj.ErrCodeRateLimitExceeded
	case strings.Contains(errStr, "insufficient_quota") || strings.Contains(errStr, "quota"):
		return obj.ErrCodeInsufficientQuota
	case strings.Contains(errStr, "content_policy") || strings.Contains(errStr, "content_filter"):
		return obj.ErrCodeContentFiltered
	default:
		// For any other AI API error, return generic AI error
		return obj.ErrCodeAiError
	}
}

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
		return nil, nil, &obj.HTTPError{StatusCode: 400, Message: "No API key share provided."}
	}

	// Get the share - GetApiKeyShareByID already checks access permissions (including institution/workshop shares)
	log.Debug("resolving API key share", "share_id", shareID)
	share, err := db.GetApiKeyShareByID(ctx, userID, shareID)
	if err != nil {
		log.Debug("API key share not found", "share_id", shareID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "API key share not found: " + err.Error()}
	}
	log.Info("using API key for session", "key_name", share.ApiKey.Name, "key_platform", share.ApiKey.Platform, "share_id", shareID)

	// Get the game
	log.Debug("loading game", "game_id", gameID)
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("game not found", "game_id", gameID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found: " + err.Error()}
	}

	// Delete any existing sessions for this user+game (restart scenario)
	log.Debug("deleting existing sessions", "user_id", userID, "game_id", gameID)
	if err := db.DeleteUserGameSessions(ctx, userID, gameID); err != nil {
		log.Debug("failed to delete existing sessions", "error", err)
		// Non-fatal - continue with session creation
	}

	// Parse game template to get system message
	log.Debug("parsing game template", "game_id", gameID, "game_name", game.Name)
	systemMessage, err := templates.GetTemplate(game)
	if err != nil {
		log.Debug("failed to get game template", "game_id", gameID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get game template: " + err.Error()}
	}

	// Validate AI platform
	log.Debug("resolving AI platform", "platform", share.ApiKey.Platform, "requested_model", aiModel)
	_, err = ai.GetAiPlatform(share.ApiKey.Platform)
	if err != nil {
		log.Debug("failed to get AI platform", "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 400, Message: err.Error()}
	}
	log.Debug("AI platform resolved", "platform", share.ApiKey.Platform)

	// Get user to check language preference
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("failed to get user for language preference", "user_id", userID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get user: " + err.Error()}
	}

	// Create temporary session object for theme generation (needs ApiKey)
	tempSession := &obj.GameSession{
		GameID:       game.ID,
		GameName:     game.Name,
		UserID:       userID,
		ApiKeyID:     &share.ApiKey.ID,
		ApiKey:       share.ApiKey,
		AiPlatform:   share.ApiKey.Platform,
		AiModel:      aiModel,
		ImageStyle:   templates.ImageStyleOrDefault(game.ImageStyle),
		StatusFields: game.StatusFields,
	}

	// Run theme generation and game translation in parallel
	var wg sync.WaitGroup
	var theme *obj.GameTheme
	var translatedGame *obj.Game

	// Start theme generation
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		log.Debug("generating visual theme", "game_id", gameID, "game_name", game.Name)
		t, err := GenerateTheme(ctx, tempSession, game)
		if err != nil {
			log.Warn("failed to generate theme, using default", "error", err, "seconds", time.Since(start).Seconds())
		} else {
			log.Debug("theme generated successfully", "preset", t.Preset, "seconds", time.Since(start).Seconds())
			theme = t
		}
	}()

	// Start game translation if user language is not English
	var fieldNameMap map[string]string
	if user.Language != "" && user.Language != "en" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			log.Debug("translating game to user language", "game_id", gameID, "target_lang", user.Language)
			translated, fnMap, err := TranslateGame(ctx, tempSession, game, user.Language)
			if err != nil {
				log.Warn("failed to translate game, using original", "target_lang", user.Language, "error", err, "seconds", time.Since(start).Seconds())
			} else {
				log.Debug("game translated successfully", "target_lang", user.Language, "seconds", time.Since(start).Seconds())
				translatedGame = translated
				fieldNameMap = fnMap
			}
		}()
	}

	// Wait for both operations to complete
	wg.Wait()

	// Use translated game if available
	if translatedGame != nil {
		game = translatedGame
		// Re-generate system message with translated game
		systemMessage, err = templates.GetTemplate(game)
		if err != nil {
			log.Debug("failed to get template from translated game", "error", err)
			return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get game template: " + err.Error()}
		}

		// Rewrite theme statusEmojis keys from original to translated field names
		if theme != nil && theme.Override != nil && len(theme.Override.StatusEmojis) > 0 && len(fieldNameMap) > 0 {
			translatedEmojis := make(map[string]string, len(theme.Override.StatusEmojis))
			for originalName, emoji := range theme.Override.StatusEmojis {
				if translatedName, ok := fieldNameMap[originalName]; ok {
					translatedEmojis[translatedName] = emoji
				} else {
					translatedEmojis[originalName] = emoji
				}
			}
			theme.Override.StatusEmojis = translatedEmojis
			log.Debug("rewrote theme statusEmojis for translated field names", "mapping", fieldNameMap)
		}
	}

	// Persist to database with theme
	log.Debug("persisting session to database")
	session, err := db.CreateGameSession(ctx, userID, game.ID, share.ApiKey.ID, aiModel, nil, theme)
	if err != nil {
		log.Debug("failed to create session in DB", "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create session: " + err.Error()}
	}
	log.Debug("session created", "session_id", session.ID)

	// Attach API key for response
	session.ApiKey = share.ApiKey

	// First action is a system message containing the game instructions
	log.Debug("executing initial system action", "session_id", session.ID)
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)
	if httpErr != nil {
		// Clean up: delete session if first action failed (0 messages = nothing to preserve)
		log.Debug("initial action failed, deleting empty session", "session_id", session.ID, "error", httpErr.Message)
		if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
			log.Warn("failed to delete empty session after error", "session_id", session.ID, "error", delErr)
		}
		return nil, nil, httpErr
	}
	response.PromptStatusUpdate = functional.Ptr(systemMessage)
	return session, response, nil
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
	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		log.Debug("failed to get AI platform", "session_id", session.ID, "error", err)
		return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s %s: %v", failedAction, session.ID, err)}
	}

	// Store player/system action message (skip for system messages which are just prompts)
	// Track the player message so we can delete it if the AI action fails
	var playerMessageID *uuid.UUID
	if action.Type == obj.GameSessionMessageTypePlayer {
		log.Debug("storing player action message", "session_id", session.ID)
		playerMsg, err := db.CreateGameSessionMessage(ctx, session.UserID, action)
		if err != nil {
			log.Debug("failed to store player action", "session_id", session.ID, "error", err)
			return nil, &obj.HTTPError{StatusCode: 500, Message: fmt.Sprintf("%s: failed to store player action: %v", failedAction, err)}
		}
		playerMessageID = &playerMsg.ID
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
		return db.UpdateGameSessionMessageImage(context.Background(), session.UserID, messageID, imageData)
	})

	// Phase 1: ExecuteAction (blocking) - get structured JSON with plotOutline, statusFields, imagePrompt
	log.Debug("executing AI action", "session_id", session.ID, "message_id", response.ID)
	if err = platform.ExecuteAction(ctx, session, action, response); err != nil {
		log.Debug("ExecuteAction failed", "session_id", session.ID, "error", err)
		responseStream.SendError(err.Error())

		// Delete the placeholder message instead of saving empty/error content
		if delErr := db.DeleteGameSessionMessage(ctx, response.ID); delErr != nil {
			log.Warn("failed to delete placeholder message after error", "message_id", response.ID, "error", delErr)
		}

		// Delete the player message too - it was never processed by the AI
		if playerMessageID != nil {
			if delErr := db.DeleteGameSessionMessage(ctx, *playerMessageID); delErr != nil {
				log.Warn("failed to delete player message after error", "message_id", *playerMessageID, "error", delErr)
			}
		}

		// Extract AI error code and clear API key for key-related errors
		errorCode := extractAIErrorCode(err)
		if errorCode == obj.ErrCodeBillingNotActive || errorCode == obj.ErrCodeInvalidApiKey || errorCode == obj.ErrCodeInsufficientQuota {
			log.Debug("clearing session API key due to key error", "session_id", session.ID, "error_code", errorCode)
			if clearErr := db.ClearGameSessionApiKey(ctx, session.ID); clearErr != nil {
				log.Warn("failed to clear session API key", "session_id", session.ID, "error", clearErr)
			}
		}

		return nil, obj.NewHTTPErrorWithCode(500, errorCode, fmt.Sprintf("%s: ExecuteAction failed: %v", failedAction, err))
	}
	log.Debug("ExecuteAction completed", "session_id", session.ID, "has_image_prompt", response.ImagePrompt != nil)
	// Set PromptStatusUpdate to the full JSON input sent to the AI
	response.PromptStatusUpdate = functional.Ptr(action.ToAiJSON())
	response.PromptExpandStory = functional.Ptr(templates.PromptNarratePlotOutline)
	response.PromptImageGeneration = response.ImagePrompt

	// Save the structured response (plotOutline in Message, statusFields, imagePrompt)
	// This is returned to client immediately
	_ = db.UpdateGameSessionMessage(ctx, session.UserID, *response)

	// Capture values before goroutines to avoid race conditions
	messageID := response.ID

	// Phase 2 & 3: Run ExpandStory and GenerateImage in parallel (async)
	log.Debug("starting async story expansion and image generation", "session_id", session.ID)
	go func() {
		log.Debug("starting ExpandStory", "session_id", session.ID, "message_id", messageID)
		// ExpandStory streams text and updates response.Message with full narrative
		if err := platform.ExpandStory(context.Background(), session, response, responseStream); err != nil {
			log.Warn("ExpandStory failed", "session_id", session.ID, "error", err)
		} else {
			log.Debug("ExpandStory completed", "session_id", session.ID, "message_length", len(response.Message))
		}

		// Update DB with full text (replaces plotOutline)
		response.Stream = false
		if err := db.UpdateGameSessionMessage(context.Background(), session.UserID, *response); err != nil {
			log.Warn("failed to update message after ExpandStory", "session_id", session.ID, "error", err)
		}

		// Update session with new AI state (response IDs for conversation continuity)
		if err := db.UpdateGameSessionAiSession(context.Background(), session.UserID, session.ID, session.AiSession); err != nil {
			log.Warn("failed to update session AI state", "session_id", session.ID, "error", err)
		}
	}()

	go func() {
		log.Debug("starting GenerateImage", "session_id", session.ID, "message_id", messageID, "has_prompt", response.ImagePrompt != nil)
		// GenerateImage streams partial images and updates response.Image with final
		// Note: Image is saved to DB inside stream.SendImage when isDone=true
		// Use captured imagePrompt to avoid race condition with response pointer
		if response.ImagePrompt == nil || *response.ImagePrompt == "" {
			log.Debug("no image prompt, skipping image generation")
			return
		}
		if session.ImageStyle == templates.ImageStyleNoImage {
			log.Debug("image generation disabled (NO_IMAGE)")
			return
		}
		if err := platform.GenerateImage(context.Background(), session, response, responseStream); err != nil {
			log.Warn("GenerateImage failed", "session_id", session.ID, "error", err)

			// Check for errors that require action
			errorCode := extractAIErrorCode(err)
			switch errorCode {
			case obj.ErrCodeOrgVerificationRequired:
				// Mark session as having unverified organization (user needs to verify with OpenAI)
				if dbErr := db.UpdateGameSessionOrganisationUnverified(context.Background(), session.ID, true); dbErr != nil {
					log.Warn("failed to update session org unverified status", "error", dbErr)
				}
			case obj.ErrCodeBillingNotActive, obj.ErrCodeInvalidApiKey, obj.ErrCodeInsufficientQuota:
				// API key is no longer valid - clear it so user can select a new one
				log.Info("clearing invalid API key from session", "session_id", session.ID, "error_code", errorCode)
				if dbErr := db.ClearGameSessionApiKey(context.Background(), session.ID); dbErr != nil {
					log.Warn("failed to clear session API key", "error", dbErr)
				}
			}
		} else {
			log.Debug("GenerateImage completed", "session_id", session.ID, "image_size", len(response.Image))
		}
	}()

	return response, nil
}
