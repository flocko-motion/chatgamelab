// package: game / engine sessions from the platform's data
// type:    logic
// job:     creates and resumes engine sessions: one checked offer from the key cascade, the rule, the spec.
// limits:  no HTTP or access policy (-> api); storage is db.SessionStore; running it is the engine's.
package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"cgl/db"
	"cgl/log"
	"cgl/obj"
	"engine"

	"github.com/google/uuid"
)

// Created is a session just launched, and the player's earlier sessions of the
// same game it replaces, already deleted.
type Created struct {
	Session  *engine.Session
	Replaced []uuid.UUID
}

// CreateGameSession launches a session of a game for a signed-in player. As
// starting a game anew, it replaces the player's earlier sessions of that game.
// The context outlives the request, as the session's lifetime.
func CreateGameSession(ctx context.Context, player *obj.User, gameID uuid.UUID, language string) (*Created, error) {
	if player == nil {
		return nil, engine.Refuse(engine.ProblemDenied, "sign in to play this game")
	}
	game, err := db.GetGameByID(ctx, &player.ID, gameID)
	if err != nil {
		return nil, engine.Refuse(engine.ProblemDenied, "this game cannot be played here")
	}
	if err := db.CanPlayGame(ctx, player.ID, gameID, nil); err != nil {
		return nil, engine.Refuse(engine.ProblemDenied, err.Error())
	}

	candidates, httpErr := resolveApiKeyCandidates(ctx, player.ID, gameID, "")
	if httpErr != nil {
		return nil, engine.Refuse(engine.ProblemNoKey, httpErr.Message)
	}
	offer, chosen, err := firstWorkingOffer(ctx, engine.Genre(game.Genre), candidates)
	if err != nil {
		return nil, err
	}

	earlier, err := db.DeleteUserGameSessions(ctx, player.ID, gameID)
	if err != nil {
		log.Warn("failed to delete earlier sessions", "user_id", player.ID, "game_id", gameID, "error", err)
	}

	spec := gameSpec(game, playerLanguage(player, language), db.ResolveConstraint(ctx, player, nil))
	session, err := launch(ctx, spec, offer, db.NewSession{
		GameID:        game.ID,
		UserID:        player.ID,
		WorkshopID:    playerWorkshop(player),
		ApiKeyShareID: shareID(chosen),
		Platform:      string(offer.Platform),
		Language:      spec.Language,
	})
	if err != nil {
		return nil, err
	}
	return &Created{Session: session, Replaced: earlier}, nil
}

// CreateShareSession launches a session from a share link, for a guest
// (player nil) or a signed-in player playing the share. The share's key pays,
// and the play counts against the share's limit once the key is known to work.
func CreateShareSession(ctx context.Context, player *obj.User, token string, language string) (*Created, error) {
	play, httpErr := PrepareSharePlay(ctx, token, player)
	if httpErr != nil {
		return nil, refusalOf(httpErr)
	}
	candidate := resolvedKey{Share: play.Key, AiQualityTier: play.AiQualityTier, KeyType: obj.ApiKeyTypeSponsor}
	offer, _, err := firstWorkingOffer(ctx, engine.Genre(play.Game.Genre), []resolvedKey{candidate})
	if err != nil {
		return nil, err
	}
	if httpErr := play.Consume(ctx, language); httpErr != nil {
		return nil, refusalOf(httpErr)
	}

	spec := gameSpec(play.Game, playerLanguage(play.Player, language), db.ResolveConstraint(ctx, play.Player, play.GameShare))
	session, err := launch(ctx, spec, offer, db.NewSession{
		GameID:        play.Game.ID,
		UserID:        play.Player.ID,
		WorkshopID:    play.GameShare.WorkshopID,
		GameShareID:   &play.GameShare.ID,
		ApiKeyShareID: shareID(candidate),
		Platform:      string(offer.Platform),
		Language:      spec.Language,
	})
	if err != nil {
		return nil, err
	}
	return &Created{Session: session}, nil
}

// launch starts the session under a fresh id; its first save creates the row
// and counts the play, so a refused launch leaves nothing behind.
func launch(ctx context.Context, spec engine.SessionSpec, offer engine.Offer, meta db.NewSession) (*engine.Session, error) {
	spec.ID = uuid.NewString()
	spec.Persist = db.Launching(meta)
	session, err := engine.Launch(ctx, spec, []engine.Offer{offer})
	if err != nil {
		return nil, err
	}
	log.Info("session created", "session_id", spec.ID, "game_id", meta.GameID, "user_id", meta.UserID, "platform", meta.Platform, "offer", offer.ID)
	return session, nil
}

// ResumeSession restores a stored session that is not in memory, on an offer
// rebuilt as at creation: the share's key for a session from a share link,
// otherwise the owner's key cascade, restricted to the session's platform.
// The context outlives the request, as the session's lifetime.
func ResumeSession(ctx context.Context, sessionID uuid.UUID) (*engine.Session, error) {
	row, err := db.GetGameSession(ctx, sessionID)
	if err != nil {
		return nil, engine.Refuse(engine.ProblemUnknown, fmt.Sprintf("there is no session %q", sessionID))
	}

	var candidates []resolvedKey
	if share := shareOf(ctx, row); share != nil {
		candidates = []resolvedKey{*share}
	} else {
		var httpErr *obj.HTTPError
		candidates, httpErr = resolveApiKeyCandidates(ctx, row.UserID, row.GameID, row.Platform)
		if httpErr != nil {
			return nil, engine.Refuse(engine.ProblemNoKey, httpErr.Message)
		}
	}
	// The stored session already ran on this platform, so its genre fits.
	offer, _, err := firstWorkingOffer(ctx, "", candidates)
	if err != nil {
		return nil, err
	}
	return engine.Resume(ctx, sessionID.String(), db.SessionStore{}, []engine.Offer{offer})
}

// shareOf is the share key a session from a share link resumes on, or nil for
// a session outside a share or one whose share is gone.
func shareOf(ctx context.Context, row *obj.GameSession) *resolvedKey {
	if row.GameShareID == nil {
		return nil
	}
	gameShare, err := db.GetGameShareByID(ctx, *row.GameShareID)
	if err != nil {
		return nil
	}
	key, httpErr := resolvePrivateShareKey(ctx, gameShare)
	if httpErr != nil || key.ApiKey.Platform != row.Platform {
		return nil
	}
	return &resolvedKey{Share: key, AiQualityTier: shareTier(ctx, gameShare), KeyType: obj.ApiKeyTypeSponsor}
}

// firstWorkingOffer turns the key candidates, in order, into offers and checks
// each with its provider; the first that can run the genre and whose key works
// is the session's single offer. An empty genre skips the genre check.
// Without one, the first candidate's refusal answers.
func firstWorkingOffer(ctx context.Context, genre engine.Genre, candidates []resolvedKey) (engine.Offer, resolvedKey, error) {
	var first error
	refuse := func(err error) {
		if first == nil {
			first = err
		}
	}
	for _, c := range candidates {
		offer, toggles, ok := offerFor(c)
		if !ok {
			refuse(engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the engine does not run platform %q", c.Share.ApiKey.Platform)))
			continue
		}
		if genre != "" && !slices.Contains(engine.PlayableGenres(toggles), genre) {
			refuse(engine.Refuse(engine.ProblemGenre, fmt.Sprintf("the key on %s cannot run %s", offer.Platform, genre)))
			continue
		}
		err := engine.CheckOffer(ctx, offer)
		recordKeyCheck(ctx, c, err)
		if err != nil {
			log.Info("key check failed", "offer", offer.ID, "platform", offer.Platform, "source", c.KeyType, "error", err)
			refuse(err)
			continue
		}
		return offer, c, nil
	}
	if first == nil {
		first = engine.Refuse(engine.ProblemNoKey, "no API key is available for this game")
	}
	return engine.Offer{}, resolvedKey{}, first
}

// recordKeyCheck keeps the outcome on the key, as its last usage shows to its
// owner. An unreachable provider says nothing about the key.
func recordKeyCheck(ctx context.Context, c resolvedKey, err error) {
	switch {
	case err == nil:
		db.UpdateApiKeyLastUsageSuccess(ctx, c.Share.ApiKey.ID, true)
	case errors.Is(err, engine.ErrUnreachable):
	default:
		db.UpdateApiKeyLastUsageSuccess(ctx, c.Share.ApiKey.ID, false, string(engine.RefusalOf(err).Problem))
	}
}

// offerFor makes a candidate key an offer: that tier the only one allowed,
// every toggle its platform has on there by default, and no choice for the
// player. Its ID is the key share's, or the key's for the system free-use key,
// which has no share. The key is read on every call, so a rotated key applies
// at once.
func offerFor(c resolvedKey) (engine.Offer, engine.Toggles, bool) {
	platform := engine.Platform(c.Share.ApiKey.Platform)
	i := slices.IndexFunc(engine.Platforms(), func(p engine.PlatformInfo) bool { return p.ID == platform })
	if i < 0 {
		return engine.Offer{}, engine.Toggles{}, false
	}
	toggles := engine.Platforms()[i].Toggles
	tier := engine.Tier(c.AiQualityTier).Normal()
	if !tier.Known() {
		tier = engine.TierMedium
	}

	id := c.Share.ApiKey.ID.String()
	if c.Share.ID != uuid.Nil {
		id = c.Share.ID.String()
	}
	keyID := c.Share.ApiKey.ID
	return engine.Offer{
		Platform: platform,
		ID:       id,
		Keys: func(ctx context.Context) (string, error) {
			key, err := db.GetApiKeyByID(ctx, keyID)
			if err != nil {
				return "", fmt.Errorf("the API key is gone: %w", err)
			}
			return key.Key, nil
		},
		Caveats: engine.Caveats{
			Allow:   map[engine.Tier]engine.Toggles{tier: toggles},
			Default: engine.Choice{Tier: tier, Toggles: toggles},
		},
	}, toggles, true
}

// gameSpec assembles a session spec from a game, the player's language and
// the youth-protection rule resolved for this play.
func gameSpec(game *obj.Game, language string, rule db.ConstraintResolution) engine.SessionSpec {
	spec := engine.SessionSpec{
		Genre:      engine.Genre(game.Genre),
		Title:      game.Name,
		Scenario:   game.SystemMessageScenario,
		InitPrompt: game.SystemMessageGameStart,
		ImageStyle: game.ImageStyle,
		Language:   language,
		Status:     statusValues(game.StatusFields),
	}
	if game.Theme != nil {
		spec.Theme = *game.Theme
	}
	if rule.Text != nil {
		spec.Guardrail = *rule.Text
	}
	return spec
}

// statusValues reads a game's status fields as the values a session starts with.
func statusValues(statusFields string) map[string]string {
	var fields []obj.StatusField
	if err := json.Unmarshal([]byte(statusFields), &fields); err != nil || len(fields) == 0 {
		return nil
	}
	values := make(map[string]string, len(fields))
	for _, f := range fields {
		if f.Name != "" {
			values[f.Name] = f.Value
		}
	}
	return values
}

// playerLanguage is a signed-in player's profile language, else the one the
// playback-engine sent, else English.
func playerLanguage(player *obj.User, requested string) string {
	switch {
	case player != nil && player.Language != "":
		return player.Language
	case requested != "":
		return requested
	}
	return "en"
}

// playerWorkshop is the workshop a player plays in, if any.
func playerWorkshop(player *obj.User) *uuid.UUID {
	if player.Role == nil || player.Role.Workshop == nil {
		return nil
	}
	return &player.Role.Workshop.ID
}

// shareID is the key share that pays, nil for the system free-use key.
func shareID(c resolvedKey) *uuid.UUID {
	if c.Share == nil || c.Share.ID == uuid.Nil {
		return nil
	}
	id := c.Share.ID
	return &id
}

// refusalOf reads a platform error as the refusal the playback-engine shows.
func refusalOf(httpErr *obj.HTTPError) error {
	switch {
	case httpErr.Code == obj.ErrCodeNoApiKey:
		return engine.Refuse(engine.ProblemNoKey, httpErr.Message)
	case httpErr.Code == errCodeShareExhausted:
		return engine.Refuse(engine.ProblemNoPlays, httpErr.Message)
	case httpErr.StatusCode == 404 || httpErr.StatusCode == 403:
		return engine.Refuse(engine.ProblemDenied, httpErr.Message)
	}
	return engine.Refuse(engine.ProblemFailed, httpErr.Message)
}
