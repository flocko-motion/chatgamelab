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
// is the turn the thread has reached, and a number after a colon the spoken
// replies the provider thread holds as audio.
const threadPrefix = "mock-thread-"

// Thread answers each call with its turn number, which a continuation or a
// seeding transcript carries on, streamed word by word over the Pace.
//
// A plain turn answers "[story turn N]" and its input's first words. A structured turn
// answers JSON of its schema: each text field "[<field> N]" and the action,
// each object field its value carried on, marked "[turn N]". A cue that is no
// JSON stands in for the action. A spoken turn follows each word with a PCM16
// tone, billed as audio tokens.
//
// Its input grows with the thread, as a provider's does: it bills the System
// Message, the summary and the call's own text, and each turn the thread
// holds bills turnTokens of input, and each spoken reply a continued thread
// holds bills spokenTokens more, as audio input, as a Realtime session keeps
// a spoken reply as audio. A fresh thread is seeded from text, so it holds no
// audio; a summary it is seeded with counts as a turn.
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

// What a held turn takes of the input: its text, and a spoken reply's audio
// on top, which takes more than its transcript. Text sent is billed at about
// four characters a token, as English runs.
const (
	turnTokens    = 200
	spokenTokens  = 300
	charsPerToken = 4
)

func (t Thread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	held, heldSpoken, continued := reached(turn.Continuation.ID)
	if !continued {
		held = len(turn.Transcript)
		if turn.Summary != "" {
			held++
		}
	}
	n := held + 1

	text := fmt.Sprintf("[story turn %d] %s", n, echo(turn.Input))
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

	// The thread holds the System Message and the summary throughout.
	sent := len(turn.System) + len(turn.Summary) + len(turn.Input) + len(turn.Constraints)
	used := adapters.Usage{
		Model:            t.Model(),
		InputTokens:      int64(400 + sent/charsPerToken + turnTokens*held),
		AudioInputTokens: int64(spokenTokens * heldSpoken),
		OutputTokens:     int64(len(text)),
	}
	if spoken {
		used.AudioOutputTokens = int64(float64(len(words)) * wordSeconds * audioTokensPerSecond)
		heldSpoken++
	}
	id := threadPrefix + strconv.Itoa(n)
	if heldSpoken > 0 {
		id += ":" + strconv.Itoa(heldSpoken)
	}
	return adapters.ThreadReply{
		Text:  text,
		Usage: used,
		Continuation: adapters.Continuation{
			ID:    id,
			Model: t.Model(),
			Made:  time.Now().UTC(),
		},
	}, nil
}

// reached reads a continuation the mock made: the turns its thread holds,
// the spoken replies among them, and whether it is one at all.
func reached(id string) (held, spoken int, ok bool) {
	rest, ok := strings.CutPrefix(id, threadPrefix)
	if !ok {
		return 0, 0, false
	}
	turns, audio, _ := strings.Cut(rest, ":")
	held, err := strconv.Atoi(turns)
	if err != nil {
		return 0, 0, false
	}
	spoken, _ = strconv.Atoi(audio)
	return held, spoken, true
}

// echoWords bounds a plain turn's echo, so a long prompt makes a reply, and a
// spoken one audio, of a turn's usual size.
const echoWords = 12

// echo is the input's first words.
func echo(input string) string {
	words := strings.Fields(input)
	if len(words) <= echoWords {
		return input
	}
	return strings.Join(words[:echoWords], " ")
}

// marker is the turn mark a carried value ends in, replaced on each turn.
var marker = regexp.MustCompile(`\s*\[turn \d+\]$`)

// structured builds a reply of the schema's shape for turn n.
func structured(schema map[string]any, input string, n int) string {
	var in map[string]any
	said := input
	if json.Unmarshal([]byte(input), &in) == nil {
		said, _ = in["input"].(string)
	}

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
