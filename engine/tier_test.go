package engine

import (
	"context"
	"encoding/json"
	"testing"

	"engine/internal/adapters"
)

func noKey(context.Context) (string, error) { return "k", nil }

// oneOffer is an offer at a single tier with the given toggles allowed and on.
func oneOffer(platform Platform, tier Tier, toggles Toggles) Offer {
	return Offer{
		Platform: platform,
		ID:       string(platform) + "-" + string(tier),
		Keys:     noKey,
		Caveats: Caveats{
			Allow:   map[Tier]Toggles{tier: toggles},
			Default: Choice{Tier: tier, Toggles: toggles},
		},
	}
}

var live = Toggles{Images: true, AudioLive: true}

// A session's models come from its platform's tier table for the settled tier,
// and nowhere else.
func TestModelsComeFromTheSettledTier(t *testing.T) {
	spec := game{spec: SessionSpec{Genre: GenreNPCLive}, offers: []Offer{oneOffer(PlatformOpenAI, TierHigh, live)}}
	if issues := check(spec); hasError(issues) {
		t.Fatal(issues)
	}
	settled, _, _ := spec.opening()

	entry, _ := catalogueEntry(PlatformOpenAI)
	want, _ := entry.Models(TierHigh)
	got, _ := entry.Models(settled.Tier)
	if got[ModelImage].Name != want[ModelImage].Name || got[ModelImage].Params[adapters.ParamQuality] != "high" {
		t.Errorf("image = %+v, want the high tier's %+v", got[ModelImage], want[ModelImage])
	}
}

// max is not offered, but stored data and offers that name it mean high, and
// Validate says so.
func TestMaxIsReadAsHigh(t *testing.T) {
	var offers []Offer
	blob := `[{"platform":"mock","id":"m","caveats":{
		"allow":{"max":{"images":true,"audio-live":true}},
		"default":{"tier":"max","toggles":{"images":true,"audio-live":true}}}}]`
	if err := json.Unmarshal([]byte(blob), &offers); err != nil {
		t.Fatal(err)
	}
	spec := game{spec: SessionSpec{Genre: GenreNPCLive}, offers: offers}

	issues := check(spec)
	if hasError(issues) {
		t.Fatalf("max must be accepted: %v", issues)
	}
	warned := map[string]bool{}
	for _, i := range issues {
		if i.Severity == SeverityWarning {
			warned[i.Field] = true
		}
	}
	for _, field := range []string{"offers[0].caveats.allow.max", "offers[0].caveats.default.tier"} {
		if !warned[field] {
			t.Errorf("no warning on %s: %v", field, issues)
		}
	}

	settled, _, _ := spec.opening()
	if settled.Tier != TierHigh {
		t.Errorf("settled tier = %q, want high", settled.Tier)
	}
}

func TestUnknownTierIsRefused(t *testing.T) {
	spec := game{spec: SessionSpec{Genre: GenreNPCLive}, offers: []Offer{oneOffer(PlatformMock, "luxurious", live)}}
	issues := check(spec)
	if !hasWarningAt(issues, "offers[0].caveats.allow.luxurious") || !hasErrorAt(issues, "offers") {
		t.Errorf("issues = %v, want the offer left out for its unknown tier", issues)
	}
}

func hasErrorAt(issues []Issue, field string) bool {
	for _, i := range issues {
		if i.Severity == SeverityError && i.Field == field {
			return true
		}
	}
	return false
}
