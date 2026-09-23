// Package openai implements the adapter roles against OpenAI.
//
// The Realtime wire format here follows the documented event names, but nothing
// in this file has been run against the live API — it needs a key and a first
// session before any of it is load-bearing. Treat the mock adapter as the
// verified path and this as the thing the spike exists to prove.
package openai

const (
	realtimeURL  = "wss://api.openai.com/v1/realtime"
	responsesURL = "https://api.openai.com/v1/responses"
	imagesURL    = "https://api.openai.com/v1/images/generations"
)
