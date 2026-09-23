package openai

import "engine/internal/adapters"

// Prices are hardcoded because OpenAI does not publish them through the API.
// They are therefore a rough guide with a date on it rather than a bill: the
// figures below are dollars per million tokens as published on 11 August 2026,
// the same table TECHNOLOGY.md records.
//
// The point is to make cost visible while playing, not to be an invoice.
var prices = map[string]adapters.Price{
	"gpt-5.1":       {InputPerMTok: 1.25, CachedInputPerMTok: 0.125, OutputPerMTok: 10.00},
	"gpt-5-mini":    {InputPerMTok: 0.25, CachedInputPerMTok: 0.025, OutputPerMTok: 2.00},
	"gpt-5.6-luna":  {InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20},
	"gpt-5.6-terra": {InputPerMTok: 2.00, CachedInputPerMTok: 0.20, OutputPerMTok: 12.00},
	"gpt-5.6-sol":   {InputPerMTok: 5.00, CachedInputPerMTok: 0.50, OutputPerMTok: 30.00},
	"gpt-image-2":   {InputPerMTok: 5.00, OutputPerMTok: 30.00, PerImage: 0.04},

	// Realtime is metered here in audio seconds rather than tokens, which is
	// the unit a conversation is easiest to reason about.
	"gpt-realtime-2.1": {AudioPerMinute: 0.30},
}

// Prices returns what is known about a model's cost. The second result reports
// whether anything is known at all, so a caller can say "unpriced" rather than
// showing a confident zero.
func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
