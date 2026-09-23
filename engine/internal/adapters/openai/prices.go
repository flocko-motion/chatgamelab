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
	// The image models bill text input and image input at different rates, which
	// this table cannot express; the text rate is used, because that is what a
	// prompt costs and the engine never sends an image in.
	//
	// PerImage stays zero for all of them. The API reports the output tokens a
	// picture actually cost, so pricing it per image as well would charge for
	// the same picture twice.
	"gpt-image-2.5-flare":    {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-2.5-sunburst": {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-2":            {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00},
	"gpt-image-1.5":          {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 32.00},
	"gpt-image-1":            {InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 40.00},
	"gpt-image-1-mini":       {InputPerMTok: 2.00, CachedInputPerMTok: 0.20, OutputPerMTok: 8.00},

	// GPT-Live bills for the time a session stays open — $0.05 per minute,
	// charged per second and never rounded up — so a silent minute costs a
	// minute. Backend work, if a session ever delegates any, is billed apart
	// from this in tokens.
	"gpt-live-1": {AudioPerMinute: 0.05},
}

// Prices returns what is known about a model's cost. The second result reports
// whether anything is known at all, so a caller can say "unpriced" rather than
// showing a confident zero.
func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
