package session

import (
	"testing"

	"cgl/core"
	"engine"

	"github.com/google/uuid"
)

func candidate(platform, tier string, shareID uuid.UUID) core.Payer {
	p := core.Payer{KeyID: uuid.New(), Platform: platform, Tier: tier}
	if shareID != uuid.Nil {
		p.ShareID = &shareID
	}
	return p
}

// An offer allows the resolved tier alone, with every toggle its platform
// has, on by default, and leaves the player nothing to choose.
func TestOfferForAllowsTheResolvedTierOnly(t *testing.T) {
	share := uuid.New()
	offer, toggles, ok := offerFor(candidate("openai", "high", share))
	if !ok {
		t.Fatal("openai is no platform of the engine's")
	}
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
	offer, _, _ := offerFor(c)
	if offer.ID != c.KeyID.String() {
		t.Errorf("offer ID %q, want the key's %q", offer.ID, c.KeyID)
	}
	if _, ok := offer.Caveats.Allow[engine.TierHigh]; !ok {
		t.Errorf("max is not read as high: %v", offer.Caveats.Allow)
	}
}

func TestOfferForAnUnknownPlatform(t *testing.T) {
	if _, _, ok := offerFor(candidate("nonesuch", "low", uuid.New())); ok {
		t.Error("an unknown platform made an offer")
	}
}

func TestGameSpec(t *testing.T) {
	theme := "noir"
	rule := "Keep it gentle."
	game := &gameFacts{
		Genre:        "npc-live",
		Name:         "Bridge",
		Description:  "Talk your way past a troll.",
		Scenario:     "A troll guards a bridge.",
		GameStart:    "Greet the traveller.",
		ImageStyle:   "woodcut",
		Theme:        &theme,
		StatusFields: `[{"name":"Mood","value":"grumpy"}]`,
	}
	spec := gameSpec(game, "de", Rule{Text: &rule})
	want := engine.SessionSpec{
		Genre: engine.GenreNPCLive, Title: "Bridge", Scenario: "A troll guards a bridge.",
		InitPrompt: "Greet the traveller.", ImageStyle: "woodcut", Theme: "noir", Language: "de",
		Guardrail: rule, Status: map[string]string{"Mood": "grumpy"},
	}
	if spec.Genre != want.Genre || spec.Title != want.Title || spec.Scenario != want.Scenario ||
		spec.InitPrompt != want.InitPrompt || spec.ImageStyle != want.ImageStyle || spec.Theme != want.Theme ||
		spec.Language != want.Language || spec.Guardrail != want.Guardrail || spec.Status["Mood"] != "grumpy" ||
		spec.Description != "Talk your way past a troll." {
		t.Errorf("spec %+v, want %+v", spec, want)
	}
}

func TestPlayerLanguage(t *testing.T) {
	cases := []struct{ profile, requested, want string }{
		{"fr", "de", "fr"},
		{"", "de", "de"},
		{"", "", "en"},
	}
	for _, c := range cases {
		if got := playerLanguage(c.profile, c.requested); got != c.want {
			t.Errorf("playerLanguage(%q, %q) = %q, want %q", c.profile, c.requested, got, c.want)
		}
	}
}
