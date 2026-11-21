// Package mistral implements the adapter roles against Mistral.
//
// Every role runs through the Conversations API, and every role is text: the
// tooling queries and the story thread.
//
// Mistral offers no picture, voice or transcription adapter, so a game on a
// Mistral key plays as text, and a genre that needs more cannot run on this
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
