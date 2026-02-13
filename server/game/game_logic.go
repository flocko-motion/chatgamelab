package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

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

// sessionSetupResult holds the output of generateSessionSetup.
type sessionSetupResult struct {
	candidateIndex int               // index of the candidate that succeeded
	theme          *obj.GameTheme    // generated or game-level theme (may be nil → default)
	translatedGame *obj.Game         // translated game (nil if not needed or failed)
	fieldNameMap   map[string]string // original→translated field name mapping
	usage          obj.TokenUsage    // accumulated token usage for theme+translation
}

// generateSessionSetup runs theme generation and game translation in parallel using
// the given API key candidates. If theme generation fails with a key-related error,
// the next candidate is tried (translation results from the failed key are discarded).
//
// Returns an error only when ALL candidates fail with key-related errors during
// theme generation — this means every key is broken and there's no point continuing.
// Non-key errors (e.g. parse failures) are treated as non-fatal (default theme used).
func generateSessionSetup(
	ctx context.Context,
	candidates []resolvedKey,
	game *obj.Game,
	user *obj.User,
) (*sessionSetupResult, *obj.HTTPError) {
	result := &sessionSetupResult{}
	needsTranslation := user.Language != "" && user.Language != "en"

	// If game already has a theme, skip AI generation entirely
	if game.Theme != nil {
		log.Debug("using game-level theme override", "game_id", game.ID, "preset", game.Theme.Preset)
		result.theme = game.Theme
		result.candidateIndex = 0
		// Still run translation (no key validation via theme, but best-effort)
		result.translatedGame, result.fieldNameMap, result.usage = translateIfNeeded(ctx, candidates[0], game, user)
		return result, nil
	}

	// Try each candidate: run theme + translation in parallel, check for key errors after both finish
	var lastErr error
	var lastErrCode string
	for i, candidate := range candidates {
		if _, platformErr := ai.GetAiPlatform(candidate.Share.ApiKey.Platform); platformErr != nil {
			log.Debug("candidate has invalid platform, skipping", "index", i, "platform", candidate.Share.ApiKey.Platform)
			continue
		}

		tempSession := makeTempSession(candidate, game, user.ID)
		start := time.Now()

		// Run theme generation and translation in parallel
		var wg sync.WaitGroup
		var theme *obj.GameTheme
		var themeUsage obj.TokenUsage
		var themeErr error
		var translatedGame *obj.Game
		var fieldNameMap map[string]string
		var translateUsage obj.TokenUsage

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Debug("generating visual theme", "game_id", game.ID, "candidate_index", i, "key_name", candidate.Share.ApiKey.Name)
			theme, themeUsage, themeErr = GenerateTheme(ctx, tempSession, game, user.Language)
		}()

		if needsTranslation {
			wg.Add(1)
			go func() {
				defer wg.Done()
				translatedGame, fieldNameMap, translateUsage = translateIfNeeded(ctx, candidate, game, user)
			}()
		}

		wg.Wait()
		result.usage = result.usage.Add(themeUsage).Add(translateUsage)

		if themeErr == nil {
			log.Debug("theme generated successfully", "preset", theme.Preset, "seconds", time.Since(start).Seconds())
			result.theme = theme
			result.candidateIndex = i
			result.translatedGame = translatedGame
			result.fieldNameMap = fieldNameMap
			return result, nil
		}

		// Theme generation failed — check if it's a key error
		errCode := extractAIErrorCode(themeErr)
		lastErr = themeErr
		lastErrCode = errCode
		log.Warn("theme generation failed", "candidate_index", i, "key_name", candidate.Share.ApiKey.Name, "error", themeErr, "error_code", errCode, "seconds", time.Since(start).Seconds())

		if !isKeyRelatedError(errCode) {
			// Non-key error (e.g. parse failure, timeout) — use default theme, keep translation
			log.Debug("non-key error during theme generation, using default theme")
			result.candidateIndex = i
			result.translatedGame = translatedGame
			result.fieldNameMap = fieldNameMap
			return result, nil
		}

		// Key-related error — mark key as failed and discard translation (same broken key), try next candidate
		db.UpdateApiKeyLastUsageSuccess(ctx, candidate.Share.ApiKey.ID, false, errCode)
		log.Info("API key failed during theme generation, trying next candidate", "candidate_index", i, "error_code", errCode)
	}

	// All candidates exhausted with key-related errors
	log.Warn("all API key candidates failed during theme generation", "last_error", lastErr)
	return nil, obj.NewHTTPErrorWithCode(500, lastErrCode, fmt.Sprintf("All API keys failed: %v", lastErr))
}

// makeTempSession creates a temporary session object for theme/translation AI calls.
func makeTempSession(candidate resolvedKey, game *obj.Game, userID uuid.UUID) *obj.GameSession {
	return &obj.GameSession{
		GameID:       game.ID,
		GameName:     game.Name,
		UserID:       userID,
		ApiKeyID:     &candidate.Share.ApiKey.ID,
		ApiKey:       candidate.Share.ApiKey,
		AiPlatform:   candidate.Share.ApiKey.Platform,
		AiModel:      candidate.AiQualityTier,
		ImageStyle:   templates.ImageStyleOrDefault(game.ImageStyle),
		StatusFields: game.StatusFields,
	}
}

// translateIfNeeded runs game translation if the user's language is not English.
// Returns nil translatedGame if translation is not needed or fails (non-fatal).
func translateIfNeeded(ctx context.Context, candidate resolvedKey, game *obj.Game, user *obj.User) (*obj.Game, map[string]string, obj.TokenUsage) {
	if user.Language == "" || user.Language == "en" {
		return nil, nil, obj.TokenUsage{}
	}

	tempSession := makeTempSession(candidate, game, user.ID)
	start := time.Now()
	log.Debug("translating game to user language", "game_id", game.ID, "target_lang", user.Language)
	translated, fieldNameMap, usage, err := TranslateGame(ctx, tempSession, game, user.Language)
	if err != nil {
		log.Warn("failed to translate game, using original", "target_lang", user.Language, "error", err, "seconds", time.Since(start).Seconds())
		return nil, nil, usage
	}
	log.Debug("game translated successfully", "target_lang", user.Language, "seconds", time.Since(start).Seconds())
	return translated, fieldNameMap, usage
}

// CreateSession creates a new game session for a user.
// The API key and AI quality tier are resolved server-side using the following priority:
//  1. Workshop key + workshop.aiQualityTier
//  2. Sponsored game key (public sponsor on the game)
//  3. Institution free-use key + institution.freeUseAiQualityTier
//  4. User's default API key + user.aiQualityTier
//  5. System free-use key + system_settings.freeUseAiQualityTier
//
// If no key can be resolved, returns ErrCodeNoApiKey.
// If the source's tier is empty, falls back to system_settings.defaultAiQualityTier.
// Returns *obj.HTTPError (which implements the standard error interface) for client-facing errors with appropriate status codes.
func CreateSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	log.Debug("creating session", "user_id", userID, "game_id", gameID)

	// Resolve all API key candidates using priority chain (up to 3, deduplicated)
	candidates, httpErr := resolveApiKeyCandidates(ctx, userID, gameID)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	log.Info("resolved API key candidates", "count", len(candidates), "primary_key", candidates[0].Share.ApiKey.Name, "primary_platform", candidates[0].Share.ApiKey.Platform)

	// Get the game
	log.Debug("loading game", "game_id", gameID)
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("game not found", "game_id", gameID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(404, obj.ErrCodeNotFound, "Game not found")
	}

	// Delete any existing sessions for this user+game (restart scenario)
	log.Debug("deleting existing sessions", "user_id", userID, "game_id", gameID)
	if err := db.DeleteUserGameSessions(ctx, userID, gameID); err != nil {
		log.Debug("failed to delete existing sessions", "error", err)
		// Non-fatal - continue with session creation
	}

	// Get user to check language preference
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("failed to get user for language preference", "user_id", userID, "error", err)
		return nil, nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get user: " + err.Error()}
	}

	// Run theme generation + translation with fallback across candidates.
	// If all keys fail with key-related errors, we fail early here.
	setup, httpErr := generateSessionSetup(ctx, candidates, game, user)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	// Use the candidate that succeeded for theme generation
	share := candidates[setup.candidateIndex].Share
	aiModel := candidates[setup.candidateIndex].AiQualityTier
	theme := setup.theme
	sessionUsage := setup.usage

	// Apply translation if available
	if setup.translatedGame != nil {
		game = setup.translatedGame

		// Rewrite theme statusEmojis keys from original to translated field names
		if theme != nil && len(theme.StatusEmojis) > 0 && len(setup.fieldNameMap) > 0 {
			translatedEmojis := make(map[string]string, len(theme.StatusEmojis))
			for originalName, emoji := range theme.StatusEmojis {
				if translatedName, ok := setup.fieldNameMap[originalName]; ok {
					translatedEmojis[translatedName] = emoji
				} else {
					translatedEmojis[originalName] = emoji
				}
			}
			theme.StatusEmojis = translatedEmojis
			log.Debug("rewrote theme statusEmojis for translated field names", "mapping", setup.fieldNameMap)
		}
	}

	// Generate system message from (possibly translated) game
	log.Debug("generating system message", "game_id", gameID, "game_name", game.Name)
	systemMessage, err := templates.GetTemplate(game)
	if err != nil {
		log.Debug("failed to get game template", "game_id", gameID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get game template")
	}

	// Persist to database with theme
	log.Debug("persisting session to database")
	session, err := db.CreateGameSession(ctx, userID, game, share.ApiKey.ID, aiModel, nil, theme, user.Language)
	if err != nil {
		log.Debug("failed to create session in DB", "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create session")
	}
	log.Debug("session created", "session_id", session.ID)

	// Attach API key for response
	session.ApiKey = share.ApiKey

	// First action: system message containing the game instructions.
	// Start with the candidate that worked for theme generation; retry with remaining fallbacks on key errors.
	log.Debug("executing initial system action", "session_id", session.ID)
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)

	// Retry with remaining fallback candidates on key-related errors
	remainingCandidates := candidates[setup.candidateIndex+1:]
	if httpErr != nil && len(remainingCandidates) > 0 {
		errorCode := httpErr.Code
		if isKeyRelatedError(errorCode) {
			log.Debug("initial action failed with key error, trying fallback", "session_id", session.ID, "error_code", errorCode)
			if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
				log.Warn("failed to delete empty session before fallback", "session_id", session.ID, "error", delErr)
			}

			for _, fallback := range remainingCandidates {
				log.Info("retrying with fallback API key", "key_name", fallback.Share.ApiKey.Name, "key_platform", fallback.Share.ApiKey.Platform)

				if _, platformErr := ai.GetAiPlatform(fallback.Share.ApiKey.Platform); platformErr != nil {
					log.Debug("fallback key has invalid platform, skipping", "platform", fallback.Share.ApiKey.Platform)
					continue
				}

				session, err = db.CreateGameSession(ctx, userID, game, fallback.Share.ApiKey.ID, fallback.AiQualityTier, nil, theme, user.Language)
				if err != nil {
					log.Debug("failed to create fallback session", "error", err)
					continue
				}
				session.ApiKey = fallback.Share.ApiKey

				startAction.GameSessionID = session.ID
				response, httpErr = DoSessionAction(ctx, session, startAction)
				if httpErr == nil {
					share = fallback.Share
					aiModel = fallback.AiQualityTier
					log.Info("fallback API key succeeded", "key_name", share.ApiKey.Name, "key_platform", share.ApiKey.Platform)
					break
				}

				log.Debug("fallback key also failed", "error", httpErr.Message)
				if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
					log.Warn("failed to delete fallback session", "session_id", session.ID, "error", delErr)
				}

				if !isKeyRelatedError(httpErr.Code) {
					break
				}
			}
		}
	}

	if httpErr != nil {
		log.Debug("initial action failed, deleting empty session", "session_id", session.ID, "error", httpErr.Message)
		if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
			log.Warn("failed to delete empty session after error", "session_id", session.ID, "error", delErr)
		}
		return nil, nil, httpErr
	}
	response.PromptStatusUpdate = functional.Ptr(systemMessage)

	// Increment play count only for non-creator plays
	if !game.Meta.CreatedBy.Valid || game.Meta.CreatedBy.UUID != userID {
		if err := db.IncrementGamePlayCount(ctx, gameID); err != nil {
			log.Warn("failed to increment play count", "game_id", gameID, "error", err)
		}
	}

	// Accumulate setup usage (theme + translation) with action usage
	if response.TokenUsage != nil {
		totalUsage := sessionUsage.Add(*response.TokenUsage)
		response.TokenUsage = &totalUsage
	} else {
		response.TokenUsage = &sessionUsage
	}
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

	// Store the action message (player or system) so it appears in session history.
	// Track the message ID so we can delete it if the AI action fails.
	var actionMessageID *uuid.UUID
	log.Debug("storing action message", "session_id", session.ID, "type", action.Type)
	actionMsg, err := db.CreateGameSessionMessage(ctx, session.UserID, action)
	if err != nil {
		log.Debug("failed to store action message", "session_id", session.ID, "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to store action message")
	}
	actionMessageID = &actionMsg.ID

	// Create placeholder message with Stream=true (client will connect to SSE)
	log.Debug("creating streaming message", "session_id", session.ID)
	response, err = db.CreateStreamingMessage(ctx, session.UserID, session.ID, obj.GameSessionMessageTypeGame)
	if err != nil {
		log.Debug("failed to create streaming message", "session_id", session.ID, "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create streaming message")
	}
	log.Debug("streaming message created", "message_id", response.ID)

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

	// Phase 0: Rephrase player input in third person with uncertain outcome (best-effort)
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
	log.Debug(fmt.Sprintf("executing AI action, session_id=%s, message_id=%s, schema=%s",
		session.ID, response.ID, string(gameSchemaJSON)))
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
	log.Debug("ExecuteAction completed", "session_id", session.ID, "has_image_prompt", response.ImagePrompt != nil)
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
	response.PromptExpandStory = functional.Ptr(templates.PromptNarratePlotOutline)
	response.PromptImageGeneration = response.ImagePrompt

	// Set capability flags based on platform tier
	if model := platform.ResolveModelInfo(session.AiModel); model != nil {
		response.HasImage = model.SupportsImage && response.ImagePrompt != nil && *response.ImagePrompt != "" && session.ImageStyle != templates.ImageStyleNoImage
		response.HasAudio = model.SupportsAudio
	}

	// Persist AI session state immediately so the next action can find the conversation/response ID.
	// (ExpandStory may update it again later with a newer ID, which the goroutine will also persist.)
	if err := db.UpdateGameSessionAiSession(ctx, session.UserID, session.ID, session.AiSession); err != nil {
		log.Warn("failed to persist AI session state after ExecuteAction", "session_id", session.ID, "error", err)
	}

	// Save the structured response (plotOutline in Message, statusFields, imagePrompt)
	// This is returned to client immediately
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
		expandUsage, err := platform.ExpandStory(context.Background(), session, response, responseStream)
		if err != nil {
			log.Warn("ExpandStory failed", "session_id", session.ID, "error", err)
		} else {
			log.Debug("ExpandStory completed", "session_id", session.ID, "message_length", len(response.Message))
		}
		log.Debug("ExpandStory token usage", "session_id", session.ID, "input_tokens", expandUsage.InputTokens, "output_tokens", expandUsage.OutputTokens, "total_tokens", expandUsage.TotalTokens)

		// Update DB with full text (replaces plotOutline)
		response.Stream = false
		if err := db.UpdateGameSessionMessage(context.Background(), session.UserID, *response); err != nil {
			log.Warn("failed to update message after ExpandStory", "session_id", session.ID, "error", err)
		}

		// Update session with new AI state (response IDs for conversation continuity)
		log.Debug("[TRACE] persisting AiSession to DB", "session_id", session.ID, "ai_session", session.AiSession)
		if err := db.UpdateGameSessionAiSession(context.Background(), session.UserID, session.ID, session.AiSession); err != nil {
			log.Warn("failed to update session AI state", "session_id", session.ID, "error", err)
		}

		// Phase 4: Generate audio narration (after text is finalized)
		if !response.HasAudio || len(response.Message) == 0 {
			log.Debug("audio generation not active for this message, signaling audioDone")
			responseStream.Send(obj.GameSessionMessageChunk{AudioDone: true})
		} else {
			log.Debug("starting GenerateAudio", "session_id", session.ID, "message_id", messageID, "text_length", len(response.Message))
			audioData, err := platform.GenerateAudio(context.Background(), session, response.Message, responseStream)
			if err != nil {
				log.Warn("GenerateAudio failed", "session_id", session.ID, "error", err)
			} else {
				response.Audio = audioData
				log.Debug("GenerateAudio completed", "session_id", session.ID, "audio_bytes", len(audioData))
			}
		}
	}()

	go func() {
		log.Debug("starting GenerateImage", "session_id", session.ID, "message_id", messageID, "hasImage", response.HasImage)
		if !response.HasImage {
			log.Debug("image generation not active for this message, signaling imageDone")
			responseStream.Send(obj.GameSessionMessageChunk{ImageDone: true})
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
