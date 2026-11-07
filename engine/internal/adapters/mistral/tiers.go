package mistral

import (
	"fmt"

	"engine/internal/adapters"
)

// Model choices per tier, following the table in TECHNOLOGY.md as of August
// 2026. The helper roles stay on the small model at every tier, as their
// prompts are short.
//
// The image model is the conversation model that calls the image_generation
// tool; the tool takes no size or quality, so neither is set.
//
// Live and Audio stay empty: Mistral offers no live voice or speech model.
var tierModels = map[adapters.Tier]adapters.ModelSet{
	adapters.TierMax: {
		Tool:     "mistral-small-latest",
		Threaded: "mistral-large-latest",
		Image:    "mistral-small-latest",
	},
	adapters.TierPremium: {
		Tool:     "mistral-small-latest",
		Threaded: "mistral-large-latest",
		Image:    "mistral-small-latest",
	},
	adapters.TierBalanced: {
		Tool:     "mistral-small-latest",
		Threaded: "mistral-medium-latest",
		Image:    "mistral-small-latest",
	},
	adapters.TierEconomy: {
		Tool:     "mistral-small-latest",
		Threaded: "mistral-small-latest",
		Image:    "mistral-small-latest",
	},
}

// Models resolves a tier into concrete model names.
func Models(tier adapters.Tier) (adapters.ModelSet, error) {
	set, ok := tierModels[tier]
	if !ok {
		return adapters.ModelSet{}, fmt.Errorf("mistral: unknown tier %q", tier)
	}
	return set, nil
}
