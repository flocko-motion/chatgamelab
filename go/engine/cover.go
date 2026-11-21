package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"engine/internal/adapters"
	"engine/internal/genre"
	"engine/internal/ports"
)

// CoverSpec is what a game's cover is made from: the game's own words, the
// youth-protection rule the picture keeps to, and the designer's hint.
type CoverSpec struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Scenario    string `json:"scenario"`
	ImageStyle  string `json:"imageStyle"`
	// Guardrail is the youth-protection rule, already resolved.
	Guardrail string `json:"guardrail"`
	// Hint is what the designer wants the cover to show; empty leaves it to
	// the game.
	Hint string `json:"hint,omitempty"`
	// MockPaceMs slows the mock platform down, as in a session.
	MockPaceMs int `json:"mockPaceMs,omitempty"`
}

// Cover is a generated cover: the picture and what it cost.
type Cover struct {
	Image []byte
	Usage UsageReport
}

// The names a cover's calls are billed under.
const (
	coverNodeStyle    = "cover-style"
	coverNodeCondense = "cover-condense"
	coverNodeImage    = "cover-image"
)

// CoverTier is the tier a cover is drawn at on the offer: the cheapest tier
// that allows pictures, whatever the offer's default. A cover is one small
// picture on a card, so the cheapest and fastest model does it well. False
// when the offer allows no pictures at all.
func CoverTier(offer Offer) (Tier, bool) {
	for _, tier := range offer.Caveats.tiers() {
		if toggles, _ := offer.Caveats.allowed(tier); toggles.Images {
			return tier, true
		}
	}
	return "", false
}

// MakeCover draws a game's cover on one offer, outside any session: it
// adapts the image style to the rule and condenses a long scenario, as an
// Adventure's preparation does, then draws one picture inside the cover
// frame. An offer that allows no pictures is refused with ProblemNoKey. It
// blocks until the picture is made; the caller checks the key beforehand.
func MakeCover(ctx context.Context, spec CoverSpec, offer Offer) (Cover, error) {
	for _, issue := range offerIssues("offer", offer) {
		if issue.Severity == SeverityError {
			return Cover{}, fmt.Errorf("%s", issue)
		}
	}
	tier, ok := CoverTier(offer)
	if !ok {
		return Cover{}, Refuse(ProblemNoKey, fmt.Sprintf("offer %q allows no pictures", offer.ID))
	}
	toggles, _ := offer.Caveats.allowed(tier)
	kit, err := SessionSpec{MockPaceMs: spec.MockPaceMs}.adapters(settlement{offer: offer, Choice: Choice{Tier: tier, Toggles: toggles}})
	if err != nil {
		return Cover{}, err
	}
	defer kit.close()
	if kit.image == nil || kit.tool == nil {
		return Cover{}, Refuse(ProblemNoKey, fmt.Sprintf("platform %q draws no pictures", offer.Platform))
	}

	ledger := newUsageLedger()
	ledger.price(offer.Platform)
	bill := func(node string, used adapters.Usage) {
		ledger.record(ports.Usage{
			Node: node, Model: used.Model,
			InputTokens: used.InputTokens, CachedInputTokens: used.CachedInputTokens, OutputTokens: used.OutputTokens,
			AudioInputTokens: used.AudioInputTokens, CachedAudioInputTokens: used.CachedAudioInputTokens,
			AudioOutputTokens: used.AudioOutputTokens, AudioSeconds: used.AudioSeconds, Images: used.Images,
		})
	}

	style, used, err := withRetry(ctx, func() (string, adapters.Usage, error) {
		return kit.tool.Query(ctx, genre.CoverStylePrompt(), genre.CoverStyleMessage(spec.ImageStyle, spec.Guardrail))
	})
	bill(coverNodeStyle, used)
	if err != nil {
		return Cover{Usage: ledger.report()}, coverFailure("adapting the image style", err)
	}

	scenario := spec.Scenario
	if genre.CoverNeedsCondensing(scenario) {
		condensed, used, err := withRetry(ctx, func() (string, adapters.Usage, error) {
			return kit.tool.Query(ctx, genre.CoverCondensePrompt(), scenario)
		})
		bill(coverNodeCondense, used)
		if err != nil {
			return Cover{Usage: ledger.report()}, coverFailure("condensing the scenario", err)
		}
		scenario = condensed
	}

	prompt := genre.CoverPrompt(spec.Title, spec.Description, scenario, spec.Hint, strings.TrimSpace(style))
	picture, used, err := withRetry(ctx, func() ([]byte, adapters.Usage, error) {
		return kit.image.Generate(ctx, adapters.ImageRequest{Prompt: prompt})
	})
	bill(coverNodeImage, used)
	if err != nil {
		return Cover{Usage: ledger.report()}, coverFailure("drawing the cover", err)
	}
	return Cover{Image: picture, Usage: ledger.report()}, nil
}

// coverFailure names a failed cover call's problem: a key the provider
// rejects and a provider that does not answer are refusals of their own, so
// the platform tells them apart; anything else is the error as it is.
func coverFailure(what string, err error) error {
	switch {
	case errors.Is(err, adapters.ErrKeyRejected):
		return Refuse(ProblemKeyRejected, what+": "+err.Error())
	case errors.Is(err, adapters.ErrUnreachable):
		return Refuse(ProblemUnreachable, what+": "+err.Error())
	}
	return fmt.Errorf("%s: %w", what, err)
}

// coverAttempts is how often a cover's call is made before it fails.
const coverAttempts = 2

// withRetry makes a cover's call again once after a failure, unless the
// context ended or the provider rejected the key; the usage of every attempt
// counts.
func withRetry[T any](ctx context.Context, call func() (T, adapters.Usage, error)) (T, adapters.Usage, error) {
	var total adapters.Usage
	var out T
	var err error
	for attempt := 0; attempt < coverAttempts; attempt++ {
		var used adapters.Usage
		out, used, err = call()
		total.Add(used)
		if err == nil || ctx.Err() != nil || errors.Is(err, adapters.ErrKeyRejected) {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	return out, total, err
}
