package mistral

import "engine/internal/adapters"

// pricing is where Mistral publishes its prices.
const pricing = "https://docs.mistral.ai/inference/pricing"

// Models is Mistral's model table: what each model takes and makes, and its
// price, hand-maintained, keyed by the dated or versioned ids the tiers name,
// so each figure stays true. Text is priced per million tokens; Voxtral bills
// the audio it hears per minute.
var Models = adapters.ModelTable{
	"mistral-small-2603": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumImage}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.15, CachedInputPerMTok: 0.015, OutputPerMTok: 0.60},
		Read:  "2026-11-21", Source: pricing,
	},
	"mistral-large-4-0": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumImage}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.70, CachedInputPerMTok: 0.07, OutputPerMTok: 2.10},
		Read:  "2026-11-21", Source: pricing,
	},
	// The only Voxtral that takes audio with instructions; the others only
	// transcribe. A cached minute, $0.0004, is not counted apart.
	"voxtral-small-2507": {
		In: []adapters.Medium{adapters.MediumText, adapters.MediumAudio}, Out: []adapters.Medium{adapters.MediumText},
		Price: adapters.Price{InputPerMTok: 0.10, OutputPerMTok: 0.40, AudioPerMinute: 0.004},
		Read:  "2026-11-21", Source: pricing,
	},
}
