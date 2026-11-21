package blocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Result names the fields of a structured reply that feed the block's
// outputs: a text, a second text, and props, an object of strings. A name
// left empty feeds nothing.
type Result struct {
	Text, SecondaryText, Props string
}

// Structured is a threaded call with a structured result: one block's part in
// a thread, whose reply is JSON of the schema it is built with. The text it
// receives is its input, sent with the current props, as the thread's memory
// is not trusted for them; the reply's fields go out on its text, second text
// and props outputs, as its Result names them.
//
// Parsing is part of the call. A reply that does not parse against the
// schema, a refusal or one cut short is a bad reply: never kept on the thread,
// it is made again from the same continuation, up to Attempts times. Then the
// turn is dropped: nothing goes out, the thread skips the seats after this
// one, and the block is ready again, saying so with ReasonTurnFailed.
//
// A failure that may pass is retried after each Backoff, as on Threaded; one
// that will not, or an exhausted retry, reports the phase failed, which ends
// the game.
type Structured struct {
	name, instructions string
	schema             adapters.Schema
	result             Result
	thread             *Thread
	seat               int

	// Backoff is the wait before each retry of a failure that may pass. Set
	// before Start.
	Backoff []time.Duration
	// Attempts is how many bad replies a turn gets before it is dropped. Set
	// before Start.
	Attempts int

	in        ports.TextInput
	propsIn   ports.PropsInput
	out       ports.TextBroadcast
	secondary ports.TextBroadcast
	props     ports.PropsBroadcast
	state     ports.StateBroadcast
	meter     meter

	// current is the props as last known; only the block's goroutine uses it.
	current ports.PropMap
}

// NewStructured builds a block that takes part in the thread, in turn after
// the blocks built on it before.
func NewStructured(name string, thread *Thread, instructions string, schema adapters.Schema, result Result) *Structured {
	return &Structured{
		name:         name,
		instructions: instructions,
		schema:       schema,
		result:       result,
		thread:       thread,
		seat:         thread.join(),
		Backoff:      []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		Attempts:     3,
		in:           make(ports.TextInput, 16),
		propsIn:      make(ports.PropsInput, 16),
		meter:        meter{name: name},
		current:      ports.PropMap{},
	}
}

func (b *Structured) NodeName() string                  { return b.name }
func (b *Structured) TextInPort() chan<- string         { return b.in }
func (b *Structured) PropsInPort() chan<- ports.PropMap { return b.propsIn }

// RequiredInputs has props, so a genre that forgets to wire the current
// values fails validation instead of relying on the thread's memory.
func (b *Structured) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindText, ports.KindProps}
}

// TextOutPort is the reply's text field.
func (b *Structured) TextOutPort() <-chan string { return b.out.Subscribe() }

// SecondaryTextOutPort is the reply's second text field.
func (b *Structured) SecondaryTextOutPort() <-chan string { return b.secondary.Subscribe() }

// PropsOutPort is the reply's props, when its schema has them.
func (b *Structured) PropsOutPort() <-chan ports.PropMap { return b.props.Subscribe() }

// UsageOutPort reports what the calls have cost, bad replies included.
func (b *Structured) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

// StateOutPort reports what this block is doing; the session reads a failure
// or a dropped turn off it, so it needs no edge.
func (b *Structured) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Structured) Start(ctx context.Context) {
	go func() {
		defer func() {
			b.out.Close()
			b.secondary.Close()
			b.props.Close()
			b.state.Close()
			b.meter.usage.Close()
		}()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		// A required input is required: nothing is called before the first
		// snapshot of the current values, or a turn would decide from none.
		select {
		case <-ctx.Done():
			return
		case initial, open := <-b.propsIn:
			if !open {
				return
			}
			maps.Copy(b.current, initial)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case update, open := <-b.propsIn:
				if !open {
					return
				}
				maps.Copy(b.current, update)
			case text, open := <-b.in:
				if !open {
					return
				}
				if !b.turn(ctx, text) {
					return
				}
			}
		}
	}()
}

// turn makes one call in the block's seat, and reports false when the block
// cannot go on.
func (b *Structured) turn(ctx context.Context, text string) bool {
	if !b.thread.take(ctx, b.seat) {
		return false
	}
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
	got, err := b.call(ctx, text)
	if ctx.Err() != nil {
		return false
	}
	var dropped errDropped
	switch {
	case errors.As(err, &dropped):
		b.thread.drop(b.seat)
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady, Reason: ports.ReasonTurnFailed, Message: err.Error()})
		return true
	case err != nil:
		// The game cannot go on, so neither does the block, and the thread
		// stays with it.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseFailed, Reason: reason(err), Message: err.Error()})
		return false
	}

	b.thread.pass(b.seat)
	b.out.Send(got.text)
	b.secondary.Send(got.secondary)
	if got.props != nil {
		// Known at once, so the next turn sends them even before the store
		// has passed them back.
		maps.Copy(b.current, got.props)
		b.props.Send(got.props)
	}
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	return true
}

// parsed is a reply read into the block's outputs.
type parsed struct {
	text, secondary string
	// props is nil when the schema has none.
	props ports.PropMap
}

// errDropped is a turn that got only bad replies.
type errDropped struct{ error }

func (e errDropped) Unwrap() error { return e.error }

// call continues the thread until a reply parses, the bad replies run out, or
// a failure ends the game.
func (b *Structured) call(ctx context.Context, text string) (parsed, error) {
	input, err := b.input(text)
	if err != nil {
		return parsed{}, err
	}
	var got parsed
	accept := func(reply string) error {
		var err error
		got, err = b.parse(reply)
		return err
	}

	bad, retries := 0, 0
	for {
		reply, err := b.thread.call(ctx, threadCall{
			instructions: b.instructions,
			input:        input,
			schema:       &b.schema,
			accept:       accept,
		})
		// A bad reply is billed all the same.
		if reply.Usage != (adapters.Usage{}) {
			b.meter.bill(reply.Usage)
		}
		switch {
		case err == nil:
			return got, nil
		case errors.Is(err, adapters.ErrBadReply):
			bad++
			if bad >= b.Attempts {
				return parsed{}, errDropped{fmt.Errorf("%d bad replies, the last: %w", bad, err)}
			}
		case errors.Is(err, adapters.ErrTransient):
			if retries == len(b.Backoff) {
				return parsed{}, errExhausted{err}
			}
			if !sleep(ctx, b.Backoff[retries]) {
				return parsed{}, ctx.Err()
			}
			retries++
		default:
			return parsed{}, err
		}
	}
}

// input is what a call sends: the text, with the current props when the
// schema has props.
func (b *Structured) input(text string) (string, error) {
	in := map[string]any{"input": text}
	if b.hasProps() {
		in[b.result.Props] = b.current
	}
	blob, err := json.Marshal(in)
	return string(blob), err
}

func (b *Structured) hasProps() bool {
	return b.result.Props != "" && b.schema.Has(b.result.Props)
}

// parse reads a reply against the schema, and into the outputs.
func (b *Structured) parse(reply string) (parsed, error) {
	fields, err := b.schema.Check(reply)
	if err != nil {
		return parsed{}, err
	}
	got := parsed{}
	got.text, _ = fields[b.result.Text].(string)
	got.secondary, _ = fields[b.result.SecondaryText].(string)
	if b.hasProps() {
		got.props = ports.PropMap{}
		values, _ := fields[b.result.Props].(map[string]any)
		for name, value := range values {
			s, ok := value.(string)
			if !ok {
				return parsed{}, fmt.Errorf("reply.%s.%s: want a string", b.result.Props, name)
			}
			got.props[name] = s
		}
	}
	return got, nil
}
