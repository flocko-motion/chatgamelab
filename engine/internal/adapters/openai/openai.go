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

const (
	liveSessionsURL = "https://api.openai.com/v1/live/sessions"
	responsesURL    = "https://api.openai.com/v1/responses"
	imagesURL       = "https://api.openai.com/v1/images/generations"
)
