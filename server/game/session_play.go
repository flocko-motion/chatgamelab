package game

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"cgl/db"
	"cgl/functional"
	"cgl/game/ai"
	"cgl/game/imagecache"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/lang"
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
	case strings.Contains(errStr, "previous_response_not_found"):
		return obj.ErrCodePreviousResponseNotFound
	case strings.Contains(errStr, "invalid_json_schema") || strings.Contains(errStr, "invalid schema"):
		return obj.ErrCodeInvalidJsonSchema
	default:
		// For any other AI API error, return generic AI error
		return obj.ErrCodeAiError
	}
}

// isKeyRelatedError returns true if the error code indicates the API key itself
// is broken (not a transient or content error). These errors warrant retrying
// with a fallback key.
func isKeyRelatedError(errorCode string) bool {
	switch errorCode {
	case obj.ErrCodeInvalidApiKey, obj.ErrCodeBillingNotActive, obj.ErrCodeInsufficientQuota:
		return true
	default:
		return false
	}
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

	// Acquire per-session lock to serialize AI calls.
	// Mistral's Conversations API rejects concurrent requests (409), and even
	// for OpenAI the ExpandStory response ID must be persisted before the next
	// ExecuteAction can reference it for conversation continuity.
	unlock := sessionLocks.Lock(session.ID)
	// unlock is passed to the ExpandStory goroutine below; if we return early
	// (error paths), we must unlock here.
	earlyReturn := true
	defer func() {
		if earlyReturn {
			unlock()
		}
	}()

	if session == nil {
		log.Error("session action failed: session is nil")
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, fmt.Sprintf("%s: session is nil", failedAction))
	}
	if session.ApiKey == nil {
		log.Error("session action failed: no API key", "session_id", session.ID)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeInvalidApiKey, "Session has no API key. Please select a new API key.")
	}

	log.Debug("getting AI platform", "platform", session.AiPlatform, "model", session.AiModel)
	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		log.Debug("failed to get AI platform", "session_id", session.ID, "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeInvalidPlatform, fmt.Sprintf("AI platform error: %v", err))
	}

	// Phase 0a: Transcribe audio input to text (if player sent voice instead of text)
	// Done before storing the action message so the DB record already has the transcribed text.
	var transcription string // populated if audio was transcribed, returned to client
	if action.Type == obj.GameSessionMessageTypePlayer && action.AudioBase64 != "" && action.AudioMimeType != "" {
		audioData, decErr := base64.StdEncoding.DecodeString(action.AudioBase64)
		if decErr != nil {
			log.Warn("failed to decode audio base64", "session_id", session.ID, "error", decErr)
		} else {
			log.Debug("transcribing player audio input", "session_id", session.ID, "audio_bytes", len(audioData), "mime_type", action.AudioMimeType)
			transcribed, transcribeErr := platform.TranscribeAudio(ctx, session.ApiKey.Key, audioData, action.AudioMimeType)
			if transcribeErr != nil {
				log.Warn("audio transcription failed, falling back to empty message", "session_id", session.ID, "error", transcribeErr)
				action.Message = `"..."`
			} else {
				log.Debug("audio transcribed", "session_id", session.ID, "text", transcribed)
				action.Message = transcribed
				transcription = transcribed
			}
		}
	}

	// TWO-PHASE INITIALIZATION: Detect opening scene generation
	// Frontend sends system action with "init" message to trigger opening scene
	messageCount, countErr := db.CountGameSessionMessages(ctx, session.ID)
	if countErr != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to count session messages")
	}

	isOpeningScene := messageCount == 1 && strings.TrimSpace(action.Message) == "init"
	var actionMessageID *uuid.UUID

	if isOpeningScene {
		log.Debug("detected opening scene generation (system action 'init')", "session_id", session.ID)
		// Load the system message that was created during session creation
		// It contains the translated scenario and game instructions
		systemMsg, loadErr := db.GetLatestGameSessionMessage(ctx, session.UserID, session.ID)
		if loadErr != nil || systemMsg == nil || systemMsg.Type != obj.GameSessionMessageTypeSystem {
			return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to load system message for opening scene")
		}
		// Use the existing system message (with full content) instead of the init trigger
		// Don't store the init trigger - the system message already exists
		action = *systemMsg
		actionMessageID = &systemMsg.ID
		log.Debug("opening scene: using existing system message", "session_id", session.ID, "message_length", len(action.Message))
	} else {
		// Store the action message (player or system) so it appears in session history.
		// Track the message ID so we can delete it if the AI action fails.
		actionMsg, err := db.CreateGameSessionMessage(ctx, session.UserID, action)
		if err != nil {
			return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to store action message")
		}
		actionMessageID = &actionMsg.ID
	}

	// Create placeholder message with Stream=true (client will connect to SSE)
	response, err = db.CreateStreamingMessage(ctx, session.UserID, session.ID, obj.GameSessionMessageTypeGame)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create streaming message")
	}

	// Tag the AI response with the API key source type (shown in AI Insight panel)
	response.ApiKeyType = session.ApiKeyType

	// Attach transcription to the response so the client can display what was recognized
	if transcription != "" {
		response.Transcription = transcription
	}

	// Create stream for SSE with ImageSaver and AudioSaver to persist before signaling done
	responseStream := stream.Get().Create(ctx, response,
		func(messageID uuid.UUID, imageData []byte) error {
			return db.UpdateGameSessionMessageImage(context.Background(), session.UserID, messageID, imageData)
		},
		func(messageID uuid.UUID, audioData []byte) error {
			return db.UpdateGameSessionMessageAudio(context.Background(), session.UserID, messageID, audioData)
		},
	)

	// Build game-specific JSON schema that enforces exact status field names
	gameSchema := templates.BuildResponseSchema(session.StatusFields)
	gameSchemaJSON, _ := json.Marshal(gameSchema)

	// Phase 0b: Rephrase player input in third person with uncertain outcome (best-effort)
	// Skip if the key's last usage failed — avoids wasting a round-trip on a known-broken key
	keyUsable := session.ApiKey != nil && (session.ApiKey.LastUsageSuccess == nil || *session.ApiKey.LastUsageSuccess)
	if action.Type == obj.GameSessionMessageTypePlayer && keyUsable {
		prompt := fmt.Sprintf(templates.PromptObjectivizePlayerInput, lang.GetLanguageName(session.Language), action.Message)
		if rephrased, err := platform.ToolQuery(ctx, session.ApiKey.Key, prompt); err != nil {
			log.Warn("ToolQuery rephrasing failed, using original input", "session_id", session.ID, "error", err)
			// Mark key as failed if ToolQuery hit a key-related error (e.g. insufficient_quota)
			if errCode := extractAIErrorCode(err); isKeyRelatedError(errCode) {
				db.UpdateApiKeyLastUsageSuccess(ctx, session.ApiKey.ID, false, errCode)
			}
		} else {
			log.Debug("player input rephrased", "session_id", session.ID, "original", action.Message, "rephrased", rephrased)
			action.Message = rephrased
		}
	}

	// Phase 1: ExecuteAction (blocking) - get structured JSON with plotOutline, statusFields, imagePrompt
	log.Debug("executing AI action", "session_id", session.ID, "message_id", response.ID)
	usage, err := platform.ExecuteAction(ctx, session, action, response, gameSchema)
	if err != nil {
		log.Debug("ExecuteAction failed", "session_id", session.ID, "error", err)
		// Extract AI error code and track failure with the specific code
		errorCode := extractAIErrorCode(err)
		// Track API key usage failure with the specific error code
		if session.ApiKey != nil {
			db.UpdateApiKeyLastUsageSuccess(ctx, session.ApiKey.ID, false, errorCode)
		}
		responseStream.SendError(errorCode, err.Error())

		// Delete the placeholder message instead of saving empty/error content
		if delErr := db.DeleteGameSessionMessage(ctx, response.ID); delErr != nil {
			log.Warn("failed to delete placeholder message after error", "message_id", response.ID, "error", delErr)
		}

		// Delete the action message too - it was never processed by the AI
		if delErr := db.DeleteGameSessionMessage(ctx, *actionMessageID); delErr != nil {
			log.Warn("failed to delete action message after error", "message_id", *actionMessageID, "error", delErr)
		}
		if errorCode == obj.ErrCodeBillingNotActive || errorCode == obj.ErrCodeInvalidApiKey || errorCode == obj.ErrCodeInsufficientQuota {
			log.Debug("clearing session API key due to key error", "session_id", session.ID, "error_code", errorCode)
			if clearErr := db.ClearGameSessionApiKey(ctx, session.ID); clearErr != nil {
				log.Warn("failed to clear session API key", "session_id", session.ID, "error", clearErr)
			}

			// Auto-remove sponsorship if the failing key was a sponsored game key
			if removed := autoRemoveSponsorshipOnKeyFailure(ctx, session.GameID, session.ApiKey.ID); removed {
				return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeSponsoredApiKeyNotWorking, fmt.Sprintf("Sponsored API key is not working: %v", err))
			}
		}

		return nil, obj.NewHTTPErrorWithCode(500, errorCode, fmt.Sprintf("%s: ExecuteAction failed: %v", failedAction, err))
	}
	// Track API key usage success
	if session.ApiKey != nil {
		db.UpdateApiKeyLastUsageSuccess(ctx, session.ApiKey.ID, true)
	}
	response.TokenUsage = &usage
	// Set prompts on response for transparency (educational debug view)
	// PromptStatusUpdate is set by the platform's ExecuteAction (platform-specific input format)
	if response.PromptStatusUpdate == nil {
		response.PromptStatusUpdate = functional.Ptr(action.ToAiJSON())
	}
	response.PromptResponseSchema = functional.Ptr(string(gameSchemaJSON))
	response.PromptExpandStory = functional.Ptr(templates.PromptNarratePlotOutline(session.Language, session.WorkshopPromptConstraints))
	if response.ImagePrompt != nil {
		scenarioForImage := functional.First(session.GameScenarioImagePrompt, session.GameScenario)
		plotOutline := ""
		if response.Plot != nil {
			plotOutline = functional.Deref(response.Plot, "")
		}
		response.PromptImageGeneration = functional.Ptr(templates.BuildImagePrompt(session.GameDescription, scenarioForImage, plotOutline, functional.Deref(response.ImagePrompt, ""), session.ImageStyle))
	}

	// Set capability flags based on platform tier
	if model := platform.ResolveModelInfo(session.AiModel); model != nil {
		response.HasImage = model.SupportsImage && functional.Deref(response.ImagePrompt, "") != "" && session.ImageStyle != templates.ImageStyleNoImage
		response.HasAudioIn = model.SupportsAudioIn
		response.HasAudioOut = model.SupportsAudioOut
	}

	// Persist AI session state immediately so the next action can find the conversation/response ID.
	// (ExpandStory may update it again later with a newer ID, which the goroutine will also persist.)
	if err := db.UpdateGameSessionAiSession(ctx, session.UserID, session.ID, session.AiSession); err != nil {
		log.Warn("failed to persist AI session state after ExecuteAction", "session_id", session.ID, "error", err)
	}

	// Save the structured response (statusFields, plot, imagePrompt, prompts) to DB.
	// Plot holds the plot outline; Message stays empty until ExpandStory writes the prose.
	log.Info("[AI] plotOutline (initial)", "session_id", session.ID, "text", functional.Deref(response.Plot, ""))
	_ = db.UpdateGameSessionMessage(ctx, session.UserID, *response)

	// Capture values before goroutines to avoid race conditions
	messageID := response.ID

	// Phase 2 & 3: Run ExpandStory and GenerateImage in parallel (async)
	log.Debug("starting async story expansion and image generation", "session_id", session.ID)
	// The ExpandStory goroutine takes ownership of the session lock.
	earlyReturn = false
	go func() {
		defer unlock()
		log.Debug("starting ExpandStory", "session_id", session.ID, "message_id", messageID)
		// ExpandStory streams text and updates response.Message with full narrative
		_, err := platform.ExpandStory(context.Background(), session, response, responseStream)
		if err != nil {
			log.Warn("ExpandStory failed", "session_id", session.ID, "error", err)
		} else {
			log.Info("[AI] prose (final)", "session_id", session.ID, "text", response.Message)
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

		// Phase 4: Generate audio narration (after text is finalized)
		if !response.HasAudioOut || len(response.Message) == 0 {
			responseStream.Send(obj.GameSessionMessageChunk{AudioDone: true})
		} else {
			audioData, err := platform.GenerateAudio(context.Background(), session, response.Message, responseStream)
			if err != nil {
				log.Warn("GenerateAudio failed", "session_id", session.ID, "error", err)
			} else {
				response.Audio = audioData
				log.Debug("audio generated", "session_id", session.ID, "audio_bytes", len(audioData))
			}
		}
	}()

	go func() {
		if !response.HasImage {
			responseStream.Send(obj.GameSessionMessageChunk{ImageDone: true})
			return
		}
		if err := platform.GenerateImage(context.Background(), session, response, responseStream); err != nil {
			log.Warn("GenerateImage failed", "session_id", session.ID, "error", err)

			// Signal the error to frontend via both cache (for polling) and stream (for SSE).
			// Some platforms (OpenAI) already call cache.SetError internally; calling it
			// again here is a no-op but ensures all platforms are covered.
			errorCode := extractAIErrorCode(err)
			imagecache.Get().SetError(response.ID, errorCode, err.Error())
			responseStream.Send(obj.GameSessionMessageChunk{ImageDone: true, ImageError: err.Error()})

			// Check for errors that require action
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
		}
	}()

	return response, nil
}

// DoSessionActionWithFallback resolves all API key candidates for an existing session,
// then tries them in order. If the primary key fails with a key-related error,
// it retries with fallback candidates before giving up.
func DoSessionActionWithFallback(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (*obj.GameSessionMessage, *obj.HTTPError) {
	candidates, resolveErr := ResolveSessionApiKeyCandidates(ctx, session)
	if resolveErr != nil {
		return nil, resolveErr
	}

	// Apply the primary candidate
	applyResolvedKey(session, &candidates[0])

	response, httpErr := DoSessionAction(ctx, session, action)
	if httpErr == nil {
		return response, nil
	}

	// Retry with fallback candidates on key-related errors
	if len(candidates) > 1 && isKeyRelatedError(httpErr.Code) {
		for i := 1; i < len(candidates); i++ {
			fallback := candidates[i]
			log.Info("retrying session action with fallback API key", "attempt", i+1, "session_id", session.ID, "key_name", fallback.Share.ApiKey.Name, "key_platform", fallback.Share.ApiKey.Platform)

			applyResolvedKey(session, &fallback)
			response, httpErr = DoSessionAction(ctx, session, action)
			if httpErr == nil {
				log.Info("fallback API key succeeded for session action", "key_name", fallback.Share.ApiKey.Name)
				return response, nil
			}

			if !isKeyRelatedError(httpErr.Code) {
				break
			}
		}
	}

	return nil, httpErr
}

// autoRemoveSponsorshipOnKeyFailure checks if the failing API key is the game's public sponsor
// and auto-removes the sponsorship if so. Returns true if sponsorship was removed.
func autoRemoveSponsorshipOnKeyFailure(ctx context.Context, gameID uuid.UUID, apiKeyID uuid.UUID) bool {
	game, err := db.GetGameByID(ctx, nil, gameID)
	if err != nil || game.PublicSponsoredApiKeyShareID == nil {
		return false
	}

	// Check if the sponsored share uses the failing API key
	share, err := db.GetApiKeyShareByID(ctx, uuid.Nil, *game.PublicSponsoredApiKeyShareID)
	if err != nil || share.ApiKey == nil || share.ApiKey.ID != apiKeyID {
		return false
	}

	// The failing key is the sponsor - remove the sponsorship
	if clearErr := db.ClearGamePublicSponsorshipByShareID(ctx, gameID, *game.PublicSponsoredApiKeyShareID); clearErr != nil {
		log.Warn("failed to auto-remove game sponsorship", "game_id", gameID, "share_id", *game.PublicSponsoredApiKeyShareID, "error", clearErr)
		return false
	}

	log.Info("auto-removed game sponsorship due to key failure", "game_id", gameID, "api_key_id", apiKeyID)
	return true
}

// RetryImageGeneration re-triggers image generation for a message that has an imagePrompt
// but no persisted image. Called when loading a session and detecting a missing image.
// Runs asynchronously - the frontend will poll /messages/{id}/status to track progress.
// Only retries if images are enabled on the session and the message is not already generating.
func RetryImageGeneration(session *obj.GameSession, message *obj.GameSessionMessage) {
	if session == nil || message == nil {
		return
	}
	if session.ApiKey == nil {
		log.Debug("skip image retry: no API key", "message_id", message.ID)
		return
	}
	if message.ImagePrompt == nil || *message.ImagePrompt == "" {
		return
	}
	if session.ImageStyle == templates.ImageStyleNoImage {
		log.Debug("skip image retry: images disabled", "session_id", session.ID)
		return
	}

	// Check if already generating (in cache)
	cache := imagecache.Get()
	status := cache.GetStatus(message.ID)
	if status.Exists {
		log.Debug("skip image retry: already in cache", "message_id", message.ID, "complete", status.IsComplete)
		return
	}

	log.Info("retrying image generation for message", "session_id", session.ID, "message_id", message.ID)

	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		log.Warn("skip image retry: failed to get AI platform", "error", err)
		return
	}

	// Create a stream for the image generation (no SSE consumer - just for the ImageSaver)
	responseStream := stream.Get().Create(context.Background(), message,
		func(messageID uuid.UUID, imageData []byte) error {
			return db.UpdateGameSessionMessageImage(context.Background(), session.UserID, messageID, imageData)
		},
		nil, // no audio saver for image retry
	)

	go func() {
		if err := platform.GenerateImage(context.Background(), session, message, responseStream); err != nil {
			log.Warn("image retry failed", "session_id", session.ID, "message_id", message.ID, "error", err)
			errorCode := extractAIErrorCode(err)
			cache.SetError(message.ID, errorCode, err.Error())
		} else {
			log.Info("image retry completed", "session_id", session.ID, "message_id", message.ID, "image_size", len(message.Image))
		}
		// Clean up the stream (no SSE consumer will drain it)
		stream.Get().Remove(message.ID)
	}()
}

// ApplySessionCapabilities stamps HasAudioIn onto messages loaded from the DB (where it is not
// persisted). Call this after loading historical messages so the frontend can enable voice input.
func ApplySessionCapabilities(session *obj.GameSession, msgs []obj.GameSessionMessage) {
	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		return
	}
	model := platform.ResolveModelInfo(session.AiModel)
	if model == nil {
		return
	}
	for i := range msgs {
		msgs[i].HasAudioIn = model.SupportsAudioIn
	}
}
