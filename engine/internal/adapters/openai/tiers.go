package openai

import (
	"fmt"

	"engine/internal/adapters"
)

// Model choices per tier, following the table in TECHNOLOGY.md as of August
// 2026. The helper roles do not move with the tier: their prompts are short, so
// a flagship model buys nothing there.
//
// A tier presets the picture's size and quality alongside its model, because
// all three are what the session is paying for. No genre chooses them: the
// engine decides what a picture costs, and a wiring only decides that it wants
// one.
//
// Most tiers ask for landscape, which is the shape that sits well above a
// conversation. Economy cannot: custom dimensions are a 2.5 feature and it runs
// on an older model, so it takes the smallest of the three sizes that model
// offers and accepts a square. That is the one place a cheaper rung shows as a
// difference in kind rather than in quality, and it is the picture's shape
// rather than anything about the game.
//
// GPT-Live publishes no tier ladder, so every tier resolves to the same voice
// model. It is billed by the second a session stays open rather than by token,
// which makes it the one role where a cheaper tier would have to mean a shorter
// conversation rather than a smaller model.
var tierModels = map[adapters.Tier]adapters.ModelSet{
	adapters.TierMax: {
		Live:         "gpt-live-1",
		Tool:         "gpt-5.6-luna",
		Threaded:     "gpt-5.1",
		Image:        "gpt-image-2.5-flare",
		ImageQuality: "high",
		ImageSize:    "1024x672",
		Audio:        "gpt-4o-mini-tts",
	},
	adapters.TierPremium: {
		Live:         "gpt-live-1",
		Tool:         "gpt-5.6-luna",
		Threaded:     "gpt-5.1",
		Image:        "gpt-image-2.5-flare",
		ImageQuality: "high",
		ImageSize:    "1024x672",
		Audio:        "gpt-4o-mini-tts",
	},
	adapters.TierBalanced: {
		Live:         "gpt-live-1",
		Tool:         "gpt-5.6-luna",
		Threaded:     "gpt-5-mini",
		Image:        "gpt-image-2.5-flare",
		ImageQuality: "medium",
		ImageSize:    "1024x672",
		// Speech output is max/premium only.
		Audio: "",
	},
	adapters.TierEconomy: {
		Live:     "gpt-live-1",
		Tool:     "gpt-5.6-luna",
		Threaded: "gpt-5.6-luna",
		// The cheapest picture on offer, at a quarter the output price of the
		// 2.5 models. Economy is what a workshop runs on — a room of young
		// people playing at once, on a school's budget — so the saving is
		// multiplied by the size of the room.
		Image:        "gpt-image-1-mini",
		ImageQuality: "low",
		// The smallest this model offers: a megapixel square, against the one
		// and a half megapixels of either rectangle it could make instead.
		// Custom dimensions are a 2.5 feature, so the landscape the other tiers
		// ask for is not available here.
		ImageSize: "1024x1024",
		Audio:     "",
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
