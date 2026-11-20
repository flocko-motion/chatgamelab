package mock

import "engine/internal/adapters"

// Platform is the mock's catalogue entry: every feature on, the same canned
// models at every tier, and no key to supply.
func Platform() adapters.Platform {
	models := func() adapters.ModelSet {
		return adapters.ModelSet{
			adapters.ModelTooling:       {Name: "mock-tool"},
			adapters.ModelStory:         {Name: "mock-story"},
			adapters.ModelImage:         {Name: "mock-image"},
			adapters.ModelLive:          {Name: "gpt-live-mock"},
			adapters.ModelSpeech:        {Name: "mock-speech"},
			adapters.ModelTranscription: {Name: "mock-transcription"},
		}
	}
	return adapters.Platform{
		ID:      "mock",
		Name:    "Mock",
		Toggles: adapters.Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true},
		Tiers:   adapters.Ladder(models(), models(), models()),
	}
}
