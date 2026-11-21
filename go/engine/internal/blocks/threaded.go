package blocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Thread is one conversation on the story model, shared by the threaded
// blocks that take part in it, such as Outline and Expand. It runs one call
// at a time, in the order the blocks joined: the next turn's Outline waits
// for Expand.
//
// The thread holds the System Message, sent once as the start of the provider
// thread, and the constraints, worded, which a block's call carries as often
// as its constraint frequency says.
//
// Its state is exported once, under its own name: the continuation, the model
// and offer it was made on, and the transcript, which seeds a fresh provider
// thread when the continuation cannot be used.
type Thread struct {
	name    string
	adapter adapters.Thread
	// system is the System Message; constraints the youth-protection rule,
	// worded for a call to carry, empty when there is none.
	system, constraints string

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

// NewThread builds a conversation that starts with the System Message; the
// constraints are what a call carries of the youth-protection rule.
func NewThread(name string, adapter adapters.Thread, system, constraints string) *Thread {
	// A missing adapter is a wiring mistake; it surfaces where the graph is built.
	if adapter == nil {
		panic("blocks: " + name + " needs a thread adapter")
	}
	return &Thread{name: name, adapter: adapter, system: system, constraints: constraints}
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

// endRound hands the turn back to the first seat, skipping the seats after
// the one holding it this round: a block whose call failed gives them no
// input, so they make no call.
func (t *Thread) endRound() {
	t.seats[0] <- struct{}{}
}

// cued says the thread's last kept call was the game's own message, such as
// the opening's cue.
func (t *Thread) cued() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := len(t.state.Transcript)
	return n > 0 && t.state.Transcript[n-1].Directive
}

// calls counts the calls the block has kept on the thread.
func (t *Thread) calls(block string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, e := range t.state.Transcript {
		if e.Block == block {
			n++
		}
	}
	return n
}

// threadCall is one block's call on the thread.
type threadCall struct {
	block, input string
	// directive sends the input as the game's own message, not the player's.
	directive bool
	// constrain makes the call carry the constraints.
	constrain bool
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
		System:       t.system,
		Input:        c.input,
		Directive:    c.directive,
		Schema:       c.schema,
		Speak:        c.speak,
		Continuation: t.state.Continuation,
		Transcript:   slices.Clone(t.state.Transcript),
		Piece:        c.piece,
		Audio:        c.audio,
	}
	if c.constrain {
		turn.Constraints = t.constraints
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
	t.state.Transcript = append(t.state.Transcript, turn.Exchange(c.block, reply.Text))
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

// Constrained is a block whose calls carry the constraints, as often as its
// frequency says: 1 with every call, 2 with every other. The frequency may
// change while the game runs; the genre checks it.
type Constrained interface {
	ports.Named
	ConstraintFrequency() int
	SetConstraintFrequency(n int)
}

// constraining is how often a threaded block's calls carry the constraints:
// each block counts its own calls on the thread, and its first call always
// carries them.
type constraining struct {
	// every is the frequency; zero is the default, 1.
	every atomic.Int32
}

// ConstraintFrequency is the frequency set, 1 for none or one below 1.
func (c *constraining) ConstraintFrequency() int {
	if n := c.every.Load(); n > 0 {
		return int(n)
	}
	return 1
}

func (c *constraining) SetConstraintFrequency(n int) { c.every.Store(int32(n)) }

// due says whether the call after the block's first calls carries the
// constraints.
func (c *constraining) due(calls int) bool {
	return calls%c.ConstraintFrequency() == 0
}

// Prompt is what a threaded block sends as its message: First right after the
// game's own cue, such as the opening's, Then after a player's action.
type Prompt struct {
	First, Then string
}

// message is the prompt for a call that follows the cue or an action.
func (p Prompt) message(cued bool) string {
	if cued {
		return p.First
	}
	return p.Then
}

// Threaded is one block's part in a thread: the text it receives makes the
// call, and its message is its prompt, the game's own message on the thread.
// Expand is one: the outline it narrates is already on the thread, as the
// reply before. Its outputs are the reply streamed piece by piece, each reply
// closed with PieceClose, and the whole reply once finished.
//
// While its Narration switch is on, its turns are spoken: the model says the
// reply, the transcript streams and is the reply, and the audio goes out on a
// third output as it comes.
//
// A finished reply is tidied by Clean. A reply that changed is replaced on
// the stream before it closes, and the whole reply is the tidied one; a reply
// with nothing left is a bad reply, discarded and made again from the same
// continuation, up to Attempts times.
//
// A failure that may pass is retried after each Backoff, from the same
// continuation, as the provider never completed the broken response; the
// pieces already streamed are dropped with PieceDiscard first, and the audio
// already spoken with AudioDiscard. A call that
// fails for good sends its text on the failure output and hands the thread's
// turn on, and the block is ready again.
type Threaded struct {
	failing
	constraining
	name   string
	prompt Prompt
	thread *Thread
	seat   int

	// Backoff is the wait before each retry of a failure that may pass, so a
	// call is made len(Backoff)+1 times at most. Set before Start.
	Backoff []time.Duration
	// Attempts is how many bad replies a call gets before it fails. Set
	// before Start.
	Attempts int
	// Clean tidies a finished reply; nil keeps every reply as it came. Set
	// before Start.
	Clean func(string) string

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
func NewThreaded(name string, thread *Thread, prompt Prompt) *Threaded {
	return &Threaded{
		name:     name,
		prompt:   prompt,
		thread:   thread,
		seat:     thread.join(),
		Backoff:  []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		Attempts: 3,
		in:       make(ports.TextInput, 16),
		meter:    meter{name: name},
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

// UsageOutPort reports what the calls have cost, bad replies included.
func (b *Threaded) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

// StateOutPort reports what this block is doing, with the provider's reason
// after a failed call.
func (b *Threaded) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Threaded) Start(ctx context.Context) {
	go func() {
		defer func() {
			b.stream.Close()
			b.whole.Close()
			b.audio.Close()
			b.failure.Close()
			b.state.Close()
			b.meter.usage.Close()
		}()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		if b.thread.calls(b.name) > 0 {
			// A resumed session: the block's first call is done, and it owes
			// whatever waits on that call a working-then-ready cycle.
			b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
			b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		}

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
				reply, err := b.call(ctx)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					b.thread.endRound()
					b.fail(&b.state, b.name, text, err)
					continue
				}
				b.thread.pass(b.seat)
				b.stream.Send(ports.PieceClose)
				b.whole.Send(reply)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// call continues the thread with the block's message until a reply is kept,
// retrying a failure that may pass after each backoff and a bad reply at once.
func (b *Threaded) call(ctx context.Context) (string, error) {
	calls := b.thread.calls(b.name)
	message, constrain := b.prompt.message(b.thread.cued()), b.due(calls)
	speak := b.narration.on.Load()
	bad, retries := 0, 0
	for {
		streamed, spoken := false, false
		var kept string
		reply, err := b.thread.call(ctx, threadCall{
			block:     b.name,
			input:     message,
			directive: true,
			constrain: constrain,
			speak:     speak,
			piece: func(piece string) {
				streamed = true
				b.stream.Send(ports.Piece(piece))
			},
			audio: func(chunk []byte) {
				spoken = true
				b.audio.Send(ports.AudioChunk(chunk))
			},
			accept: func(text string) error {
				kept = text
				if b.Clean != nil {
					kept = b.Clean(text)
				}
				if strings.TrimSpace(kept) == "" {
					return errors.New("nothing left once tidied")
				}
				return nil
			},
		})
		// A bad reply is billed all the same.
		if reply.Usage != (adapters.Usage{}) {
			b.meter.bill(reply.Usage)
		}
		if err == nil {
			if kept != reply.Text && kept != strings.TrimSpace(reply.Text) {
				b.stream.Send(ports.Replacement(kept))
			}
			return kept, nil
		}
		if streamed {
			b.stream.Send(ports.PieceDiscard)
		}
		if spoken {
			b.audio.Send(ports.AudioDiscard)
		}
		switch {
		case errors.Is(err, adapters.ErrBadReply):
			bad++
			if bad >= b.Attempts {
				return "", fmt.Errorf("%d bad replies, the last: %w", bad, err)
			}
		case errors.Is(err, adapters.ErrTransient):
			if retries == len(b.Backoff) {
				return "", errExhausted{err}
			}
			if !sleep(ctx, b.Backoff[retries]) {
				return "", ctx.Err()
			}
			retries++
		default:
			return "", err
		}
	}
}
