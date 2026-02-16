package game

import (
	"context"

	"cgl/db"
	"cgl/functional"
	"cgl/game/templates"
	"cgl/log"
	"cgl/obj"

	ung "github.com/dillonstreator/go-unique-name-generator"
	dictionaries "github.com/dillonstreator/go-unique-name-generator/dictionaries"
	"github.com/google/uuid"
)

// CreateGuestSession creates a game session for an anonymous guest via a private share token.
// It validates the token, checks the remaining counter, creates an anonymous user,
// and runs the full session creation flow (theme, translation, first AI action).
func CreateGuestSession(ctx context.Context, token string, language string) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	// 1. Validate token → load game
	game, httpErr := ValidatePrivateShareToken(ctx, token)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	// 2. Resolve the sponsored API key (required for private shares)
	share, httpErr := resolvePrivateShareKey(ctx, game)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	// 3. Decrement remaining counter (atomic, race-safe)
	if httpErr := decrementPrivateShareRemaining(ctx, game.ID); httpErr != nil {
		return nil, nil, httpErr
	}

	// 4. Create anonymous guest user
	guestUser, httpErr := createGuestUser(ctx, game.ID)
	if httpErr != nil {
		return nil, nil, httpErr
	}
	log.Info("guest session: created anonymous user", "user_id", guestUser.ID, "user_name", guestUser.Name, "game_id", game.ID)

	// 5. Set language on guest user (not persisted — used for theme/translation in this session)
	if language != "" {
		guestUser.Language = language
	}

	log.Debug("guest session: language for session", "language", guestUser.Language, "requested_language", language)

	// 6. Run the full session creation flow (reuses existing logic)
	return createSessionForGuest(ctx, guestUser, game, share)
}

// ValidatePrivateShareToken checks if the token maps to a valid, playable game.
func ValidatePrivateShareToken(ctx context.Context, token string) (*obj.Game, *obj.HTTPError) {
	if token == "" {
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeValidation, "Missing share token")
	}

	game, err := db.GetGameByToken(ctx, token)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(404, obj.ErrCodeNotFound, "Invalid or expired share link")
	}

	// Verify the game has a private sponsored key configured
	if game.PrivateSponsoredApiKeyShareID == nil {
		log.Warn("guest session: game has share token but no sponsored key", "game_id", game.ID)
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "This share link is not fully configured")
	}

	// Verify required game fields
	if game.SystemMessageScenario == "" || game.SystemMessageGameStart == "" {
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeValidation, "Game is not ready to play")
	}

	return game, nil
}

// resolvePrivateShareKey loads the API key from the game's private sponsored share.
func resolvePrivateShareKey(ctx context.Context, game *obj.Game) (*obj.ApiKeyShare, *obj.HTTPError) {
	share, err := db.GetApiKeyShareByID(ctx, uuid.Nil, *game.PrivateSponsoredApiKeyShareID)
	if err != nil || share.ApiKey == nil {
		log.Warn("guest session: sponsored key not accessible", "game_id", game.ID, "share_id", *game.PrivateSponsoredApiKeyShareID)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeNoApiKey, "Sponsored API key is not available")
	}
	return share, nil
}

// decrementPrivateShareRemaining atomically decrements the remaining counter.
// Returns nil if unlimited (NULL) or still has remaining plays.
func decrementPrivateShareRemaining(ctx context.Context, gameID uuid.UUID) *obj.HTTPError {
	_, err := db.DecrementPrivateShareRemaining(ctx, gameID)
	if err != nil {
		return obj.NewHTTPErrorWithCode(403, "share_exhausted", "This share link has reached its play limit")
	}
	return nil
}

// createGuestUser creates an anonymous user for a guest play session.
// The user has no email, no auth0 ID, and no participant token.
func createGuestUser(ctx context.Context, gameID uuid.UUID) (*obj.User, *obj.HTTPError) {
	nameGenerator := ung.NewUniqueNameGenerator(
		ung.WithDictionaries([][]string{
			dictionaries.Colors,
			dictionaries.Animals,
		}),
		ung.WithSeparator("-"),
	)
	name := "guest-" + nameGenerator.Generate()
	userID := uuid.New()

	err := db.CreateGuestUser(ctx, userID, name, gameID)
	if err != nil {
		log.Error("guest session: failed to create anonymous user", "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create guest user")
	}

	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to load guest user")
	}
	return user, nil
}

// createSessionForGuest runs the full session creation flow for a guest.
// This is very similar to CreateSession but uses a pre-resolved API key share
// and doesn't need user-level key resolution.
func createSessionForGuest(ctx context.Context, user *obj.User, game *obj.Game, share *obj.ApiKeyShare) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	// Load system settings for the default tier fallback
	settings, _ := db.GetSystemSettings(ctx)
	defaultTier := obj.AiModelBalanced
	if settings != nil && settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}
	aiModel := defaultTier

	log.Info("guest session: using API key", "key_name", share.ApiKey.Name, "platform", share.ApiKey.Platform, "ai_model", aiModel)

	// Build a single-element candidate list for generateSessionSetup
	candidates := []resolvedKey{{Share: share, AiQualityTier: aiModel, KeyType: obj.ApiKeyTypePrivateShare}}

	// Run theme generation + translation; fails early on key-related errors
	setup, httpErr := generateSessionSetup(ctx, candidates, game, user)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	theme := setup.theme
	sessionUsage := setup.usage

	if setup.translatedGame != nil {
		game = setup.translatedGame
	}

	// Generate system message from (possibly translated) game
	systemMessage, err := templates.GetTemplate(game, user.Language)
	if err != nil {
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get game template")
	}

	// Persist session
	keyType := obj.ApiKeyTypePrivateShare
	session, err := db.CreateGameSession(ctx, user.ID, game, share.ApiKey.ID, aiModel, nil, theme, user.Language, &keyType)
	if err != nil {
		return nil, nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create session")
	}
	session.ApiKey = share.ApiKey

	// Run first AI action
	startAction := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypeSystem,
		Message:       systemMessage,
	}
	response, httpErr := DoSessionAction(ctx, session, startAction)
	if httpErr != nil {
		if delErr := db.DeleteEmptyGameSession(ctx, session.ID); delErr != nil {
			log.Warn("guest session: failed to delete empty session after error", "session_id", session.ID, "error", delErr)
		}
		return nil, nil, httpErr
	}
	response.PromptStatusUpdate = functional.Ptr(systemMessage)

	// Increment play count
	if err := db.IncrementGamePlayCount(ctx, game.ID); err != nil {
		log.Warn("guest session: failed to increment play count", "game_id", game.ID, "error", err)
	}

	// Accumulate token usage
	if response.TokenUsage != nil {
		totalUsage := sessionUsage.Add(*response.TokenUsage)
		response.TokenUsage = &totalUsage
	} else {
		response.TokenUsage = &sessionUsage
	}

	return session, response, nil
}

// ResolveGuestSessionApiKey re-resolves the API key for a guest session from the game's private share.
// This is the guest equivalent of ResolveSessionApiKey.
func ResolveGuestSessionApiKey(ctx context.Context, session *obj.GameSession, gameObj *obj.Game) *obj.HTTPError {
	share, httpErr := resolvePrivateShareKey(ctx, gameObj)
	if httpErr != nil {
		return httpErr
	}
	session.ApiKey = share.ApiKey
	session.ApiKeyID = &share.ApiKey.ID
	session.AiPlatform = share.ApiKey.Platform
	keyType := obj.ApiKeyTypePrivateShare
	session.ApiKeyType = &keyType
	return nil
}
