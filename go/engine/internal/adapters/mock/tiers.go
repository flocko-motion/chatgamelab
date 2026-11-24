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
		Voices:  map[adapters.ModelType][]adapters.BuiltinVoice{adapters.ModelLive: BuiltinVoices, adapters.ModelSpeech: BuiltinVoices},
	}
}

// BuiltinVoices are the mock's own voices, tagged as a provider's are: both
// genders in English and German, and one whose gender is not stated. Its
// live and speaking models speak in any of them, and sound the same in each.
var BuiltinVoices = []adapters.BuiltinVoice{
	{Name: "mock-anna", Language: "en", Accent: "mock", Gender: "feminine"},
	{Name: "mock-ben", Language: "en", Accent: "mock", Gender: "masculine"},
	{Name: "mock-alex", Language: "en", Accent: "mock"},
	{Name: "mock-greta", Language: "de", Accent: "mock", Gender: "feminine"},
	{Name: "mock-hans", Language: "de", Accent: "mock", Gender: "masculine"},
}
