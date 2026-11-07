// Package mistral implements the adapter roles against Mistral.
//
// Every role runs through the Conversations API: text as a one-shot
// conversation that is not stored, pictures as a one-shot conversation with
// the built-in image_generation tool, fetched afterwards from the Files API.
//
// Mistral has no full-duplex voice model, so there is no Live adapter and a
// genre that needs one cannot run on this platform.
package mistral

const baseURL = "https://api.mistral.ai/v1"
