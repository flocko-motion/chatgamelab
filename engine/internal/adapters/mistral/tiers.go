package mistral

import "engine/internal/adapters"

// Model choices per tier follow the table in TECHNOLOGY.md as of August 2026.
// Tooling stays on the small model at every tier, as its prompts are short.
//
// The image model is the conversation model that calls the image_generation
// tool; the tool takes no size or quality, so neither is set.
//
// Toggles list what this adapter implements: text and pictures.
var (
	small = adapters.Model{Name: "mistral-small-latest"}
)

// Platform is Mistral's catalogue entry.
func Platform() adapters.Platform {
	return adapters.Platform{
		ID:       "mistral",
		Name:     "Mistral",
		UserKeys: true,
		Toggles:  adapters.Toggles{Images: true},
		Tiers: adapters.Ladder(
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   small,
				adapters.ModelImage:   small,
			},
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   {Name: "mistral-medium-latest"},
				adapters.ModelImage:   small,
			},
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   {Name: "mistral-large-latest"},
				adapters.ModelImage:   small,
			},
		),
	}
}
