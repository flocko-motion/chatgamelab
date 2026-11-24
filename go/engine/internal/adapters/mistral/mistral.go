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

// BaseURL is where every Mistral call goes; a variable so a test can point it
// at a fake.
var BaseURL = "https://api.mistral.ai/v1"

// Check asks Mistral whether the key works and the account behind it can pay,
// with the smallest call that is billed: adapters.ProbePrompt on the lowest
// tier's tooling model. It answers nil, or an error wrapping ErrKeyRejected,
// ErrNoCredit or ErrTransient.
func Check(ctx context.Context, keys adapters.KeyFunc) error {
	models, err := Platform().Models(adapters.TierLow)
	if err != nil {
		return err
	}
	_, _, err = NewTool(keys, models[adapters.ModelTooling].Name).Query(ctx, "", adapters.ProbePrompt)
	return err
}
