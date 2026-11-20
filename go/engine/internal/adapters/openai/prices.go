package openai

import "engine/internal/adapters"

// Prices are hand-maintained, as OpenAI publishes none through its API: dollars per
// million tokens as published on 11 August 2026.
var prices = map[string]adapters.Price{
	"gpt-5.1":       {InputPerMTok: 1.25, CachedInputPerMTok: 0.125, OutputPerMTok: 10.00},
	"gpt-5-mini":    {InputPerMTok: 0.25, CachedInputPerMTok: 0.025, OutputPerMTok: 2.00},
	"gpt-5.6-luna":  {InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20},
	"gpt-5.6-terra": {InputPerMTok: 2.00, CachedInputPerMTok: 0.20, OutputPerMTok: 12.00},
	"gpt-5.6-sol":   {InputPerMTok: 5.00, CachedInputPerMTok: 0.50, OutputPerMTok: 30.00},
	// Image models use the text rate, as the engine sends no image in; PerImage stays
	// zero because the reported output tokens already price the picture.
	"gpt-image-2.5-flare":    {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-2.5-sunburst": {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-2":            {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-1.5":          {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 32.00},
	"gpt-image-1":            {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 40.00},
	"gpt-image-1-mini":       {InputPerMTok: 2.00, CachedInputPerMTok: 0.20, OutputPerMTok: 8.00},

	// GPT-Live bills per second a session stays open, so a silent minute costs a minute.
	"gpt-live-1": {AudioPerMinute: 0.05},
}

// Prices reports whether a model is priced, so a caller can say "unpriced" over a zero.
func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
