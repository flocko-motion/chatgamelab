// package: game / private share plays and guest players
// type:    logic
// job:     prepares a play from a share link: token, key and tier; then takes a play and mints a guest user.
// limits:  does not run the game; the engine does.
package game

import (
	"context"

	"cgl/db"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"engine"

	ung "github.com/dillonstreator/go-unique-name-generator"
	dictionaries "github.com/dillonstreator/go-unique-name-generator/dictionaries"
	"github.com/google/uuid"
)

// SharePlay is what a share link settles for one play: who plays which game,
// the sponsored key that pays and the AI quality tier.
type SharePlay struct {
	Player        *obj.User
	Game          *obj.Game
	GameShare     *obj.GameShare
	Key           *obj.ApiKeyShare
	AiQualityTier string
}

// PrepareSharePlay settles a play of a private share link without consuming
// anything: the token, the game and the share's key and tier. Consume takes the
// play once the key is known to work.
//
// The player argument decides who plays and, therefore, which youth-protection
// cascade applies (see db.ResolveConstraint):
//   - player == nil  → Consume creates an anonymous guest user; the share's "Gäste" cascade applies.
//   - player != nil  → the authenticated user plays as themselves; their own cascade applies,
//     with the share supplying stages 1–2.
//
// Either way the sponsored share key pays (a sponsored game is sponsored for everybody).
func PrepareSharePlay(ctx context.Context, token string, player *obj.User) (*SharePlay, *obj.HTTPError) {
	gameShare, game, httpErr := ValidatePrivateShareToken(ctx, token)
	if httpErr != nil {
		return nil, httpErr
	}

	key, httpErr := resolvePrivateShareKey(ctx, gameShare)
	if httpErr != nil {
		return nil, httpErr
	}

	return &SharePlay{
		Player:        player,
		Game:          game,
		GameShare:     gameShare,
		Key:           key,
		AiQualityTier: shareTier(ctx, gameShare),
	}, nil
}

// Consume takes one play from the share's limit and, for a guest, creates the
// anonymous guest user who plays it.
func (sp *SharePlay) Consume(ctx context.Context, language string) *obj.HTTPError {
	if httpErr := decrementPrivateShareRemaining(ctx, sp.GameShare.ID); httpErr != nil {
		return httpErr
	}
	if sp.Player != nil {
		log.Info("share play: authenticated player", "user_id", sp.Player.ID, "game_id", sp.Game.ID)
		return nil
	}
	guestUser, httpErr := createGuestUser(ctx, sp.GameShare.ID)
	if httpErr != nil {
		return httpErr
	}
	log.Info("share play: created anonymous guest user", "user_id", guestUser.ID, "user_name", guestUser.Name, "game_id", sp.Game.ID)
	// The guest's language is not persisted; it holds for this play only.
	// Authenticated players keep their own profile language.
	if language != "" {
		guestUser.Language = language
	}
	sp.Player = guestUser
	return nil
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
		log.Warn("share play: sponsored key not accessible", "game_share_id", gameShare.ID, "api_key_share_id", gameShare.ApiKeyShareID)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeNoApiKey, "Sponsored API key is not available")
	}
	if !db.AcceptsKeys(share.ApiKey.Platform) {
		log.Warn("share play: sponsored key's platform is not supported", "game_share_id", gameShare.ID, "platform", share.ApiKey.Platform)
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "The sponsored API key's platform is not supported yet")
	}
	return share, nil
}

// errCodeShareExhausted is a share link with no plays left.
const errCodeShareExhausted = "share_exhausted"

// decrementPrivateShareRemaining atomically decrements the remaining counter.
// Returns nil if unlimited (NULL) or still has remaining plays.
func decrementPrivateShareRemaining(ctx context.Context, shareID uuid.UUID) *obj.HTTPError {
	_, err := db.DecrementGameShareRemaining(ctx, shareID)
	if err != nil {
		return obj.NewHTTPErrorWithCode(403, errCodeShareExhausted, "This share link has reached its play limit")
	}
	return nil
}

// createGuestUser creates an anonymous user for a guest play.
// The user has no email, no auth0 ID, and no guest token.
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
		log.Error("share play: failed to create anonymous user", "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to create guest user")
	}

	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to load guest user")
	}
	return user, nil
}

// shareTier picks the AI quality tier of a share play: the workshop's tier for a
// workshop share, else the share's own tier, else the system default.
func shareTier(ctx context.Context, gameShare *obj.GameShare) string {
	settings, _ := db.GetSystemSettings(ctx)
	tier := string(engine.TierMedium)
	if settings != nil && settings.DefaultAiQualityTier != "" {
		tier = settings.DefaultAiQualityTier
	}

	if gameShare.WorkshopID != nil {
		workshop, err := db.GetWorkshopByID(ctx, uuid.Nil, *gameShare.WorkshopID)
		if err == nil && workshop.AiQualityTier != nil && *workshop.AiQualityTier != "" {
			tier = *workshop.AiQualityTier
		}
	} else if gameShare.AiQualityTier != nil && *gameShare.AiQualityTier != "" {
		tier = *gameShare.AiQualityTier
	}
	return tier
}
