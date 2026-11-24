package blocks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// errNothingHeard is a clip in which the model heard no words.
var errNothingHeard = errors.New("nothing was heard in the clip")

// HearingCall is a tool call that takes audio: each clip of the player's
// speech, with the system prompt, goes to a model that hears, and its reply,
// JSON of the schema, goes out as its Result names the fields: the text, made
// from what was heard, and the second text, what was heard.
//
// Parsing is part of the call: a reply is tidied of a fence and the text
// around its JSON, and one that still does not parse, or has no text, is a
// bad reply, made again up to Attempts times. A reply that heard nothing, its
// second text empty, starts nothing: it fails the call, as does a silent clip,
// which is never sent. A call that fails for good sends an empty input on the
// failure output, as the clip is not kept.
//
// Its switch says whether the player may speak: the wiring refuses a clip
// while it is off. A clip that reached the block is heard all the same, as a
// switch affects only what is made from then on.
type HearingCall struct {
	Switch
	failing
	name    string
	system  tunable[string]
	hearing adapters.Hearing
	schema  adapters.Schema
	result  Result

	// Backoff is the wait before each retry of a failure that may pass. Set
	// before Start.
	Backoff []time.Duration
	// Attempts is how many bad replies a call gets before it fails. Set
	// before Start.
	Attempts int

	in        ports.AudioInput
	out       ports.TextBroadcast
	secondary ports.TextBroadcast
	state     ports.StateBroadcast
	meter     meter
}

// NewHearingCall builds the block; result names the reply's fields for its
// text and its second text.
func NewHearingCall(name string, hearing adapters.Hearing, system string, schema adapters.Schema, result Result) *HearingCall {
	if hearing == nil {
		panic("blocks: " + name + " needs a hearing adapter")
	}
	return &HearingCall{
		name:     name,
		system:   newTunable(system),
		hearing:  hearing,
		schema:   schema,
		result:   result,
		Backoff:  []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
		Attempts: 3,
		in:       make(ports.AudioInput, 4),
		meter:    meter{name: name},
	}
}

func (b *HearingCall) NodeName() string { return b.name }

// SetSystem changes its system message from the next clip on.
func (b *HearingCall) SetSystem(system string) { b.system.set(system) }

func (b *HearingCall) AudioInPort() chan<- ports.AudioChunk { return b.in }
func (b *HearingCall) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindAudio} }

// TextOutPort is the reply's text field.
func (b *HearingCall) TextOutPort() <-chan string { return b.out.Subscribe() }

// SecondaryTextOutPort is the reply's second text field.
func (b *HearingCall) SecondaryTextOutPort() <-chan string { return b.secondary.Subscribe() }

// UsageOutPort reports what the calls have cost, bad replies included.
func (b *HearingCall) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

// StateOutPort reports what this block is doing, with the provider's reason
// after a failed call.
func (b *HearingCall) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *HearingCall) Start(ctx context.Context) {
	go func() {
		defer func() {
			b.out.Close()
			b.secondary.Close()
			b.failure.Close()
			b.state.Close()
			b.meter.usage.Close()
		}()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		for {
			select {
			case <-ctx.Done():
				return
			case clip, open := <-b.in:
				if !open {
					return
				}
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				text, secondary, err := b.hear(ctx, adapters.Clip(clip))
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					b.fail(&b.state, b.name, "", err)
					continue
				}
				// What was heard first, so the player's line shows before the
				// turn it starts.
				b.secondary.Send(secondary)
				b.out.Send(text)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// hear calls until a reply parses or the call fails for good. A clip with no
// sound in it is heard as nothing without a call: a model asked to hear
// silence may make words up.
func (b *HearingCall) hear(ctx context.Context, clip adapters.Clip) (text, secondary string, err error) {
	if clip.Silent() {
		return "", "", errNothingHeard
	}
	bad, retries := 0, 0
	for {
		reply, used, err := b.hearing.Hear(ctx, b.system.get(), clip, b.schema)
		// A bad reply is billed all the same.
		if used != (adapters.Usage{}) {
			b.meter.bill(used)
		}
		if err == nil {
			text, secondary, err = b.parse(reply)
		}
		switch {
		case err == nil:
			return text, secondary, nil
		case errors.Is(err, errNothingHeard):
			return "", "", err
		case errors.Is(err, adapters.ErrBadReply):
			bad++
			if bad >= b.Attempts {
				return "", "", fmt.Errorf("%d bad replies, the last: %w", bad, err)
			}
		case errors.Is(err, adapters.ErrTransient):
			if retries == len(b.Backoff) {
				return "", "", errExhausted{err}
			}
			if !sleep(ctx, b.Backoff[retries]) {
				return "", "", ctx.Err()
			}
			retries++
		default:
			return "", "", err
		}
	}
}

// parse reads a reply, tidied, against the schema, and into the outputs.
func (b *HearingCall) parse(reply string) (text, secondary string, err error) {
	fields, err := b.schema.Check(adapters.JSONObject(reply))
	if err != nil {
		return "", "", adapters.BadReply(b.name, err.Error())
	}
	text, _ = fields[b.result.Text].(string)
	secondary, _ = fields[b.result.SecondaryText].(string)
	text, secondary = strings.TrimSpace(text), strings.TrimSpace(secondary)
	switch {
	case secondary == "":
		return "", "", errNothingHeard
	case text == "":
		return "", "", adapters.BadReply(b.name, "heard words, but made nothing of them")
	}
	return text, secondary, nil
}
