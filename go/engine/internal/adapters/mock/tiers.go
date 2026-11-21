package mock

import "engine/internal/adapters"

// Platform is the mock's catalogue entry: every feature on, the same canned
// models at every tier, and no key to supply.
func Platform() adapters.Platform {
	models := func() adapters.ModelSet {
		return adapters.ModelSet{
			adapters.ModelTooling: {Name: "mock-tool"},
			adapters.ModelStory:   {Name: "mock-story"},
			adapters.ModelImage:   {Name: "mock-image"},
			adapters.ModelLive:    {Name: "gpt-live-mock"},
			// The story model, speaking: the story thread runs on it while
			// narration is on.
			adapters.ModelSpeech:  {Name: "mock-realtime", Params: map[string]string{adapters.ParamVoice: "marin"}},
			adapters.ModelHearing: {Name: "mock-hearing"},
		}
	}
	return adapters.Platform{
		ID:      "mock",
		Name:    "Mock",
		Toggles: adapters.Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true},
		Tiers:   adapters.Ladder(models(), models(), models()),
		Facts:   Models,
	}
}
