// package: session / engine sessions from the platform's data
// type:    logic
// job:     creates and resumes engine sessions: one checked offer from the payers, the rule, the spec.
// limits:  no HTTP; every read and write and every access decision is the secure core's (core/play.go,
//
//	core/session.go), for the caller in the context; running the session is the engine's.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"cgl/core"
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

// CreateGameSession launches a session of a game for the caller. As starting
// a game anew, it replaces their earlier sessions of that game. The context
// outlives the request, as the session's lifetime.
func CreateGameSession(ctx context.Context, gameID uuid.UUID, language string) (*Created, error) {
	if err := core.MayStartPlaying(ctx, gameID); err != nil {
		return nil, err
	}
	game, err := readGame(ctx, func() (*core.GameCard, error) { return core.ReadGameCard(ctx, gameID) })
	if err != nil {
		return nil, err
	}
	payers, steps, err := core.Payers(ctx, gameID, "")
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	traceKeys(gameID, "", payers, steps)
	offer, payer, err := firstWorkingOffer(ctx, engine.Genre(game.Genre), payers)
	if err != nil {
		return nil, err
	}
	return launch(ctx, game, offer, payer, nil, language, true)
}

// CreateShareSession launches a session from a share link, for a guest or a
// signed-in player playing it. The link's key pays, and the play counts
// against the link's limit with the launch.
func CreateShareSession(ctx context.Context, token string, language string) (*Created, error) {
	link, err := core.ShareLinkPlay(ctx, token)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	if link.Remaining != nil && *link.Remaining <= 0 {
		return nil, engine.Refuse(engine.ProblemNoPlays, "this share link has reached its play limit")
	}
	game, err := readGame(ctx, func() (*core.GameCard, error) { return link.Game, nil })
	if err != nil {
		return nil, err
	}
	offer, payer, err := firstWorkingOffer(ctx, engine.Genre(game.Genre), []core.Payer{link.Payer})
	if err != nil {
		return nil, err
	}
	return launch(ctx, game, offer, payer, link, language, false)
}

// launch starts the session under a fresh id, in the store the core hands
// out for it: its first save creates the session and does what the start
// takes, all in one transaction, so a refused launch leaves nothing behind.
func launch(ctx context.Context, game *gameFacts, offer engine.Offer, payer core.Payer, link *core.LinkPlay, language string, replace bool) (*Created, error) {
	facts, err := core.ReadPlayRuleFacts(ctx, link)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	spec := gameSpec(game, playerLanguage(core.PlayerLanguage(ctx), language), resolveRule(facts))
	store, player, err := core.LaunchStore(ctx, core.NewPlay{
		Game: game.ID, Link: link, Payer: payer, Platform: string(offer.Platform),
		Language: spec.Language, ReplaceEarlier: replace,
	})
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	spec.ID = uuid.NewString()
	spec.Persist = store
	session, err := engine.Launch(ctx, spec, []engine.Offer{offer})
	if errors.Is(err, core.ErrNoPlaysLeft) {
		return nil, engine.Refuse(engine.ProblemNoPlays, "this share link has reached its play limit")
	}
	if err != nil {
		return nil, err
	}
	log.Info("session created", "session_id", spec.ID, "game_id", game.ID, "user_id", player, "platform", offer.Platform, "offer", offer.ID)
	return &Created{Session: session, Replaced: core.Replaced(store)}, nil
}

// ResumeSession restores a stored session that is not in memory, for a
// caller the session guard lets play it, on an offer rebuilt as at creation:
// the link's key for a session from a share link while the link pays,
// otherwise the caller's payers on the session's platform. The context
// outlives the request, as the session's lifetime.
func ResumeSession(ctx context.Context, sessionID uuid.UUID) (*engine.Session, error) {
	r, err := core.ResumeSession(ctx, sessionID)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	var payers []core.Payer
	if r.LinkPays != nil {
		payers = []core.Payer{*r.LinkPays}
	} else {
		var steps []core.PayerStep
		payers, steps, err = core.Payers(ctx, r.GameID, r.Platform)
		if err != nil {
			return nil, core.PlayRefusal(err)
		}
		traceKeys(r.GameID, r.Platform, payers, steps)
	}
	// The stored session already ran on this platform, so its genre fits.
	offer, _, err := firstWorkingOffer(ctx, "", payers)
	if err != nil {
		return nil, err
	}
	return engine.Resume(ctx, sessionID.String(), r.Store, []engine.Offer{offer})
}

// gameFacts are what a session's spec is built from.
type gameFacts struct {
	ID           uuid.UUID
	Genre        string
	Name         string
	Description  string
	Scenario     string
	GameStart    string
	ImageStyle   string
	Theme        *string
	StatusFields string
}

// readGame reads a game the caller may play as its spec needs it: playing
// it includes reading its content.
func readGame(ctx context.Context, read func() (*core.GameCard, error)) (*gameFacts, error) {
	card, err := read()
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	content, err := card.AsGameContent()
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	return &gameFacts{
		ID: card.ID(), Genre: card.Genre(), Name: card.Name(), Description: card.Description(), Theme: card.Theme(),
		Scenario: content.Scenario(), GameStart: content.GameStart(), ImageStyle: content.ImageStyle(),
		StatusFields: content.StatusFields(),
	}, nil
}

// firstWorkingOffer turns the payers, in order, into offers and checks each
// with its provider; the first that can run the genre and whose key works is
// the session's single offer. An empty genre skips the genre check. Without
// one, the first payer's refusal answers.
func firstWorkingOffer(ctx context.Context, genre engine.Genre, payers []core.Payer) (engine.Offer, core.Payer, error) {
	return firstOffer(ctx, payers, func(offer engine.Offer, toggles engine.Toggles) error {
		if genre != "" && !slices.Contains(engine.PlayableGenres(toggles), genre) {
			return engine.Refuse(engine.ProblemGenre, fmt.Sprintf("the key on %s cannot run %s", offer.Platform, genre))
		}
		return nil
	}, "no API key is available for this game")
}

// firstOffer turns the payers, in order, into offers; the first that fits
// and whose key its provider accepts is the one. Without one, the first
// payer's refusal answers, or none, a no_key refusal saying none.
func firstOffer(ctx context.Context, payers []core.Payer, fits func(engine.Offer, engine.Toggles) error, none string) (engine.Offer, core.Payer, error) {
	var first error
	refuse := func(err error) {
		if first == nil {
			first = err
		}
	}
	for _, p := range payers {
		offer, toggles, ok := offerFor(p)
		if !ok {
			refuse(engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the engine does not run platform %q", p.Platform)))
			continue
		}
		if err := fits(offer, toggles); err != nil {
			refuse(err)
			continue
		}
		err := engine.CheckOffer(ctx, offer)
		recordKeyCheck(ctx, p, err)
		if err != nil {
			log.Info("key check failed", "offer", offer.ID, "platform", offer.Platform, "source", p.Source, "error", err)
			refuse(err)
			continue
		}
		return offer, p, nil
	}
	if first == nil {
		first = engine.Refuse(engine.ProblemNoKey, none)
	}
	return engine.Offer{}, core.Payer{}, first
}

// recordKeyCheck keeps the outcome on the key, as its last usage shows to its
// owner. An unreachable provider says nothing about the key.
func recordKeyCheck(ctx context.Context, p core.Payer, err error) {
	switch {
	case err == nil:
		p.RecordCheck(ctx, true, "")
	case errors.Is(err, engine.ErrUnreachable):
	default:
		p.RecordCheck(ctx, false, string(engine.RefusalOf(err).Problem))
	}
}

// offerFor makes a payer an offer: its tier the only one allowed, every
// toggle its platform has on there by default, and no choice for the player.
// Its ID is the key share's, or the key's for the site's free-use key, which
// has no share. The key is read on every call, so a rotated key applies at
// once.
func offerFor(p core.Payer) (engine.Offer, engine.Toggles, bool) {
	platform := engine.Platform(p.Platform)
	i := slices.IndexFunc(engine.Platforms(), func(info engine.PlatformInfo) bool { return info.ID == platform })
	if i < 0 {
		return engine.Offer{}, engine.Toggles{}, false
	}
	toggles := engine.Platforms()[i].Toggles
	tier := engine.Tier(p.Tier).Normal()
	if !tier.Known() {
		tier = engine.TierMedium
	}
	id := p.KeyID.String()
	if p.ShareID != nil {
		id = p.ShareID.String()
	}
	return engine.Offer{
		Platform: platform,
		ID:       id,
		Keys:     p.Secret,
		Caveats: engine.Caveats{
			Allow:   map[engine.Tier]engine.Toggles{tier: toggles},
			Default: engine.Choice{Tier: tier, Toggles: toggles},
		},
	}, toggles, true
}

// gameSpec assembles a session spec from a game, the player's language and
// the youth-protection rule resolved for this play.
func gameSpec(game *gameFacts, language string, rule Rule) engine.SessionSpec {
	spec := engine.SessionSpec{
		Genre:       engine.Genre(game.Genre),
		Title:       game.Name,
		Description: game.Description,
		Scenario:    game.Scenario,
		InitPrompt:  game.GameStart,
		ImageStyle:  game.ImageStyle,
		Language:    language,
		Status:      statusValues(game.StatusFields),
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

// playerLanguage is the player's profile language, else the one the
// playback-engine sent, else English.
func playerLanguage(profile, requested string) string {
	switch {
	case profile != "":
		return profile
	case requested != "":
		return requested
	}
	return "en"
}
