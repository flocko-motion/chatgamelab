package core

import (
	"database/sql"
	"testing"

	"engine"

	"github.com/google/uuid"
)

func candidate(platform, tier string, shareID uuid.UUID) Payer {
	p := Payer{KeyID: uuid.New(), KeyName: platform + " key", Platform: platform, Tier: tier}
	if shareID != uuid.Nil {
		p.ShareID = &shareID
	}
	return p
}

// checked is a payer whose key's last recorded check worked or failed with problem.
func checked(p Payer, worked bool, problem string) Payer {
	return p.lastChecked(sql.NullBool{Bool: worked, Valid: true}, sql.NullString{String: problem, Valid: problem != ""})
}

func problemOf(t *testing.T, err error) engine.Problem {
	t.Helper()
	if err == nil {
		return ""
	}
	return engine.RefusalOf(err).Problem
}

// An offer allows the resolved tier alone, with every toggle its platform
// has, on by default, and leaves the player nothing to choose.
func TestOfferForAllowsTheResolvedTierOnly(t *testing.T) {
	share := uuid.New()
	o, ok := OfferFor(candidate("openai", "high", share))
	if !ok {
		t.Fatal("openai is no platform of the engine's")
	}
	offer, toggles := o.Offer, o.Toggles
	if offer.ID != share.String() {
		t.Errorf("offer ID %q, want the key share's %q", offer.ID, share)
	}
	if len(offer.Caveats.Allow) != 1 || offer.Caveats.Allow[engine.TierHigh] != toggles {
		t.Errorf("allowance %v, want only high with %v", offer.Caveats.Allow, toggles)
	}
	if offer.Caveats.Default != (engine.Choice{Tier: engine.TierHigh, Toggles: toggles}) {
		t.Errorf("default %v", offer.Caveats.Default)
	}
	if offer.Caveats.Customizable {
		t.Error("the player may customize")
	}
	if issues := engine.Validate(engine.SessionSpec{Genre: engine.GenreAdventure}, []engine.Offer{offer}); len(issues) > 0 {
		t.Errorf("the engine finds issues: %v", issues)
	}
}

// The system free-use key has no share, so its offer is named by the key.
func TestOfferForTheSystemKeyIsNamedByTheKey(t *testing.T) {
	c := candidate("openai", "max", uuid.Nil)
	o, _ := OfferFor(c)
	if o.Offer.ID != c.KeyID.String() {
		t.Errorf("offer ID %q, want the key's %q", o.Offer.ID, c.KeyID)
	}
	if _, ok := o.Offer.Caveats.Allow[engine.TierHigh]; !ok {
		t.Errorf("max is not read as high: %v", o.Offer.Caveats.Allow)
	}
}

func TestOfferForAnUnknownPlatform(t *testing.T) {
	if _, ok := OfferFor(candidate("nonesuch", "low", uuid.New())); ok {
		t.Error("an unknown platform made an offer")
	}
}

// A key never checked, or whose last check worked, counts as working: the
// game list asks no provider, and the start checks it for real.
func TestAKeyNeverCheckedOrWorkingIsPlayable(t *testing.T) {
	for name, p := range map[string]Payer{
		"never checked": candidate("mock", "medium", uuid.New()),
		"worked":        checked(candidate("mock", "medium", uuid.New()), true, ""),
	} {
		start := &GameStart{}
		start.Offers, start.Refused = Offers([]Payer{p}, RunsGenre(engine.GenreAdventure))
		if err := start.Playable(); err != nil {
			t.Errorf("%s: %v, want playable", name, err)
		}
	}
}

// A key whose last check failed stops the game with the problem it was
// recorded with.
func TestAKeyThatFailedItsLastCheckSaysWhy(t *testing.T) {
	start := &GameStart{}
	start.Offers, start.Refused = Offers([]Payer{checked(candidate("mock", "medium", uuid.New()), false, "key_rejected")}, nil)
	if got := problemOf(t, start.Playable()); got != engine.ProblemKeyRejected {
		t.Errorf("problem %q, want key_rejected", got)
	}
}

// A failed key does not stop the game while a later payer's key works.
func TestALaterWorkingKeyStillPlays(t *testing.T) {
	failed := checked(candidate("mock", "medium", uuid.New()), false, "key_rejected")
	working := candidate("mock", "medium", uuid.New())
	start := &GameStart{}
	start.Offers, start.Refused = Offers([]Payer{failed, working}, nil)
	if err := start.Playable(); err != nil {
		t.Errorf("%v, want playable on the second key", err)
	}
}

// Without a working key, the payer earliest in the order says why: here the
// first cannot run the genre, so genre answers, not the second's rejection.
func TestTheEarliestPayersRefusalAnswers(t *testing.T) {
	unfit := candidate("mock", "medium", uuid.New())
	failed := checked(candidate("mock", "medium", uuid.New()), false, "key_rejected")
	notGenre := func(o PayerOffer) error {
		if o.Payer.KeyID == unfit.KeyID {
			return engine.Refuse(engine.ProblemGenre, "not this genre")
		}
		return nil
	}
	start := &GameStart{}
	start.Offers, start.Refused = Offers([]Payer{unfit, failed}, notGenre)
	if got := problemOf(t, start.Playable()); got != engine.ProblemGenre {
		t.Errorf("problem %q, want genre", got)
	}
}

// No payer at all is no key.
func TestNoPayerIsNoKey(t *testing.T) {
	start := &GameStart{}
	if got := problemOf(t, start.Playable()); got != engine.ProblemNoKey {
		t.Errorf("problem %q, want no_key", got)
	}
}
