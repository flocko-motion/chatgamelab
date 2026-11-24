// Package openai implements the adapter roles against OpenAI.
//
// Two model families are in play and they are not interchangeable. The live
// conversation runs on GPT-Live, a full-duplex voice model addressed through
// its own /v1/live endpoints; everything else runs on ordinary GPT models
// through the Responses API.
//
// Written against the published documentation. Nothing here has been run
// against the live API — it needs a key and a first session before any of it is
// load-bearing. The mock adapter is the verified path; this is what the spike
// exists to prove.
package openai

import (
	"context"

	"engine/internal/adapters"
)

const (
	liveSessionsURL = "https://api.openai.com/v1/live/sessions"
	imagesURL       = "https://api.openai.com/v1/images/generations"
)

// ResponsesURL is where every GPT call but the live ones goes; a variable so a
// test can point it at a fake.
var ResponsesURL = "https://api.openai.com/v1/responses"

// Check asks OpenAI whether the key works and the account behind it can pay,
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
