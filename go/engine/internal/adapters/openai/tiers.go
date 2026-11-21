package openai

import "engine/internal/adapters"

// Model choices per tier.
//
// Tooling does not move with the tier: its prompts are short, so a flagship
// model buys nothing there.
//
// A tier presets the picture's size and quality alongside its model, because
// all three are what the session is paying for. No genre chooses them: the
// engine decides what a picture costs, and a wiring only decides that it wants
// one.
//
// Medium and high ask for landscape, the shape that sits well above a
// conversation. Low runs on an older model without custom dimensions, so it
// takes that model's smallest size, a square.
//
// GPT-Live publishes no tier ladder, so every tier resolves to the same voice
// model. Transcription has one model, so it does not move with the tier
// either.
//
// Speech is the story model, speaking: while narration is on, the story
// thread runs on a gpt-realtime model, which writes the prose and says it.
// The mini model serves low and medium, the full one high.
var (
	live          = adapters.Model{Name: "gpt-live-1"}
	tooling       = adapters.Model{Name: "gpt-5.6-luna"}
	speechMini    = speaking("gpt-realtime-2.1-mini")
	speechFull    = speaking("gpt-realtime-2.1")
	transcription = adapters.Model{Name: "gpt-4o-mini-transcribe"}
)

// speaking is a gpt-realtime model with the voice it speaks in unless a game
// names another.
func speaking(name string) adapters.Model {
	return adapters.Model{Name: name, Params: map[string]string{adapters.ParamVoice: DefaultVoice}}
}

func image(name, quality, size string) adapters.Model {
	return adapters.Model{Name: name, Params: map[string]string{adapters.ParamQuality: quality, adapters.ParamSize: size}}
}

// Platform is OpenAI's catalogue entry.
func Platform() adapters.Platform {
	return adapters.Platform{
		ID:       "openai",
		Name:     "OpenAI",
		UserKeys: true,
		Toggles:  adapters.Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true},
		Tiers: adapters.Ladder(
			adapters.ModelSet{
				adapters.ModelTooling: tooling,
				adapters.ModelStory:   {Name: "gpt-5.6-luna"},
				// The cheapest picture on offer: a workshop runs on low, a room
				// of young people playing at once, so the saving multiplies.
				adapters.ModelImage:         image("gpt-image-1-mini", "low", "1024x1024"),
				adapters.ModelLive:          live,
				adapters.ModelSpeech:        speechMini,
				adapters.ModelTranscription: transcription,
			},
			adapters.ModelSet{
				adapters.ModelTooling:       tooling,
				adapters.ModelStory:         {Name: "gpt-5-mini"},
				adapters.ModelImage:         image("gpt-image-2.5-flare", "medium", "1024x672"),
				adapters.ModelLive:          live,
				adapters.ModelSpeech:        speechMini,
				adapters.ModelTranscription: transcription,
			},
			adapters.ModelSet{
				adapters.ModelTooling:       tooling,
				adapters.ModelStory:         {Name: "gpt-5.1"},
				adapters.ModelImage:         image("gpt-image-2.5-flare", "high", "1024x672"),
				adapters.ModelLive:          live,
				adapters.ModelSpeech:        speechFull,
				adapters.ModelTranscription: transcription,
			},
		),
	}
}
