package mistral

import "engine/internal/adapters"

// Model choices per tier as of 21 November 2026, on dated or versioned ids so
// each is priced; mistral-large-4-0 is Large 4's versioned id.
var (
	small = adapters.Model{Name: "mistral-small-2603"}
	large = adapters.Model{Name: "mistral-large-4-0"}
)

// Platform is Mistral's catalogue entry. It offers no pictures: Mistral's
// image_generation tool draws poorly and costs much.
func Platform() adapters.Platform {
	return adapters.Platform{
		ID:       "mistral",
		Name:     "Mistral",
		UserKeys: true,
		Toggles:  adapters.Toggles{},
		Tiers: adapters.Ladder(
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   small,
			},
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   large,
			},
			adapters.ModelSet{
				adapters.ModelTooling: large,
				adapters.ModelStory:   large,
			},
		),
	}
}
