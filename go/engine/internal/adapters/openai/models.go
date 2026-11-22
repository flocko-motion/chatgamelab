package openai

import "engine/internal/adapters"

// pricing is where OpenAI publishes its prices.
const pricing = "https://developers.openai.com/api/docs/pricing"

// The media the model families take and make.
var (
	text       = []adapters.Medium{adapters.MediumText}
	textImage  = []adapters.Medium{adapters.MediumText, adapters.MediumImage}
	textAudio  = []adapters.Medium{adapters.MediumText, adapters.MediumAudio}
	imageOut   = []adapters.Medium{adapters.MediumImage}
	realtimeIn = []adapters.Medium{adapters.MediumText, adapters.MediumAudio, adapters.MediumImage}
)

// facts is a model's entry, read from the pricing page on a date.
func facts(in, out []adapters.Medium, read string, price adapters.Price) adapters.ModelFacts {
	return adapters.ModelFacts{In: in, Out: out, Price: price, Read: read, Source: pricing}
}

// Models is OpenAI's model table: what each model takes and makes, and its
// price in dollars per million tokens, hand-maintained.
//
// Image models are priced at the text rate, as the engine sends no image in;
// PerImage stays zero because the reported output tokens already price the
// picture. The speaking and hearing models bill text and audio tokens apart.
// GPT-Live bills per second a session stays open, so a silent minute costs a
// minute.
var Models = adapters.ModelTable{
	"gpt-6.2-sol":   facts(textImage, text, "2026-11-22", adapters.Price{InputPerMTok: 2.00, CachedInputPerMTok: 0.10, OutputPerMTok: 10.00}),
	"gpt-6.1-terra": facts(textImage, text, "2026-11-22", adapters.Price{InputPerMTok: 1.00, CachedInputPerMTok: 0.05, OutputPerMTok: 5.00}),
	"gpt-5.6-luna":  facts(textImage, text, "2026-08-11", adapters.Price{InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20}),

	"gpt-image-2.5-flare": facts(textImage, imageOut, "2026-08-11", adapters.Price{InputPerMTok: 5.00, CachedInputPerMTok: 1.25, OutputPerMTok: 30.00}),

	"gpt-realtime-2.1": facts(realtimeIn, textAudio, "2026-11-21", adapters.Price{
		InputPerMTok: 4.00, CachedInputPerMTok: 0.40, OutputPerMTok: 24.00,
		AudioInputPerMTok: 32.00, CachedAudioInputPerMTok: 0.40, AudioOutputPerMTok: 64.00,
	}),
	"gpt-realtime-2.1-mini": facts(realtimeIn, textAudio, "2026-11-21", adapters.Price{
		InputPerMTok: 0.60, CachedInputPerMTok: 0.06, OutputPerMTok: 2.40,
		AudioInputPerMTok: 10.00, CachedAudioInputPerMTok: 0.30, AudioOutputPerMTok: 20.00,
	}),
	// No cached rate is listed.
	"gpt-audio-1.5": facts(textAudio, textAudio, "2026-11-22", adapters.Price{
		InputPerMTok: 2.50, OutputPerMTok: 10.00, AudioInputPerMTok: 32.00,
	}),

	"gpt-live-1": facts(textAudio, textAudio, "2026-08-11", adapters.Price{AudioPerMinute: 0.05}),
}
