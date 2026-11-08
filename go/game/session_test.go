package game

import (
	"testing"

	"cgl/db"
	"cgl/obj"
	"engine"

	"github.com/google/uuid"
)

func candidate(platform, tier string, shareID uuid.UUID) resolvedKey {
	return resolvedKey{
		Share:         &obj.ApiKeyShare{ID: shareID, ApiKey: &obj.ApiKey{ID: uuid.New(), Platform: platform}},
		AiQualityTier: tier,
	}
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
	if offer.ID != c.Share.ApiKey.ID.String() {
		t.Errorf("offer ID %q, want the key's %q", offer.ID, c.Share.ApiKey.ID)
	}
	if _, ok := offer.Caveats.Allow[engine.TierHigh]; !ok {
		t.Errorf("max is not read as high: %v", offer.Caveats.Allow)
	}
	if shareID(c) != nil {
		t.Error("the system key names a share")
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
	game := &obj.Game{
		Genre:                  "npc-live",
		Name:                   "Bridge",
		SystemMessageScenario:  "A troll guards a bridge.",
		SystemMessageGameStart: "Greet the traveller.",
		ImageStyle:             "woodcut",
		Theme:                  &theme,
		StatusFields:           `[{"name":"Mood","value":"grumpy"}]`,
	}
	spec := gameSpec(game, "de", dbRule(&rule))
	want := engine.SessionSpec{
		Genre: engine.GenreNPCLive, Title: "Bridge", Scenario: "A troll guards a bridge.",
		InitPrompt: "Greet the traveller.", ImageStyle: "woodcut", Theme: "noir", Language: "de",
		Guardrail: rule, Status: map[string]string{"Mood": "grumpy"},
	}
	if spec.Genre != want.Genre || spec.Title != want.Title || spec.Scenario != want.Scenario ||
		spec.InitPrompt != want.InitPrompt || spec.ImageStyle != want.ImageStyle || spec.Theme != want.Theme ||
		spec.Language != want.Language || spec.Guardrail != want.Guardrail || spec.Status["Mood"] != "grumpy" {
		t.Errorf("spec %+v, want %+v", spec, want)
	}
}

func TestPlayerLanguage(t *testing.T) {
	signedIn := &obj.User{Language: "fr"}
	cases := []struct {
		player    *obj.User
		requested string
		want      string
	}{
		{signedIn, "de", "fr"},
		{&obj.User{}, "de", "de"},
		{nil, "", "en"},
	}
	for _, c := range cases {
		if got := playerLanguage(c.player, c.requested); got != c.want {
			t.Errorf("playerLanguage(%v, %q) = %q, want %q", c.player, c.requested, got, c.want)
		}
	}
}

func dbRule(text *string) db.ConstraintResolution { return db.ConstraintResolution{Text: text} }
