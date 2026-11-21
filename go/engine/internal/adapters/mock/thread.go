package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"engine/internal/adapters"
)

// threadPrefix starts every continuation the mock makes; the number after it
// is the turn the thread has reached.
const threadPrefix = "mock-thread-"

// Thread answers each call with its turn number, so a resumed thread is told
// apart from a fresh one: a continuation goes on counting, a fresh thread
// seeded with a transcript counts on from its exchanges, and a fresh thread
// starts at one. The reply streams word by word, spread over the Pace.
//
// A plain turn answers "[story turn N]" and its input. A structured turn
// answers JSON of its schema's shape, read from an input that is a JSON
// object, as the structured block sends it: each text field says
// "[<field> N]" and the input's "input" field, and each field of an object,
// such as a status value, carries the input's value forward marked
// "[turn N]", so a test sees it change.
// A spoken turn follows each word with a PCM16 tone, billed as audio tokens.
type Thread struct {
	Pace Pace
	// Name is the model the thread runs on; empty is mock-story.
	Name string
}

func (t Thread) Model() string {
	if t.Name == "" {
		return "mock-story"
	}
	return t.Name
}

// A spoken word's length, and audio tokens per second as OpenAI bills them.
const (
	wordSeconds          = 0.1
	audioTokensPerSecond = 20
)

func (t Thread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	n := len(turn.Transcript) + 1
	if before, ok := strings.CutPrefix(turn.Continuation.ID, threadPrefix); ok {
		if reached, err := strconv.Atoi(before); err == nil {
			n = reached + 1
		}
	}

	text := fmt.Sprintf("[story turn %d] %s", n, turn.Input)
	if turn.Schema != nil {
		text = structured(turn.Schema.Body, turn.Input, n)
	}
	spoken := turn.Speak && turn.Schema == nil
	words := strings.SplitAfter(text, " ")
	step := time.Duration(0)
	if t.Pace > 0 {
		step = t.Pace / time.Duration(len(words))
	}
	for i, word := range words {
		wait(ctx, step)
		if err := ctx.Err(); err != nil {
			return adapters.ThreadReply{}, err
		}
		turn.Hear(word)
		if spoken {
			for _, frame := range tone(i, wordSeconds) {
				turn.HearAudio(frame)
			}
		}
	}

	seen := len(turn.Instructions) + len(turn.Input)
	used := adapters.Usage{
		Model:        t.Model(),
		InputTokens:  int64(400 + seen),
		OutputTokens: int64(len(text)),
	}
	if spoken {
		used.AudioOutputTokens = int64(float64(len(words)) * wordSeconds * audioTokensPerSecond)
	}
	return adapters.ThreadReply{
		Text:  text,
		Usage: used,
		Continuation: adapters.Continuation{
			ID:    threadPrefix + strconv.Itoa(n),
			Model: t.Model(),
			Made:  time.Now().UTC(),
		},
	}, nil
}

// marker is the turn mark a carried value ends in, replaced on each turn.
var marker = regexp.MustCompile(` \[turn \d+\]$`)

// structured builds a reply of the schema's shape for turn n.
func structured(schema map[string]any, input string, n int) string {
	var in map[string]any
	_ = json.Unmarshal([]byte(input), &in)
	said, _ := in["input"].(string)

	reply := map[string]any{}
	for name, property := range properties(schema) {
		switch property["type"] {
		case "string":
			reply[name] = fmt.Sprintf("[%s %d] %s", name, n, said)
		case "object":
			before, _ := in[name].(map[string]any)
			values := map[string]string{}
			for field := range properties(property) {
				was, _ := before[field].(string)
				values[field] = strings.TrimSpace(marker.ReplaceAllString(was, "") + fmt.Sprintf(" [turn %d]", n))
			}
			reply[name] = values
		}
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(reply)
	return strings.TrimSuffix(out.String(), "\n")
}

// properties reads a schema object's properties.
func properties(schema map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	props, _ := schema["properties"].(map[string]any)
	for name, p := range props {
		if property, ok := p.(map[string]any); ok {
			out[name] = property
		}
	}
	return out
}
