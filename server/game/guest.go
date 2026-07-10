// package: game / private share sessions and guest players
// type:    logic
// job:     creates game sessions from private share links, minting anonymous guest users and resolving the share's API key.
// limits:  does not run the game turns or generate content; hands off to session play and the AI platforms (-> ai).
package game

import (
	"context"

	"cgl/db"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"

	ung "github.com/dillonstreator/go-unique-name-generator"
	dictionaries "github.com/dillonstreator/go-unique-name-generator/dictionaries"
	"github.com/google/uuid"
)

// CreateShareSession creates a game session for a private share link.
//
// The player argument decides who owns the session and, therefore, which
// youth-protection cascade applies (see db.ResolveConstraint):
//   - player == nil  → an anonymous guest user is created; the share's "Gäste" cascade applies.
//   - player != nil  → the authenticated user plays as themselves; their own cascade applies,
//     with the share supplying stages 1–2. The play shows up in their "recently played".
//
// Either way the sponsored share key pays (a sponsored game is sponsored for everybody)
// and the share's play-limit counter is consumed.
func CreateShareSession(ctx context.Context, token string, language string, player *obj.User) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	// 1. Validate token → load game share and game
	gameShare, game, httpErr := ValidatePrivateShareToken(ctx, token)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	// 2. Resolve the sponsored API key (required for private shares)
	share, httpErr := resolvePrivateShareKey(ctx, gameShare)
	if httpErr != nil {
		return nil, nil, httpErr
	}

	// 3. Decrement remaining counter (atomic, race-safe)
	if httpErr := decrementPrivateShareRemaining(ctx, gameShare.ID); httpErr != nil {
		return nil, nil, httpErr
	}

	// 4. Determine the session owner: the authenticated player, or a fresh guest user.
	owner := player
	if owner == nil {
		guestUser, httpErr := createGuestUser(ctx, gameShare.ID)
		if httpErr != nil {
			return nil, nil, httpErr
		}
		log.Info("share session: created anonymous guest user", "user_id", guestUser.ID, "user_name", guestUser.Name, "game_id", game.ID)
		// Set language on guest user (not persisted — used for theme/translation this session).
		// Authenticated players keep their own profile language.
		if language != "" {
			guestUser.Language = language
		}
		owner = guestUser
	} else {
		log.Info("share session: authenticated player", "user_id", owner.ID, "game_id", game.ID)
	}

	// 5. Run the full session creation flow (reuses existing logic)
	return createSessionForGuest(ctx, owner, game, share, gameShare)
}

// ValidatePrivateShareToken checks if the token maps to a valid, playable game.
func ValidatePrivateShareToken(ctx context.Context, token string) (*obj.GameShare, *obj.Game, *obj.HTTPError) {
	if token == "" {
		return nil, nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeValidation, "Missing share token")
	}

	gameShare, err := db.GetGameShareByToken(ctx, token)
	if err != nil {
		return nil, nil, obj.NewHTTPErrorWithCode(404, obj.ErrCodeNotFound, "Invalid or expired share link")
	}

	game, err := db.GetGameByIDWithShareToken(ctx, gameShare.GameID, token)
	if err != nil {
		return nil, nil, obj.NewHTTPErrorWithCode(404, obj.ErrCodeNotFound, "Game not found")
	}

	// Verify required game fields
	if game.SystemMessageScenario == "" || game.SystemMessageGameStart == "" {
		return nil, nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeValidation, "Game is not ready to play")
	}

	return gameShare, game, nil
}

// resolvePrivateShareKey loads the API key from the game share's api_key_share_id.
func resolvePrivateShareKey(ctx context.Context, gameShare *obj.GameShare) (*obj.ApiKeyShare, *obj.HTTPError) {
	share, err := db.GetApiKeyShareByID(ctx, uuid.Nil, gameShare.ApiKeyShareID)
	if err != nil || share.ApiKey == nil {
		log.Warn("guest session: sponsored key not accessible", "game_share_id", gameShare.ID, "api_key_share_id", gameShare.ApiKeyShareID)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeNoApiKey, "Sponsored API key is not available")
	}
	return share, nil
}

// decrementPrivateShareRemaining atomically decrements the remaining counter.
// Returns nil if unlimited (NULL) or still has remaining plays.
func decrementPrivateShareRemaining(ctx context.Context, shareID uuid.UUID) *obj.HTTPError {
	_, err := db.DecrementGameShareRemaining(ctx, shareID)
	if err != nil {
		return obj.NewHTTPErrorWithCode(403, "share_exhausted", "This share link has reached its play limit")
	}
	return nil
}

// createGuestUser creates an anonymous user for a guest play session.
// The user has no email, no auth0 ID, and no participant token.
func createGuestUser(ctx context.Context, shareID uuid.UUID) (*obj.User, *obj.HTTPError) {
	nameGenerator := ung.NewUniqueNameGenerator(
		ung.WithDictionaries([][]string{
			dictionaries.Colors,
			dictionaries.Animals,
		}),
		ung.WithSeparator("-"),
	)
	// Append a short random suffix — the readable color+animal pair has limited
	// combinations and can collide under load.
	name := "guest-" + nameGenerator.Generate() + "-" + functional.First(functional.GenerateSecureToken(2))
	userID := uuid.New()

	err := db.CreateGuestUser(ctx, userID, name, shareID)
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
func createSessionForGuest(ctx context.Context, user *obj.User, game *obj.Game, share *obj.ApiKeyShare, gameShare *obj.GameShare) (*obj.GameSession, *obj.GameSessionMessage, *obj.HTTPError) {
	// Resolve AI quality tier with priority:
	// 1. Workshop tier (if this is a workshop share)
	// 2. Per-share tier (if set on the game share)
	// 3. System default tier
	settings, _ := db.GetSystemSettings(ctx)
	defaultTier := obj.AiModelBalanced
	if settings != nil && settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}

	aiModel := defaultTier
	if gameShare.WorkshopID != nil {
		// Workshop share: use the workshop's AI quality tier
		workshop, err := db.GetWorkshopByID(ctx, uuid.Nil, *gameShare.WorkshopID)
		if err == nil && workshop.AiQualityTier != nil && *workshop.AiQualityTier != "" {
			aiModel = *workshop.AiQualityTier
		}
	} else if gameShare.AiQualityTier != nil && *gameShare.AiQualityTier != "" {
		// Non-workshop share: use the per-share tier
		aiModel = *gameShare.AiQualityTier
	}

	log.Info("guest session: using API key", "key_name", share.ApiKey.Name, "platform", share.ApiKey.Platform, "ai_model", aiModel)

	// Build a single-element candidate list for createSessionInternal
	candidates := []resolvedKey{{Share: share, AiQualityTier: aiModel, KeyType: obj.ApiKeyTypePrivateShare}}

	// Use shared internal implementation. Passing gameShare lets the canonical
	// resolver (db.ResolveConstraint) pick the right cascade for whoever owns this
	// session — the share's cascade for a guest user, the user's own cascade (with
	// the share supplying stages 1–2) for an authenticated player.
	// Guest users: no retries (nil), don't delete existing sessions (false)
	return createSessionInternal(ctx, user.ID, game, user, gameShare, candidates, nil, false)
}

// ResolveGuestSessionApiKey re-resolves the API key for a guest session from the game share.
// This is the guest equivalent of ResolveSessionApiKey.
func ResolveGuestSessionApiKey(ctx context.Context, session *obj.GameSession, gameShare *obj.GameShare) *obj.HTTPError {
	share, httpErr := resolvePrivateShareKey(ctx, gameShare)
	if httpErr != nil {
		return httpErr
	}
	session.ApiKey = share.ApiKey
	session.ApiKeyID = &share.ApiKey.ID
	session.AiPlatform = share.ApiKey.Platform
	session.ApiKeyType = obj.ApiKeyTypePrivateShare
	return nil
}
