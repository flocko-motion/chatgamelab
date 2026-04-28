package game

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cgl/db"
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
	// No platform filter for new sessions — first successful candidate determines the platform.
	candidates, httpErr := resolveApiKeyCandidates(ctx, userID, gameID, "")
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
	adaptedImageStyle   string            // translated and workshop-adapted image style
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

// translateAndAdaptImageStyle translates the image style to English and adapts it based on workshop constraints.
// Returns the adapted style, or the original if translation fails (non-fatal).
func translateAndAdaptImageStyle(ctx context.Context, session *obj.GameSession, platform ai.AiPlatform, workshopConstraints *string) string {
	rawStyle := strings.TrimSpace(session.ImageStyle)
	if rawStyle == "" || rawStyle == templates.ImageStyleNoImage || session.ApiKey == nil {
		return rawStyle
	}

	var prompt string
	if workshopConstraints != nil && *workshopConstraints != "" {
		prompt = fmt.Sprintf(templates.PromptAdaptImageStyle, rawStyle, *workshopConstraints)
	} else {
		prompt = fmt.Sprintf(templates.PromptTranslateImageStyle, rawStyle)
	}

	adapted, err := platform.ToolQuery(ctx, session.ApiKey.Key, prompt)
	if err != nil {
		log.Warn("ToolQuery image style adaptation failed, using original", "session_id", session.ID, "error", err)
		return rawStyle
	}

	return strings.TrimSpace(adapted)
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

	// Resolve prompt constraints early for parallel tasks (image style adaptation)
	promptConstraints, _ := db.ResolveUserConstraint(ctx, user)

	// If game already has a theme, skip AI generation entirely
	if game.Theme != nil {
		log.Debug("using game-level theme override", "game_id", game.ID, "preset", game.Theme.Preset)
		result.theme = game.Theme
		result.candidateIndex = 0
		// Still run translation and image style adaptation (no key validation via theme, but best-effort)
		result.translatedGame, result.fieldNameMap, result.usage = translateIfNeeded(ctx, candidates[0], game, user)
		// Adapt image style if needed
		if platform, err := ai.GetAiPlatform(candidates[0].Share.ApiKey.Platform); err == nil {
			tempSession := makeTempSession(candidates[0], game, user.ID)
			result.adaptedImageStyle = translateAndAdaptImageStyle(ctx, tempSession, platform, promptConstraints)
		}
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

		// Run theme generation, translation, scenario image prompt, and image style adaptation in parallel
		var wg sync.WaitGroup
		var theme *obj.GameTheme
		var themeUsage obj.TokenUsage
		var themeErr error
		var translatedGame *obj.Game
		var fieldNameMap map[string]string
		var translateUsage obj.TokenUsage
		var scenarioPrompt string
		var adaptedImageStyle string

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

		// Prepare scenario image prompt in parallel
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

		// Translate and adapt image style based on workshop constraints in parallel
		wg.Add(1)
		go func() {
			defer wg.Done()
			platform, platformErr := ai.GetAiPlatform(tempSession.AiPlatform)
			if platformErr != nil {
				log.Warn("failed to get AI platform for image style adaptation", "game_id", game.ID, "error", platformErr)
				return
			}
			adaptedImageStyle = translateAndAdaptImageStyle(ctx, tempSession, platform, promptConstraints)
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
			result.adaptedImageStyle = adaptedImageStyle
			return result, nil
		}

		// Theme generation failed — check if it's a key error
		errCode := extractAIErrorCode(themeErr)
		lastErr = themeErr
		lastErrCode = errCode
		log.Warn("theme generation failed", "candidate_index", i, "key_name", candidate.Share.ApiKey.Name, "error", themeErr, "error_code", errCode, "seconds", time.Since(start).Seconds())

		if !isKeyRelatedError(errCode) {
			// Non-key error (e.g. parse failure, timeout) — use default theme, keep translation and adapted style
			log.Debug("non-key error during theme generation, using default theme")
			result.candidateIndex = i
			result.translatedGame = translatedGame
			result.fieldNameMap = fieldNameMap
			result.adaptedImageStyle = adaptedImageStyle
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

	// Apply adapted image style if available
	imageStyle := templates.ImageStyleOrDefault(game.ImageStyle)
	if setup.adaptedImageStyle != "" {
		imageStyle = setup.adaptedImageStyle
		log.Debug("using workshop-adapted image style", "original", game.ImageStyle, "adapted", imageStyle)
	}

	// Resolve prompt constraints: workshop > org > age-based
	// This is resolved at session load time too (for live updates), but we need it here for the system message.
	promptConstraints, _ := db.ResolveUserConstraint(ctx, user)

	// Persist to database with theme and adapted image style
	session, err := db.CreateGameSession(ctx, userID, game, share.ApiKey.ID, aiModel, nil, theme, user.Language, imageStyle)
	if err != nil {
		log.Debug("failed to create session in DB", "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create session")
	}
	log.Debug("session created", "session_id", session.ID)

	// Store resolved constraints in session for re-injection during prose generation
	session.PromptConstraints = promptConstraints

	// Attach API key and key type for response
	session.ApiKey = share.ApiKey
	session.ApiKeyType = candidates[setup.candidateIndex].KeyType

	// Apply scenario image prompt from parallel setup
	session.GameScenarioImagePrompt = setup.scenarioImagePrompt

	// TWO-PHASE INITIALIZATION: Create system message to persist translated scenario
	// This message will be loaded during the "init" action  in DoSessionAction and sent to the AI
	systemMessage, tmplErr := templates.GetTemplate(game, user.Language)
	if tmplErr != nil {
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to generate system message")
	}
	if promptConstraints != nil {
		systemMessage += "\n\nNARRATION RULES must be respected: " + *promptConstraints
	}

	systemMsg := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	_, err = db.CreateGameSessionMessage(ctx, userID, systemMsg)
	if err != nil {
		log.Warn("failed to create system message", "session_id", session.ID, "error", err)
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create system message")
	}

	log.Debug("session created with system message (two-phase: AI action pending)", "session_id", session.ID)

	// Increment play count only for non-creator plays
	if !game.Meta.CreatedBy.Valid || game.Meta.CreatedBy.UUID != userID {
		if err := db.IncrementGamePlayCount(ctx, game.ID); err != nil {
			log.Warn("failed to increment play count", "game_id", game.ID, "error", err)
		}
	}

	// Return session without messages (phase 1 complete)
	// Note: No token usage from AI action yet - that happens in phase 2
	return session, nil, nil
}
