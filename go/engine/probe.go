package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/ports"
)

// ErrNoProbe: the engine has no probe for the model type yet.
var ErrNoProbe = errors.New("no probe for this model type yet")

// probeWait bounds a probe: an admin waits for it.
const probeWait = 60 * time.Second

// probeImagePrompt is the picture a probe asks an image model for: the
// smallest thing worth drawing.
const probeImagePrompt = "A single red dot on a white background."

// ProbeModel asks whether a model answers, with the smallest request its type
// takes, on the offer's key: a question answered in a word for a tooling or
// story model, the cheapest picture for an image model. Any model name may be
// probed, also one the catalogue does not list yet, so an admin tries a model
// before putting it into a tier. It answers what the probe cost, priced from
// the platform's model table, and nil, or the provider's refusal (as
// CheckOffer's, with ProblemFailed for anything else, such as a model the
// provider does not know), or ErrNoProbe for a model type it cannot probe yet (live,
// speech, hearing).
func ProbeModel(ctx context.Context, offer Offer, kind ModelType, model string) (UsageReport, error) {
	ledger := newUsageLedger()
	ledger.price(offer.Platform)
	if model == "" {
		return ledger.report(), fmt.Errorf("no model to probe")
	}
	if offer.Platform != PlatformMock && offer.Keys == nil {
		return ledger.report(), &Refusal{Problem: ProblemNoKey, Message: fmt.Sprintf("offer %q has no key", offer.ID)}
	}
	ctx, cancel := context.WithTimeout(ctx, probeWait)
	defer cancel()

	keys := adapters.KeyFunc(offer.Keys)
	var used adapters.Usage
	var err error
	switch kind {
	case ModelTooling, ModelStory:
		var tool adapters.Tool
		switch offer.Platform {
		case PlatformOpenAI:
			tool = openai.NewTool(keys, model)
		case PlatformMistral:
			tool = mistral.NewTool(keys, model)
		case PlatformMock:
			tool = mock.Tool{}
		default:
			return ledger.report(), fmt.Errorf("unknown platform %q", offer.Platform)
		}
		_, used, err = tool.Query(ctx, "", adapters.ProbePrompt)
	case ModelImage:
		var image adapters.Image
		switch offer.Platform {
		case PlatformOpenAI:
			image = openai.NewImage(keys, model, "low", "1024x1024")
		case PlatformMock:
			image = mock.Image{}
		default:
			return ledger.report(), Refuse(ProblemNoKey, fmt.Sprintf("platform %q draws no pictures", offer.Platform))
		}
		_, used, err = image.Generate(ctx, adapters.ImageRequest{Prompt: probeImagePrompt})
	default:
		return ledger.report(), ErrNoProbe
	}
	if used.Model == "" {
		used.Model = model
	}
	ledger.record(ports.Usage{
		Node: "probe", Model: used.Model,
		InputTokens: used.InputTokens, CachedInputTokens: used.CachedInputTokens, OutputTokens: used.OutputTokens,
		AudioInputTokens: used.AudioInputTokens, CachedAudioInputTokens: used.CachedAudioInputTokens,
		AudioOutputTokens: used.AudioOutputTokens, AudioSeconds: used.AudioSeconds, Images: used.Images,
	})
	return ledger.report(), providerRefusal(offer.Platform, err, ProblemFailed)
}
