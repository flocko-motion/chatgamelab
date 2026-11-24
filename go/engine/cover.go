package engine

import (
	"context"
	"encoding/json"
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
	case errors.Is(err, adapters.ErrNoCredit):
		return Refuse(ProblemNoCredit, what+": "+err.Error())
	case errors.Is(err, adapters.ErrUnreachable):
		return Refuse(ProblemUnreachable, what+": "+err.Error())
	}
	return fmt.Errorf("%s: %w", what, err)
}

// coverAttempts is how often a cover's call is made before it fails.
const coverAttempts = 2

// withRetry makes a cover's call again once after a failure, unless the
// context ended, the provider rejected the key or the account has no credit;
// the usage of every attempt counts.
func withRetry[T any](ctx context.Context, call func() (T, adapters.Usage, error)) (T, adapters.Usage, error) {
	var total adapters.Usage
	var out T
	var err error
	for attempt := 0; attempt < coverAttempts; attempt++ {
		var used adapters.Usage
		out, used, err = call()
		total.Add(used)
		if err == nil || ctx.Err() != nil || errors.Is(err, adapters.ErrKeyRejected) || errors.Is(err, adapters.ErrNoCredit) {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	return out, total, err
}

// CoverCheckSpec is a picture brought in as a game's cover, and the
// youth-protection rule it must keep to.
type CoverCheckSpec struct {
	Image []byte
	// Guardrail is the youth-protection rule, already resolved.
	Guardrail  string
	MockPaceMs int
}

// CoverCheck is the verdict on a picture: whether it keeps to the rule, why,
// and what asking cost.
type CoverCheck struct {
	Complies bool
	Reason   string
	Usage    UsageReport
}

// coverNodeCheck is the name a cover check is billed under.
const coverNodeCheck = "cover-check"

// CheckCover has the offer's tooling model look at a picture and say whether
// it keeps to the rule, so a picture that was not drawn under the rule may
// still become a cover. A platform whose tooling cannot look at pictures is
// refused with ProblemNoKey. It blocks until the verdict is in.
func CheckCover(ctx context.Context, spec CoverCheckSpec, offer Offer) (CoverCheck, error) {
	for _, issue := range offerIssues("offer", offer) {
		if issue.Severity == SeverityError {
			return CoverCheck{}, fmt.Errorf("%s", issue)
		}
	}
	tiers := offer.Caveats.tiers()
	if len(tiers) == 0 {
		return CoverCheck{}, Refuse(ProblemNoKey, fmt.Sprintf("offer %q has no tier", offer.ID))
	}
	toggles, _ := offer.Caveats.allowed(tiers[0])
	kit, err := SessionSpec{MockPaceMs: spec.MockPaceMs}.adapters(settlement{offer: offer, Choice: Choice{Tier: tiers[0], Toggles: toggles}})
	if err != nil {
		return CoverCheck{}, err
	}
	defer kit.close()
	looker, ok := kit.tool.(adapters.ImageTool)
	if !ok {
		return CoverCheck{}, Refuse(ProblemNoKey, fmt.Sprintf("platform %q cannot look at pictures", offer.Platform))
	}

	ledger := newUsageLedger()
	ledger.price(offer.Platform)
	reply, used, err := withRetry(ctx, func() (string, adapters.Usage, error) {
		return looker.QueryImage(ctx, genre.CoverCheckPrompt(), genre.CoverCheckMessage(spec.Guardrail), spec.Image, genre.CoverCheckSchema())
	})
	ledger.record(ports.Usage{
		Node: coverNodeCheck, Model: used.Model,
		InputTokens: used.InputTokens, CachedInputTokens: used.CachedInputTokens, OutputTokens: used.OutputTokens,
	})
	if err != nil {
		return CoverCheck{Usage: ledger.report()}, coverFailure("checking the picture", err)
	}
	var verdict struct {
		Complies bool   `json:"complies"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(reply), &verdict); err != nil {
		return CoverCheck{Usage: ledger.report()}, coverFailure("checking the picture", fmt.Errorf("%w: %v", adapters.ErrBadReply, err))
	}
	return CoverCheck{Complies: verdict.Complies, Reason: verdict.Reason, Usage: ledger.report()}, nil
}
