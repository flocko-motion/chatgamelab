package mistral

import "engine/internal/adapters"

// pricing is where Mistral publishes its prices.
const pricing = "https://docs.mistral.ai/inference/pricing"

// Where a model's context window is read: max_context_length on /v1/models,
// which the docs round ("256k", "1M").
const windowSource = "https://api.mistral.ai/v1/models"

// Models is Mistral's model table: what each model takes and makes, its
// price and a story model's context window, hand-maintained, keyed by the
// dated or versioned ids the tiers name, so each figure stays true. Text is
// priced per million tokens; Voxtral bills the audio it hears per minute.
var Models = adapters.ModelTable{
	"mistral-small-2603": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumImage}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.15, CachedInputPerMTok: 0.015, OutputPerMTok: 0.60},
		Read:  "2026-11-21", Source: pricing,
		Window: 262_144, WindowRead: "2026-11-22", WindowSource: windowSource,
	},
	"mistral-large-4-0": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumImage}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.70, CachedInputPerMTok: 0.07, OutputPerMTok: 2.10},
		Read:  "2026-11-21", Source: pricing,
		Window: 1_048_576, WindowRead: "2026-11-22", WindowSource: windowSource,
	},
	// The only Voxtral that takes audio with instructions; the others only
	// transcribe. A cached minute, $0.0004, is not counted apart.
	"voxtral-small-2507": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumAudio}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.10, OutputPerMTok: 0.40, AudioPerMinute: 0.004},
		Read:  "2026-11-21", Source: pricing,
	},
}
