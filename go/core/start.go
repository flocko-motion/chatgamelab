package core

import (
	"context"

	"engine"

	"github.com/google/uuid"
)

// Starting a game and showing whether it can start are one decision. The
// start (session/session.go) and the game views (Playability) both ask
// StartGame or StartShare how a new session would start, and pick the first
// offer whose key works with FirstPlayable; only how a key counts as working
// differs (CheckLive or LastCheck).

// NoKey is what a start without any payer says.
const NoKey = "no API key is available for this game"

// GameStart is how a new session of a game would start for the caller: the
// offers it may run on, in order, and the refusal of the payers that do not
// fit, earliest first.
type GameStart struct {
	Game    *GameCard
	Offers  []PayerOffer
	Refused error
	Steps   []PayerStep // why each source of payers did or did not count, for telemetry
}

// Playable is whether the start would go ahead, by the keys' last recorded
// checks: nil, or the refusal a start would answer. It asks no provider.
func (s *GameStart) Playable() error {
	_, err := FirstPlayable(s.Offers, s.Refused, func(o PayerOffer) error { return o.Payer.LastCheck() }, NoKey)
	return err
}

// StartGame is how the caller would start a new session of game: refused
// (denied) when they may not play it ({Play, GameCard}), or the workshop
// they are in is paused (paused), otherwise with the payers' offers that
// run its genre.
func StartGame(ctx context.Context, game uuid.UUID) (*GameStart, error) {
	card, err := ReadGameCard(ctx, game)
	if err != nil {
		return nil, engine.Refuse(engine.ProblemDenied, "this game cannot be played here")
	}
	s := newStarter(ctx)
	return s.start(ctx, card)
}

// StartShare is how the caller would start a new session through a share
// link: refused when the link has no plays left (no_plays) or its workshop
// is paused (paused), otherwise with the link's key as the one offer, if it
// runs the game's genre.
func StartShare(ctx context.Context, link *LinkPlay) (*GameStart, error) {
	if link.Remaining != nil && *link.Remaining <= 0 {
		return nil, engine.Refuse(engine.ProblemNoPlays, "this share link has reached its play limit")
	}
	if err := MayStartPlayingByLink(ctx, link.WorkshopID); err != nil {
		return nil, err
	}
	offers, refused := Offers([]Payer{link.Payer}, RunsGenre(engine.Genre(link.Game.Genre())))
	return &GameStart{Game: link.Game, Offers: offers, Refused: refused}, nil
}

// Playability is, for each game card, whether the caller can start it now:
// nil, or the refusal a start would answer (denied, paused, no_key, genre,
// key_rejected, …). Like the start it follows the payers, but it reads each
// key's last recorded check instead of asking the provider, so a whole list
// costs no provider call; the caller's side of the payers is read once.
func Playability(ctx context.Context, cards ...*GameCard) map[uuid.UUID]error {
	out := make(map[uuid.UUID]error, len(cards))
	if len(cards) == 0 {
		return out
	}
	s := newStarter(ctx)
	for _, card := range cards {
		start, err := s.start(ctx, card)
		if err == nil {
			err = start.Playable()
		}
		out[card.ID()] = PlayRefusal(err)
	}
	return out
}

// SharePlayability is whether the caller can start a new session through
// a share link now: nil, or the refusal a start would answer (no_plays,
// paused, genre, key_rejected, …), by the link key's last recorded check.
func SharePlayability(ctx context.Context, link *LinkPlay) error {
	start, err := StartShare(ctx, link)
	if err == nil {
		err = start.Playable()
	}
	return PlayRefusal(err)
}

// starter holds what every start of the caller shares: whether the workshop
// they are in is paused, and their side of the payers. Each is read once.
type starter struct {
	paused  error
	cascade *cascade
	err     error
}

func newStarter(ctx context.Context) *starter {
	s := &starter{paused: checkNotPaused(ctx, from(ctx).CurrentWorkshop())}
	s.cascade, s.err = loadCascade(ctx)
	return s
}

func (s *starter) start(ctx context.Context, card *GameCard) (*GameStart, error) {
	if !card.MayPlay() {
		return nil, engine.Refuse(engine.ProblemDenied, "this game cannot be played here")
	}
	if s.paused != nil {
		return nil, s.paused
	}
	if s.err != nil {
		return nil, s.err
	}
	payers, steps, err := s.cascade.payers(ctx, card.ID(), "")
	if err != nil {
		return nil, err
	}
	offers, refused := Offers(payers, RunsGenre(engine.Genre(card.Genre())))
	return &GameStart{Game: card, Offers: offers, Refused: refused, Steps: steps}, nil
}
