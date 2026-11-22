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
	start, err := core.StartGame(ctx, gameID)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	traceKeys(gameID, "", start.Steps)
	game, err := readGame(ctx, func() (*core.GameCard, error) { return start.Game, nil })
	if err != nil {
		return nil, err
	}
	chosen, err := firstWorking(ctx, start.Offers, start.Refused, core.NoKey)
	if err != nil {
		return nil, err
	}
	return launch(ctx, game, chosen.Offer, chosen.Payer, nil, language, true)
}

// CreateShareSession launches a session from a share link, for a guest or a
// signed-in player playing it. The link's key pays, and the play counts
// against the link's limit with the launch.
func CreateShareSession(ctx context.Context, token string, language string) (*Created, error) {
	link, err := core.ShareLinkPlay(ctx, token)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	start, err := core.StartShare(ctx, link)
	if err != nil {
		return nil, core.PlayRefusal(err)
	}
	game, err := readGame(ctx, func() (*core.GameCard, error) { return start.Game, nil })
	if err != nil {
		return nil, err
	}
	chosen, err := firstWorking(ctx, start.Offers, start.Refused, core.NoKey)
	if err != nil {
		return nil, err
	}
	return launch(ctx, game, chosen.Offer, chosen.Payer, link, language, false)
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
		traceKeys(r.GameID, r.Platform, steps)
	}
	// The stored session already ran on this platform, so its genre fits.
	offers, refused := core.Offers(payers, nil)
	chosen, err := firstWorking(ctx, offers, refused, core.NoKey)
	if err != nil {
		return nil, err
	}
	return engine.Resume(ctx, sessionID.String(), r.Store, []engine.Offer{chosen.Offer})
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

// firstWorking is the first offer whose key its provider accepts, asked
// now; each check is recorded on its key. Without one, the refusal of the
// payer earliest in the order answers.
func firstWorking(ctx context.Context, offers []core.PayerOffer, refused error, none string) (core.PayerOffer, error) {
	return core.FirstPlayable(offers, refused, func(o core.PayerOffer) error {
		err := core.CheckLive(ctx, o)
		if err != nil {
			log.Info("key check failed", "offer", o.Offer.ID, "platform", o.Offer.Platform, "source", o.Payer.Source, "error", err)
		}
		return err
	}, none)
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
