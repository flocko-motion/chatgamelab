package mistral

import "engine/internal/adapters"

// Model choices per tier as of 21 November 2026, on dated or versioned ids so
// each is priced; mistral-large-4-0 is Large 4's versioned id. Hearing is
// Voxtral Small at every tier, the only Voxtral that takes audio with
// instructions, by its dated id.
var (
	small   = adapters.Model{Name: "mistral-small-2603"}
	large   = adapters.Model{Name: "mistral-large-4-0"}
	hearing = adapters.Model{Name: "voxtral-small-2507"}
)

// Platform is Mistral's catalogue entry. It offers no pictures: Mistral's
// image_generation tool draws poorly and costs much. It hears the player's
// voice.
func Platform() adapters.Platform {
	return adapters.Platform{
		ID:       "mistral",
		Name:     "Mistral",
		UserKeys: true,
		Toggles:  adapters.Toggles{AudioIn: true},
		Facts:    Models,
		Tiers: adapters.Ladder(
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   small,
				adapters.ModelHearing: hearing,
			},
			adapters.ModelSet{
				adapters.ModelTooling: small,
				adapters.ModelStory:   large,
				adapters.ModelHearing: hearing,
			},
			adapters.ModelSet{
				adapters.ModelTooling: large,
				adapters.ModelStory:   large,
				adapters.ModelHearing: hearing,
			},
		),
	}
}
