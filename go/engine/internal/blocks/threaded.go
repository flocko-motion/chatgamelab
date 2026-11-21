package blocks

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Thread is one conversation on the story model, shared by the threaded
// blocks that take part in it, such as Outline and Expand. It runs one call at
// a time, in the order the blocks joined: the next turn's Outline waits for
// Expand.
//
// Its state is exported once, under its own name: the continuation, the model
// and offer it was made on, and the transcript, which seeds a fresh provider
// thread when the continuation cannot be used.
type Thread struct {
	name    string
	adapter adapters.Thread

	// seats pass the turn round: a block calls when it holds its seat, and
	// hands the turn to the next one when its call is done.
	seats []chan struct{}

	// Guarded because ExportState is called from the caller's goroutine while
	// the blocks are calling.
	mu    sync.Mutex
	state threadState
}

// threadState is what a stored session keeps of the thread.
type threadState struct {
	Continuation adapters.Continuation `json:"continuation"`
	// Transcript holds whole replies only.
	Transcript []adapters.Exchange `json:"transcript,omitempty"`
}

func NewThread(name string, adapter adapters.Thread) *Thread {
	// A missing adapter is a wiring mistake; it surfaces where the graph is built.
	if adapter == nil {
		panic("blocks: " + name + " needs a thread adapter")
	}
	return &Thread{name: name, adapter: adapter}
}

func (t *Thread) NodeName() string { return t.name }

// join gives a block its seat, the next in turn order. The first seat holds
// the turn. Called while the graph is built, before anything runs.
func (t *Thread) join() int {
	seat := make(chan struct{}, 1)
	if len(t.seats) == 0 {
		seat <- struct{}{}
	}
	t.seats = append(t.seats, seat)
	return len(t.seats) - 1
}

// take waits until the seat holds the turn, and reports false if the context
// ended first.
func (t *Thread) take(ctx context.Context, seat int) bool {
	select {
	case <-t.seats[seat]:
		return true
	case <-ctx.Done():
		return false
	}
}

// pass hands the turn to the next seat.
func (t *Thread) pass(seat int) {
	t.seats[(seat+1)%len(t.seats)] <- struct{}{}
}

// drop hands the turn back to the seat, skipping the seats after it this
// round: a block that drops its turn gives them no input, so they make no
// call. Only the seat that opens the round may drop it.
func (t *Thread) drop(seat int) {
	t.seats[seat] <- struct{}{}
}

// threadCall is one block's call on the thread.
type threadCall struct {
	instructions, input string
	// schema makes the call structured; nil is plain.
	schema *adapters.Schema
	// speak makes a plain call spoken.
	speak bool
	// piece hears each piece of the reply as it arrives; nil hears nothing.
	piece func(string)
	// audio hears each chunk of a spoken reply; nil hears nothing.
	audio func([]byte)
	// accept checks a finished reply before it is kept; an error makes it a
	// bad reply. Nil accepts any.
	accept func(text string) error
}

// call continues the thread from its state and keeps a finished call: the new
// continuation and the exchange. A failure, or a reply the call does not
// accept, leaves the state as it was; the reply's usage comes back either way.
func (t *Thread) call(ctx context.Context, c threadCall) (adapters.ThreadReply, error) {
	t.mu.Lock()
	turn := adapters.ThreadTurn{
		Instructions: c.instructions,
		Input:        c.input,
		Schema:       c.schema,
		Speak:        c.speak,
		Continuation: t.state.Continuation,
		Transcript:   slices.Clone(t.state.Transcript),
		Piece:        c.piece,
		Audio:        c.audio,
	}
	t.mu.Unlock()

	reply, err := t.adapter.Continue(ctx, turn)
	if err != nil {
		return reply, err
	}
	if c.accept != nil {
		if err := c.accept(reply.Text); err != nil {
			t.mu.Lock()
			if id := reply.Continuation.ID; id != "" && id == t.state.Continuation.ID {
				// The provider continued in place, as Mistral appends to its
				// stored conversation, so it holds the reply the call refuses:
				// the next call goes to a fresh thread seeded with the
				// transcript, which holds only kept replies.
				t.state.Continuation = adapters.Continuation{}
			}
			t.mu.Unlock()
			return reply, adapters.BadReply(t.name, err.Error())
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Continuation = reply.Continuation
	t.state.Transcript = append(t.state.Transcript, adapters.Exchange{Instructions: c.instructions, Input: c.input, Reply: reply.Text})
	return reply, nil
}

func (t *Thread) ExportState() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Continuation.ID == "" && len(t.state.Transcript) == 0 {
		return ""
	}
	blob, err := json.Marshal(t.state)
	if err != nil {
		return ""
	}
	return string(blob)
}

func (t *Thread) RestoreState(s string) {
	var state threadState
	if json.Unmarshal([]byte(s), &state) != nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = state
}

// Threaded is one block's part in a thread: its role is the instructions it is
// built with, and the text it receives is its input. Its outputs are the
// reply streamed piece by piece, each reply closed with PieceClose, and the
// whole reply once finished.
//
// While its Narration switch is on, its turns are spoken: the model says the
// reply, the transcript streams and is the reply, and the audio goes out on a
// third output as it comes.
//
// Like ToolCall, a failure that may pass is retried and one that will not
// reports the phase failed, which ends the game. A retry goes on from the same
// continuation, as the provider never completed the broken response; the
// pieces already streamed are dropped with PieceDiscard first, and the audio
// already spoken with AudioDiscard.
type Threaded struct {
	name, instructions string
	thread             *Thread
	seat               int

	// Backoff is the wait before each retry of a failure that may pass, so a
	// call is made len(Backoff)+1 times at most. Set before Start.
	Backoff []time.Duration

	in        ports.TextInput
	stream    ports.StreamBroadcast
	whole     ports.TextBroadcast
	audio     ports.AudioBroadcast
	state     ports.StateBroadcast
	meter     meter
	narration narration
}

// narration is the switch that has a block's turns spoken; off until set.
type narration struct{ on atomic.Bool }

func (n *narration) SetActive(on bool) { n.on.Store(on) }

// NewThreaded builds a block that takes part in the thread, in turn after the
// blocks built on it before.
func NewThreaded(name string, thread *Thread, instructions string) *Threaded {
	return &Threaded{
		name:         name,
		instructions: instructions,
		thread:       thread,
		seat:         thread.join(),
		Backoff:      []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		in:           make(ports.TextInput, 16),
		meter:        meter{name: name},
	}
}

func (b *Threaded) NodeName() string             { return b.name }
func (b *Threaded) TextInPort() chan<- string    { return b.in }
func (b *Threaded) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

// StreamOutPort is the reply as it arrives.
func (b *Threaded) StreamOutPort() <-chan ports.Piece { return b.stream.Subscribe() }

// TextOutPort is the whole reply, once finished.
func (b *Threaded) TextOutPort() <-chan string { return b.whole.Subscribe() }

// AudioOutPort is a spoken reply's audio as it arrives, AudioDiscard dropping
// what was spoken of a reply made again.
func (b *Threaded) AudioOutPort() <-chan ports.AudioChunk { return b.audio.Subscribe() }

// Narration is the switch that has the block's turns spoken, off until set; a
// turn already running keeps what it started with.
func (b *Threaded) Narration() ports.Switchable { return &b.narration }

// UsageOutPort reports what the calls have cost.
func (b *Threaded) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

// StateOutPort reports what this block is doing; the session reads a failure
// off it, so it needs no edge.
func (b *Threaded) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Threaded) Start(ctx context.Context) {
	go func() {
		defer func() { b.stream.Close(); b.whole.Close(); b.audio.Close(); b.state.Close(); b.meter.usage.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		for {
			select {
			case <-ctx.Done():
				return
			case text, open := <-b.in:
				if !open {
					return
				}
				if !b.thread.take(ctx, b.seat) {
					return
				}

				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				reply, err := b.call(ctx, text)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					// The game cannot go on, so neither does the block, and the
					// thread stays with it.
					b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseFailed, Reason: reason(err), Message: err.Error()})
					return
				}
				b.thread.pass(b.seat)
				b.stream.Send(ports.PieceClose)
				b.whole.Send(reply)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// call continues the thread, retrying a failure that may pass after each backoff.
func (b *Threaded) call(ctx context.Context, input string) (string, error) {
	speak := b.narration.on.Load()
	for attempt := 0; ; attempt++ {
		streamed, spoken := false, false
		reply, err := b.thread.call(ctx, threadCall{
			instructions: b.instructions,
			input:        input,
			speak:        speak,
			piece: func(piece string) {
				streamed = true
				b.stream.Send(ports.Piece(piece))
			},
			audio: func(chunk []byte) {
				spoken = true
				b.audio.Send(ports.AudioChunk(chunk))
			},
		})
		if err == nil {
			b.meter.bill(reply.Usage)
			return reply.Text, nil
		}
		if streamed {
			b.stream.Send(ports.PieceDiscard)
		}
		if spoken {
			b.audio.Send(ports.AudioDiscard)
		}
		if !errors.Is(err, adapters.ErrTransient) {
			return "", err
		}
		if attempt == len(b.Backoff) {
			return "", errExhausted{err}
		}
		if !sleep(ctx, b.Backoff[attempt]) {
			return "", ctx.Err()
		}
	}
}
