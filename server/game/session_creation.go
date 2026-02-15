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
	"cgl/game/templates"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// FallbackDecider returns true if we should retry with the next candidate after a failure.
// Parameters: errorCode (e.g. "invalid_api_key"), attemptIndex (0-based), candidate (the failed key).
// Return nil for no retries.
type FallbackDecider func(errorCode string, attemptIndex int, candidate resolvedKey) bool

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
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("game not found", "game_id", gameID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(404, obj.ErrCodeNotFound, "Game not found")
	}

	// Get user to check language preference
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("failed to get user for language preference", "user_id", userID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get user")
	}

	// Define fallback retry logic for authenticated users
	shouldRetry := func(errorCode string, attemptIndex int, candidate resolvedKey) bool {
		return isKeyRelatedError(errorCode)
	}

	// Use shared internal implementation
	// Authenticated users: delete existing sessions (restart scenario)
	return createSessionInternal(ctx, userID, game, user, candidates, shouldRetry, true)
}

// sessionSetupResult holds the output of generateSessionSetup.
type sessionSetupResult struct {
	candidateIndex      int               // index of the candidate that succeeded
	theme               *obj.GameTheme    // generated or game-level theme (may be nil → default)
	translatedGame      *obj.Game         // translated game (nil if not needed or failed)
	fieldNameMap        map[string]string // original→translated field name mapping
	scenarioImagePrompt string            // condensed scenario for image generation
	usage               obj.TokenUsage    // accumulated token usage for theme+translation
}

// buildScenarioImagePrompt condenses the game scenario for image generation prompts.
// Uses AI to condense long scenarios (>500 chars) to keep image prompts focused.
func buildScenarioImagePrompt(ctx context.Context, session *obj.GameSession, platform ai.AiPlatform) string {
	rawScenario := strings.TrimSpace(session.GameScenario)
	if rawScenario == "" {
		return ""
	}

	// Keep short scenarios as-is.
	if len(rawScenario) <= 500 || session.ApiKey == nil {
		return rawScenario
	}

	prompt := fmt.Sprintf(templates.PromptCondenseScenarioForImage, rawScenario)
	condensed, err := platform.ToolQuery(ctx, session.ApiKey.Key, prompt)
	if err != nil {
		log.Warn("ToolQuery scenario condensation failed, using original scenario", "session_id", session.ID, "error", err)
		return ""
	}

	return strings.TrimSpace(condensed)
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

		// Run theme generation, translation, and scenario image prompt in parallel
		var wg sync.WaitGroup
		var theme *obj.GameTheme
		var themeUsage obj.TokenUsage
		var themeErr error
		var translatedGame *obj.Game
		var fieldNameMap map[string]string
		var translateUsage obj.TokenUsage
		var scenarioPrompt string

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

		// Prepare scenario image prompt in parallel (doesn't need translation)
		wg.Add(1)
		go func() {
			defer wg.Done()
			platform, platformErr := ai.GetAiPlatform(tempSession.AiPlatform)
			if platformErr != nil {
				log.Warn("failed to get AI platform for scenario image prompt", "game_id", game.ID, "error", platformErr)
				return
			}
			model := platform.ResolveModelInfo(tempSession.AiModel)
			if model != nil && model.SupportsImage && tempSession.ImageStyle != templates.ImageStyleNoImage {
				scenarioPrompt = buildScenarioImagePrompt(ctx, tempSession, platform)
			}
		}()

		wg.Wait()
		result.usage = result.usage.Add(themeUsage).Add(translateUsage)

		if themeErr == nil {
			log.Debug("theme generated successfully", "preset", theme.Preset, "seconds", time.Since(start).Seconds())
			result.theme = theme
			result.candidateIndex = i
			result.translatedGame = translatedGame
			result.fieldNameMap = fieldNameMap
			result.scenarioImagePrompt = scenarioPrompt
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

// createSessionInternal is the shared implementation for session creation.
// Differences between authenticated and guest flows:
// - shouldRetry: fallback logic (authenticated: retry on key errors, guest: no retries)
// - deleteExistingSessions: whether to delete previous sessions (authenticated: yes, guest: no)
func createSessionInternal(
	ctx context.Context,
	userID uuid.UUID,
	game *obj.Game,
	user *obj.User,
	candidates []resolvedKey,
	shouldRetry FallbackDecider,
	deleteExistingSessions bool,
) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	log.Debug("creating session (internal)", "user_id", userID, "game_id", game.ID, "candidates", len(candidates))

	// Delete any existing sessions for this user+game (restart scenario)
	// Authenticated users: yes, Guest users: no (reason TBD - checking with colleague)
	if deleteExistingSessions {
		if err := db.DeleteUserGameSessions(ctx, userID, game.ID); err != nil {
			log.Warn("failed to delete existing sessions", "error", err)
			// Non-fatal - continue with session creation
		}
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
		// This is core game logic, not user-role-specific
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
	systemMessage, err := templates.GetTemplate(game, user.Language)
	if err != nil {
		log.Debug("failed to get game template", "game_id", game.ID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get game template")
	}

	// Persist to database with theme
	session, err := db.CreateGameSession(ctx, userID, game, share.ApiKey.ID, aiModel, nil, theme, user.Language)
	if err != nil {
		log.Debug("failed to create session in DB", "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create session")
	}
	log.Debug("session created", "session_id", session.ID)

	// Attach API key for response
	session.ApiKey = share.ApiKey

	// Apply scenario image prompt from parallel setup (already calculated)
	session.GameScenarioImagePrompt = setup.scenarioImagePrompt

	// First action: system message containing the game instructions.
	// Start with the candidate that worked for theme generation; retry with remaining fallbacks if shouldRetry allows.
	log.Debug("executing initial system action", "session_id", session.ID)
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)

	// Retry with remaining fallback candidates if shouldRetry function allows
	remainingCandidates := candidates[setup.candidateIndex+1:]
	if httpErr != nil && len(remainingCandidates) > 0 && shouldRetry != nil {
		errorCode := httpErr.Code
		attemptIndex := setup.candidateIndex + 1

		for i, fallback := range remainingCandidates {
			// Ask shouldRetry if we should try this candidate
			if !shouldRetry(errorCode, attemptIndex+i, fallback) {
				log.Debug("shouldRetry returned false, stopping fallback attempts", "error_code", errorCode, "attempt", attemptIndex+i)
				break
			}

			log.Info("retrying with fallback API key", "key_name", fallback.Share.ApiKey.Name, "key_platform", fallback.Share.ApiKey.Platform, "attempt", attemptIndex+i)

			// Delete the failed session before creating a new one
			if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
				log.Warn("failed to delete empty session before fallback", "session_id", session.ID, "error", delErr)
			}

			// Validate platform
			if _, platformErr := ai.GetAiPlatform(fallback.Share.ApiKey.Platform); platformErr != nil {
				log.Debug("fallback key has invalid platform, skipping", "platform", fallback.Share.ApiKey.Platform)
				continue
			}

			// Create new session with fallback key
			session, err = db.CreateGameSession(ctx, userID, game, fallback.Share.ApiKey.ID, fallback.AiQualityTier, nil, theme, user.Language)
			if err != nil {
				log.Debug("failed to create fallback session", "error", err)
				continue
			}
			session.ApiKey = fallback.Share.ApiKey
			session.GameScenarioImagePrompt = setup.scenarioImagePrompt

			// Retry the action
			startAction.GameSessionID = session.ID
			response, httpErr = DoSessionAction(ctx, session, startAction)
			if httpErr == nil {
				share = fallback.Share
				aiModel = fallback.AiQualityTier
				log.Info("fallback API key succeeded", "key_name", share.ApiKey.Name, "key_platform", share.ApiKey.Platform)
				break
			}

			log.Debug("fallback key also failed", "error", httpErr.Message)
			errorCode = httpErr.Code
		}
	}

	// If still failed, clean up and return error
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
		if err := db.IncrementGamePlayCount(ctx, game.ID); err != nil {
			log.Warn("failed to increment play count", "game_id", game.ID, "error", err)
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
