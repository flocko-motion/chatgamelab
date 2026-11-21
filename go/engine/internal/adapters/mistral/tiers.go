package mistral

import "engine/internal/adapters"

// Model choices per tier as of 21 November 2026, on dated or versioned ids so
// each is priced; mistral-large-4-0 is Large 4's versioned id. The image model
// is the conversation model calling the image_generation tool, which takes no
// size or quality.
var (
	small = adapters.Model{Name: "mistral-small-2603"}
	large = adapters.Model{Name: "mistral-large-4-0"}
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
				adapters.ModelStory:   large,
				adapters.ModelImage:   small,
			},
			adapters.ModelSet{
				adapters.ModelTooling: large,
				adapters.ModelStory:   large,
				adapters.ModelImage:   small,
			},
		),
	}
}
