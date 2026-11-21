package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"

	"engine/internal/adapters"
)

// Hearing hears a clip by its length: every text field of the schema says how
// long the speech was, so a test can tell what reached the turn. A silent clip
// is heard as nothing, every field empty, as a real model hears a button held
// with nobody speaking.
type Hearing struct{ Pace Pace }

func (h Hearing) Hear(ctx context.Context, system string, clip adapters.Clip, schema adapters.Schema) (string, adapters.Usage, error) {
	wait(ctx, h.Pace)
	words := ""
	if !clip.Silent() {
		words = fmt.Sprintf("[mock-hearing] %.1f s of speech", clip.Seconds())
	}
	reply := map[string]string{}
	properties, _ := schema.Body["properties"].(map[string]any)
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		if property, _ := properties[name].(map[string]any); property["type"] == "string" {
			reply[name] = words
		}
	}
	blob, err := json.Marshal(reply)
	used := adapters.Usage{
		Model:       "mock-hearing",
		InputTokens: int64(len(system)),
		// Ten tokens a second, roughly what an audio model counts.
		AudioInputTokens: int64(math.Ceil(clip.Seconds() * 10)),
		OutputTokens:     int64(len(blob)),
	}
	return string(blob), used, err
}

// SpokenClip is a clip of speech lasting seconds, for a test or a headless run
// to speak: a tone, which the mock hears by its length.
func SpokenClip(seconds float64) adapters.Clip {
	var clip adapters.Clip
	for _, frame := range tone(0, seconds) {
		clip = append(clip, frame...)
	}
	return clip
}

// SilentClip is a clip of silence lasting seconds.
func SilentClip(seconds float64) adapters.Clip {
	return make(adapters.Clip, int(seconds*adapters.ClipRate)*2)
}
