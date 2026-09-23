package openai

import (
	"fmt"

	"engine/internal/adapters"
)

// Model choices per tier, following the table in TECHNOLOGY.md as of August
// 2026. The helper roles do not move with the tier: their prompts are short, so
// a flagship model buys nothing there.
//
// The live model has no documented tier ladder yet — realtime is new — so every
// tier resolves to the same one until the spike says otherwise.
var tierModels = map[adapters.Tier]adapters.ModelSet{
	adapters.TierMax: {
		Live:     "gpt-realtime-2.1",
		Tool:     "gpt-5.6-luna",
		Threaded: "gpt-5.1",
		Image:    "gpt-image-2",
		Audio:    "gpt-4o-mini-tts",
	},
	adapters.TierPremium: {
		Live:     "gpt-realtime-2.1",
		Tool:     "gpt-5.6-luna",
		Threaded: "gpt-5.1",
		Image:    "gpt-image-2",
		Audio:    "gpt-4o-mini-tts",
	},
	adapters.TierBalanced: {
		Live:     "gpt-realtime-2.1",
		Tool:     "gpt-5.6-luna",
		Threaded: "gpt-5-mini",
		Image:    "gpt-image-2",
		// Speech output is max/premium only.
		Audio: "",
	},
	adapters.TierEconomy: {
		Live:     "gpt-realtime-2.1",
		Tool:     "gpt-5.6-luna",
		Threaded: "gpt-5.6-luna",
		// Economy generates no images: every preview frame costs image output
		// tokens, which is what makes the tier cheap.
		Image: "",
		Audio: "",
	},
}

// Models resolves a tier into concrete model names.
func Models(tier adapters.Tier) (adapters.ModelSet, error) {
	set, ok := tierModels[tier]
	if !ok {
		return adapters.ModelSet{}, fmt.Errorf("openai: unknown tier %q", tier)
	}
	return set, nil
}
