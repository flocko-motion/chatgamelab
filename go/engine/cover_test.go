package engine

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"strings"
	"testing"

	"engine/internal/genre"
)

func TestMakeCoverOnTheMock(t *testing.T) {
	cover, err := MakeCover(context.Background(), CoverSpec{
		Title: "Lighthouse", Description: "Keep the light burning", Scenario: "A storm.", ImageStyle: "watercolour",
		Guardrail: "No violence.", Hint: "the lighthouse at night",
	}, MockOffer())
	if err != nil {
		t.Fatalf("MakeCover: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(cover.Image)); err != nil {
		t.Fatalf("the cover is no picture: %v", err)
	}
	nodes := map[string]bool{}
	for _, r := range cover.Usage.ByNode {
		nodes[r.Node] = true
	}
	if !nodes[coverNodeStyle] || !nodes[coverNodeImage] || nodes[coverNodeCondense] {
		t.Errorf("billed nodes %v, want style and image, no condensing for a short scenario", nodes)
	}
}

func TestMakeCoverCondensesALongScenario(t *testing.T) {
	cover, err := MakeCover(context.Background(), CoverSpec{Title: "Long", Scenario: strings.Repeat("a castle ", 80)}, MockOffer())
	if err != nil {
		t.Fatalf("MakeCover: %v", err)
	}
	for _, r := range cover.Usage.ByNode {
		if r.Node == coverNodeCondense {
			return
		}
	}
	t.Errorf("a long scenario was not condensed: %+v", cover.Usage.ByNode)
}

func TestMakeCoverRefusesAnOfferWithoutPictures(t *testing.T) {
	offer := MockOffer()
	for tier, toggles := range offer.Caveats.Allow {
		toggles.Images = false
		offer.Caveats.Allow[tier] = toggles
	}
	offer.Caveats.Default.Toggles.Images = false
	if _, ok := CoverTier(offer); ok {
		t.Fatal("CoverTier found a tier with pictures on an offer that allows none")
	}
	_, err := MakeCover(context.Background(), CoverSpec{Title: "No pictures"}, offer)
	var refusal *Refusal
	if !errors.As(err, &refusal) || refusal.Problem != ProblemNoKey {
		t.Errorf("got %v, want a no_key refusal", err)
	}
}

func TestCoverPromptFrame(t *testing.T) {
	prompt := genre.CoverPrompt("Lighthouse", "Keep the light", "A storm.", "the lamp", "")
	for _, want := range []string{"cover art", "Lighthouse", "Keep the light", "A storm.", "The cover should show: the lamp", genre.AdventureImageStyle, "no text"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, prompt)
		}
	}
}

// A cover is drawn at the cheapest tier that allows pictures, whatever the
// offer's default.
func TestCoverTierIsTheCheapestWithPictures(t *testing.T) {
	offer := MockOffer()
	offer.Caveats.Default.Tier = TierHigh
	if tier, _ := CoverTier(offer); tier != TierLow {
		t.Errorf("got %q, want the cheapest tier, %q", tier, TierLow)
	}

	low := offer.Caveats.Allow[TierLow]
	low.Images = false
	offer.Caveats.Allow[TierLow] = low
	if tier, _ := CoverTier(offer); tier != TierMedium {
		t.Errorf("got %q, want the cheapest tier with pictures, %q", tier, TierMedium)
	}
}
