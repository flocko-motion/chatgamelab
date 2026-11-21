package mock

import "engine/internal/adapters"

// Models is the mock's model table. Its prices are invented, so a keyless run
// exercises the cost display.
var Models = adapters.ModelTable{
	"mock-tool":  invented(text, text, adapters.Price{InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20}),
	"mock-story": invented(text, text, adapters.Price{InputPerMTok: 1.50, CachedInputPerMTok: 0.15, OutputPerMTok: 6.00}),
	"mock-image": invented(text, []adapters.Medium{adapters.MediumImage}, adapters.Price{InputPerMTok: 5.00, PerImage: 0.04}),
	"mock-realtime": invented(textAudio, textAudio, adapters.Price{
		InputPerMTok: 0.60, CachedInputPerMTok: 0.06, OutputPerMTok: 2.40,
		AudioInputPerMTok: 10.00, CachedAudioInputPerMTok: 0.30, AudioOutputPerMTok: 20.00,
	}),
	"mock-hearing":  invented(textAudio, text, adapters.Price{InputPerMTok: 0.60, OutputPerMTok: 2.40, AudioInputPerMTok: 10.00}),
	"gpt-live-mock": invented(textAudio, textAudio, adapters.Price{AudioPerMinute: 0.05}),
}

var (
	text      = []adapters.Medium{adapters.MediumText}
	textAudio = []adapters.Medium{adapters.MediumText, adapters.MediumAudio}
)

// invented is an entry whose price no provider published.
func invented(in, out []adapters.Medium, price adapters.Price) adapters.ModelFacts {
	return adapters.ModelFacts{In: in, Out: out, Price: price, Read: "2026-11-21", Source: "invented for the mock"}
}
