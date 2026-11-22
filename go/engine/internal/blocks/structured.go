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

// Structured is a threaded call whose reply is JSON of its schema, its fields
// going out as its Result names them. Its input goes as JSON with the current
// props, which the thread's memory is not trusted for. The cue on its opening
// input goes as it is, as the game's own message, only on a thread without a
// call of the block's, so a resumed session never opens twice.
//
// A reply is tidied of a fence and the text around its JSON. One that still
// does not parse is a bad reply, never kept, made again up to Attempts times.
// A call that fails for good sends its text on the failure output and hands
// the turn past the seats after this one.
type Structured struct {
	failing
	constraining
	name   string
	schema adapters.Schema
	result Result
	thread *Thread
	seat   int

	// Backoff is the wait before each retry of a failure that may pass. Set
	// before Start.
	Backoff []time.Duration
	// Attempts is how many bad replies a call gets before it fails. Set
	// before Start.
	Attempts int

	in        ports.TextInput
	opening   ports.TextInput
	compacts  chan struct{}
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
func NewStructured(name string, thread *Thread, schema adapters.Schema, result Result) *Structured {
	return &Structured{
		name:     name,
		schema:   schema,
		result:   result,
		thread:   thread,
		seat:     thread.join(),
		Backoff:  []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		Attempts: 3,
		in:       make(ports.TextInput, 16),
		opening:  make(ports.TextInput, 1),
		compacts: make(chan struct{}, 1),
		propsIn:  make(ports.PropsInput, 16),
		meter:    meter{name: name},
		current:  ports.PropMap{},
	}
}

func (b *Structured) NodeName() string { return b.name }

// Thread is the thread the block takes part in.
func (b *Structured) Thread() *Thread { return b.thread }

// Compact asks the block's thread to compact now, between calls; the block
// shows working meanwhile and pays for the summary. A request while one waits
// is the same request.
func (b *Structured) Compact()                          { request(b.compacts) }
func (b *Structured) TextInPort() chan<- string         { return b.in }
func (b *Structured) OpeningInPort() chan<- string      { return b.opening }
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

// StateOutPort reports what this block is doing, with the provider's reason
// after a failed call.
func (b *Structured) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Structured) Start(ctx context.Context) {
	go func() {
		defer func() {
			b.out.Close()
			b.secondary.Close()
			b.props.Close()
			b.failure.Close()
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
			case <-b.compacts:
				compactNow(ctx, b.thread, b.name, &b.state, &b.meter, b.Backoff)
			case cue, open := <-b.opening:
				if !open {
					return
				}
				if b.thread.calls(b.name) > 0 {
					// Opened before this session was resumed: the cue is
					// done with, and whatever waits on it is owed a cycle.
					b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
					b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
					continue
				}
				if !b.turn(ctx, cue, true) {
					return
				}
			case text, open := <-b.in:
				if !open {
					return
				}
				if !b.turn(ctx, text, false) {
					return
				}
			}
		}
	}()
}

// turn makes one call in the block's seat, and reports false when the
// session has ended.
func (b *Structured) turn(ctx context.Context, text string, opening bool) bool {
	if !b.thread.take(ctx, b.seat) {
		return false
	}
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
	got, err := b.call(ctx, text, opening)
	if ctx.Err() != nil {
		return false
	}
	if err != nil {
		b.thread.endRound()
		b.fail(&b.state, b.name, text, err)
		return true
	}

	b.out.Send(got.text)
	b.secondary.Send(got.secondary)
	if got.props != nil {
		// Known at once, so the next turn sends them even before the store
		// has passed them back.
		maps.Copy(b.current, got.props)
		b.props.Send(got.props)
	}
	b.thread.pass(ctx, b.seat, &b.meter, b.Backoff)
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	return true
}

// parsed is a reply read into the block's outputs.
type parsed struct {
	text, secondary string
	// props is nil when the schema has none.
	props ports.PropMap
}

// call continues the thread until a reply parses or the call fails for good.
// The opening always carries the constraints; an action at its frequency.
func (b *Structured) call(ctx context.Context, text string, opening bool) (parsed, error) {
	input, constrain := text, true
	if !opening {
		var err error
		if input, err = b.input(text); err != nil {
			return parsed{}, err
		}
		constrain = b.due(b.thread.calls(b.name))
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
			block:     b.name,
			input:     input,
			directive: opening,
			constrain: constrain,
			schema:    &b.schema,
			accept:    accept,
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
				return parsed{}, fmt.Errorf("%d bad replies, the last: %w", bad, err)
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

// parse reads a reply, tidied, against the schema, and into the outputs.
func (b *Structured) parse(reply string) (parsed, error) {
	fields, err := b.schema.Check(adapters.JSONObject(reply))
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
