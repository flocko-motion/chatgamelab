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
// Every tier draws at low quality, fast and cheap and good enough, in
// landscape, the shape that sits well above a conversation: a workshop runs a
// room of young people playing at once, so the saving multiplies. Economy and
// Balanced draw with gpt-image-2.5-flare, High with gpt-image-2.6-flare.
//
// GPT-Live publishes no tier ladder, so every tier resolves to the same voice
// model. Hearing, the tooling model that takes the player's speech, does not
// move with the tier either: its prompt is short, as tooling's are. It is an
// audio model over plain HTTP: probed on 21 November 2026 against
// gpt-realtime-2.1-mini, each clip in a session of its own, gpt-audio-mini
// heard as faithfully, in half the time (0.8 s to 1.7 s), and without a
// WebSocket held for one shot. OpenAI retires gpt-audio-mini on 20 January
// 2027 for gpt-audio-1.5, on the same path.
//
// Speech is the story model, speaking: while narration is on, the story
// thread runs on a gpt-realtime model, which writes the prose and says it.
// The mini model serves low and medium, the full one high.
var (
	live       = adapters.Model{Name: "gpt-live-1"}
	tooling    = adapters.Model{Name: "gpt-6-luna"}
	speechMini = speaking("gpt-realtime-2.1-mini")
	speechFull = speaking("gpt-realtime-2.1")
	hearing    = adapters.Model{Name: "gpt-audio-1.5"}
	picture    = image("gpt-image-2.5-flare", "low", "1024x672")
	pictureNew = image("gpt-image-2.6-flare", "low", "1024x672")
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
		// Its models move fast; an admin replaces a retired one before a release does.
		Configurable: true,
		Toggles:      adapters.Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true},
		Facts:        Models,
		Voices:       map[adapters.ModelType][]adapters.BuiltinVoice{adapters.ModelLive: LiveVoices, adapters.ModelSpeech: RealtimeVoices},
		Tiers: adapters.Ladder(
			adapters.ModelSet{
				adapters.ModelTooling: tooling,
				adapters.ModelStory:   {Name: "gpt-6-luna"},
				adapters.ModelImage:   picture,
				adapters.ModelLive:    live,
				adapters.ModelSpeech:  speechMini,
				adapters.ModelHearing: hearing,
			},
			adapters.ModelSet{
				adapters.ModelTooling: tooling,
				adapters.ModelStory:   {Name: "gpt-6.1-terra"},
				adapters.ModelImage:   picture,
				adapters.ModelLive:    live,
				adapters.ModelSpeech:  speechMini,
				adapters.ModelHearing: hearing,
			},
			adapters.ModelSet{
				adapters.ModelTooling: tooling,
				adapters.ModelStory:   {Name: "gpt-6.2-sol"},
				adapters.ModelImage:   pictureNew,
				adapters.ModelLive:    live,
				adapters.ModelSpeech:  speechFull,
				adapters.ModelHearing: hearing,
			},
		),
	}
}
