package core

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"engine"
)

// A payer becomes an offer the engine can run a session on. Starting a game
// and saying beforehand whether it can start go through the same steps: the
// payers in order (Payers), as offers (Offers), the first whose key works
// (FirstPlayable). Only the key check differs: a start asks the provider
// (session/session.go), a game view reads the key's last recorded check
// (Playability), so loading a list asks no provider.

// PayerOffer is a payer as the offer the engine runs a session on, with the
// toggles its platform has on.
type PayerOffer struct {
	Payer   Payer
	Offer   engine.Offer
	Toggles engine.Toggles
}

// OfferFor makes a payer an offer: its tier the only one allowed, every
// toggle its platform has on there by default, and no choice for the player.
// Its ID is the key share's, or the key's for the site's free-use key, which
// has no share. The key is read on every call, so a rotated key applies at
// once. A platform the engine does not run makes none.
func OfferFor(p Payer) (PayerOffer, bool) {
	platform := engine.Platform(p.Platform)
	i := slices.IndexFunc(engine.Platforms(), func(info engine.PlatformInfo) bool { return info.ID == platform })
	if i < 0 {
		return PayerOffer{}, false
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
	return PayerOffer{
		Payer: p,
		Offer: engine.Offer{
			Platform: platform,
			ID:       id,
			Keys:     p.Secret,
			Caveats: engine.Caveats{
				Allow:   map[engine.Tier]engine.Toggles{tier: toggles},
				Default: engine.Choice{Tier: tier, Toggles: toggles},
			},
		},
		Toggles: toggles,
	}, true
}

// Offers are the payers, in order, as offers that fit: a platform the engine
// runs, and whatever fits asks of it. refused is the first payer's refusal
// among those that do not fit, nil when all fit.
func Offers(payers []Payer, fits func(PayerOffer) error) (offers []PayerOffer, refused error) {
	for _, p := range payers {
		offer, ok := OfferFor(p)
		if !ok {
			refused = firstOf(refused, engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the engine does not run platform %q", p.Platform)))
			continue
		}
		if fits != nil {
			if err := fits(offer); err != nil {
				refused = firstOf(refused, err)
				continue
			}
		}
		offers = append(offers, offer)
	}
	return offers, refused
}

// RunsGenre fits an offer whose toggles run the genre; an empty genre fits
// every offer.
func RunsGenre(genre engine.Genre) func(PayerOffer) error {
	return func(o PayerOffer) error {
		if genre != "" && !slices.Contains(engine.PlayableGenres(o.Toggles), genre) {
			return engine.Refuse(engine.ProblemGenre, fmt.Sprintf("the key on %s cannot run %s", o.Offer.Platform, genre))
		}
		return nil
	}
}

// FirstPlayable is the first offer whose key check passes. Without one, the
// refusal of the payer earliest in the order answers: refused, the one the
// offers already carry, else the first failed check, else none, a no_key
// refusal saying so.
func FirstPlayable(offers []PayerOffer, refused error, check func(PayerOffer) error, none string) (PayerOffer, error) {
	for _, o := range offers {
		err := check(o)
		if err == nil {
			return o, nil
		}
		refused = firstOf(refused, err)
	}
	if refused == nil {
		refused = engine.Refuse(engine.ProblemNoKey, none)
	}
	return PayerOffer{}, refused
}

// LastCheck is the payer's key as its last recorded check left it: nil when
// it worked or was never checked, otherwise the refusal it was recorded
// with. It asks no provider.
func (p Payer) LastCheck() error {
	if p.lastWorked == nil || *p.lastWorked {
		return nil
	}
	problem := engine.Problem(p.lastProblem)
	if problem == "" {
		problem = engine.ProblemKeyRejected
	}
	return engine.Refuse(problem, fmt.Sprintf("the key %q failed its last check", p.KeyName))
}

// CheckLive asks the offer's provider whether it accepts the key, and
// records the outcome on the key, as its last usage shows to its owner. An
// unreachable provider says nothing about the key.
func CheckLive(ctx context.Context, o PayerOffer) error {
	err := engine.CheckOffer(ctx, o.Offer)
	switch {
	case err == nil:
		o.Payer.RecordCheck(ctx, true, "")
	case errors.Is(err, engine.ErrUnreachable):
	default:
		o.Payer.RecordCheck(ctx, false, string(engine.RefusalOf(err).Problem))
	}
	return err
}

func firstOf(first, next error) error {
	if first != nil {
		return first
	}
	return next
}
