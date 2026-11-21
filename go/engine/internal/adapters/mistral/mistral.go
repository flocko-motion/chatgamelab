// Package mistral implements the adapter roles against Mistral.
//
// The tooling queries and the story thread run through the Conversations API,
// as text; hearing the player's voice runs on Voxtral through Chat
// Completions, the one API that takes audio with instructions.
//
// Mistral offers no picture, narration or live voice, so a game on a Mistral
// key is told in text, and a genre that needs more cannot run on this
// platform.
package mistral

import (
	"context"

	"engine/internal/adapters"
)

const baseURL = "https://api.mistral.ai/v1"

// ModelsURL is what a key check asks; a variable so a test can point it at a fake.
var ModelsURL = baseURL + "/models"

// Check asks Mistral whether it accepts the key, at no cost.
func Check(ctx context.Context, keys adapters.KeyFunc) error {
	return adapters.CheckKey(ctx, keys, ModelsURL)
}
