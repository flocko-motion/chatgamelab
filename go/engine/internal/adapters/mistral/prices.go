package mistral

import "engine/internal/adapters"

// Prices are hand-maintained: dollars per million tokens from
// https://docs.mistral.ai/inference/pricing as of 21 November 2026, keyed by
// the dated or versioned ids the tiers name, so each figure stays true.
var prices = map[string]adapters.Price{
	"mistral-small-2603": {InputPerMTok: 0.15, CachedInputPerMTok: 0.015, OutputPerMTok: 0.60},
	"mistral-large-4-0":  {InputPerMTok: 0.70, CachedInputPerMTok: 0.07, OutputPerMTok: 2.10},
}

// Prices reports whether a model is priced, so a caller can say "unpriced" over a zero.
func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
