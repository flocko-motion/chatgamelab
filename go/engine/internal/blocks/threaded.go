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

// Thread is one conversation on a model, shared by the threaded blocks that
// take part in it. It runs one call at a time, in the order the blocks joined: each block holds
// a seat, and calls when its seat holds the turn.
//
// The thread holds the System Message, sent once as the start of the provider
// thread, and the constraints, worded, which a block's call carries as often
// as its constraint frequency says.
//
// Its state is exported once, under its own name: the continuation, the model
// and offer it was made on, and the transcript, which seeds a fresh provider
// thread when the continuation cannot be used.
//
// The thread compacts itself before it outgrows its model's context window:
// right after a call whose input passed CompactAt of its model's window,
// before the thread hands the turn on, so the next call waits. Input that
// carried audio, as a Realtime session holds spoken replies, is answered by a fresh provider
// thread seeded from the transcript, which holds every reply as text. Input
// that carried none is answered by a summary: the model is sent the summary
// request, on the thread itself, and a fresh provider thread is seeded with the
// System Message, the summary and the last KeptExchanges exchanges. The
// transcript keeps every exchange. A compaction that fails leaves the thread
// as it was, and it tries again after its next call.
type Thread struct {
	name    string
	adapter adapters.Thread
	// system is the System Message; constraints the youth-protection rule,
	// worded for a call to carry, empty when there is none.
	system, constraints string

	// Compaction is when and how the thread compacts. Set before Start; its
	// Request changes during play through SetSummaryRequest. Guarded by mu.
	Compaction Compaction

	// seats pass the turn round: a block calls when it holds its seat, and
	// hands the turn to the next one when its call is done.
	seats []chan struct{}

	// calling holds one provider call at a time: a block's call, or a
	// compaction, asked for by the thread or on demand.
	calling sync.Mutex

	// Guarded because ExportState is called from the caller's goroutine while
	// the blocks are calling.
	mu    sync.Mutex
	state threadState
	// last is the usage of the last call kept, which says whether the thread
	// compacts; zero once it has been read.
	last adapters.Usage
	// fill is the usage of the last call kept, how full the provider thread
	// is; zero once a compaction replaced it, until the next call.
	fill adapters.Usage
}

// ThreadFill is how full a thread is: the input of its last call, in tokens, and
// the context window of the model it ran on, zero when unknown. Both are
// zero before a call, and after a compaction until the next one.
type ThreadFill struct {
	Tokens int64 `json:"tokens"`
	Window int   `json:"window"`
}

// Fill reports how full the thread is.
func (t *Thread) Fill() ThreadFill {
	t.mu.Lock()
	defer t.mu.Unlock()
	fill := ThreadFill{Tokens: t.fill.Context()}
	if t.Compaction.Window != nil && t.fill.Model != "" {
		fill.Window = t.Compaction.Window(t.fill.Model)
	}
	return fill
}

// threadState is what a stored session keeps of the thread.
type threadState struct {
	Continuation adapters.Continuation `json:"continuation"`
	// Transcript holds whole replies only, every one the thread kept.
	Transcript []adapters.Exchange `json:"transcript,omitempty"`
	// Summary is the transcript before From, condensed; empty before the thread
	// first compacted. From is the first exchange after it, where a seed's
	// transcript starts.
	Summary string `json:"summary,omitempty"`
	From    int    `json:"from,omitempty"`
}

// Compaction is what a thread needs to compact itself: its model's context
// window and the summary request. A thread missing either never compacts.
type Compaction struct {
	// Window is a model's context window in tokens, zero when unknown.
	Window func(model string) int
	// Request asks the model for a summary of the thread so far, sent as the
	// game's own message; its reply goes to the engine only.
	Request string
}

// seed is the part of the transcript a fresh provider thread is seeded with:
// the exchanges after the summary. Called with mu held.
func (s threadState) seed() []adapters.Exchange {
	return slices.Clone(s.Transcript[min(s.From, len(s.Transcript)):])
}

// CompactAt is the share of its model's context window a call's input may
// take before the thread compacts.
const CompactAt = 0.5

// KeptExchanges is how many of the last exchanges a summary leaves word for
// word, seeded after it.
const KeptExchanges = 6

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

// SetSystem changes the System Message from the next call on. The thread
// keeps what it holds, and the next call starts a fresh provider thread
// seeded from it, led by the new System Message: a provider takes the System
// Message once, at the start of its thread.
func (t *Thread) SetSystem(system string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.system = system
	t.state.Continuation = adapters.Continuation{}
}

// SetConstraints changes what a call carries of the youth-protection rule,
// from the next call on.
func (t *Thread) SetConstraints(constraints string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.constraints = constraints
}

// SetSummaryRequest changes what the thread asks for when it compacts, from
// the next compaction on.
func (t *Thread) SetSummaryRequest(request string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Compaction.Request = request
}

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

// pass hands the turn to the next seat, once the thread has compacted if the
// call just kept crowded its window. The block that made the call waits,
// working, and pays for the summary.
func (t *Thread) pass(ctx context.Context, seat int, m *meter, backoff []time.Duration) {
	if last, crowded := t.takeLast(); crowded {
		if used := t.compact(ctx, last.HeardAudio(), backoff); used != (adapters.Usage{}) {
			m.bill(used)
		}
	}
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
	t.calling.Lock()
	defer t.calling.Unlock()
	t.mu.Lock()
	turn := adapters.ThreadTurn{
		System:       t.system,
		Input:        c.input,
		Directive:    c.directive,
		Schema:       c.schema,
		Speak:        c.speak,
		Continuation: t.state.Continuation,
		Summary:      t.state.Summary,
		Transcript:   t.state.seed(),
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
	t.last, t.fill = reply.Usage, reply.Usage
	return reply, nil
}

// takeLast hands over the usage of the last call kept, and whether it took
// more of its model's window than CompactAt. It forgets the call, so the call
// is answered once.
func (t *Thread) takeLast() (adapters.Usage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	last := t.last
	t.last = adapters.Usage{}
	if t.Compaction.Window == nil || t.Compaction.Request == "" || last.Model == "" {
		return last, false
	}
	window := t.Compaction.Window(last.Model)
	return last, window > 0 && float64(last.Context()) > CompactAt*float64(window)
}

// CompactNow compacts the thread at once, between calls, with a summary
// whatever the last call's input carried, as a block asks for it on demand.
// It reports what the summary request cost; a thread without a summary
// request does nothing.
func (t *Thread) CompactNow(ctx context.Context, backoff []time.Duration) adapters.Usage {
	t.mu.Lock()
	request := t.Compaction.Request
	t.mu.Unlock()
	if request == "" {
		return adapters.Usage{}
	}
	return t.compact(ctx, false, backoff)
}

// compact replaces the provider thread: with a fresh one seeded from the
// transcript when reseed says so, as for a call whose input carried audio,
// otherwise with one seeded from a summary. A summary request that may pass
// is made again after each backoff. It reports what the request cost, and
// leaves the thread as it was when it fails.
func (t *Thread) compact(ctx context.Context, reseed bool, backoff []time.Duration) adapters.Usage {
	t.calling.Lock()
	defer t.calling.Unlock()
	t.mu.Lock()
	if reseed {
		t.state.Continuation = adapters.Continuation{}
		t.fill = adapters.Usage{}
		t.mu.Unlock()
		return adapters.Usage{}
	}
	turn := adapters.ThreadTurn{
		System:       t.system,
		Input:        t.Compaction.Request,
		Directive:    true,
		Constraints:  t.constraints,
		Continuation: t.state.Continuation,
		Summary:      t.state.Summary,
		Transcript:   t.state.seed(),
	}
	t.mu.Unlock()

	var spent adapters.Usage
	for retries := 0; ; retries++ {
		reply, err := t.adapter.Continue(ctx, turn)
		spent.Add(reply.Usage)
		if err == nil && strings.TrimSpace(reply.Text) == "" {
			err = adapters.BadReply(t.name, "an empty summary")
		}
		if err == nil {
			t.mu.Lock()
			defer t.mu.Unlock()
			t.state.Summary = strings.TrimSpace(reply.Text)
			t.state.From = max(len(t.state.Transcript)-KeptExchanges, 0)
			t.state.Continuation = adapters.Continuation{}
			t.fill = adapters.Usage{}
			return spent
		}
		if id := reply.Continuation.ID; id != "" && id == turn.Continuation.ID {
			// The provider continued in place, as Mistral appends to its
			// stored conversation, so it holds the request: the next call
			// goes to a fresh thread seeded from the transcript.
			t.mu.Lock()
			t.state.Continuation = adapters.Continuation{}
			t.mu.Unlock()
		}
		if !errors.Is(err, adapters.ErrTransient) || retries == len(backoff) || !sleep(ctx, backoff[retries]) {
			return spent
		}
	}
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

// OnThread is a block that takes part in a thread: it names the thread, and
// asks it to compact on demand.
type OnThread interface {
	ports.Named
	Thread() *Thread
	Compact()
}

// request asks once: a request while one waits is the same request.
func request(asks chan struct{}) {
	select {
	case asks <- struct{}{}:
	default:
	}
}

// compactNow compacts a block's thread on its request, the block working
// meanwhile and paying for the summary.
func compactNow(ctx context.Context, t *Thread, name string, state *ports.StateBroadcast, m *meter, backoff []time.Duration) {
	state.Send(ports.State{Node: name, Phase: ports.PhaseWorking})
	if used := t.CompactNow(ctx, backoff); used != (adapters.Usage{}) {
		m.bill(used)
	}
	state.Send(ports.State{Node: name, Phase: ports.PhaseReady})
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
// Its outputs are the reply streamed piece by piece, each reply
// closed with PieceClose, and the whole reply once finished.
//
// While its Spoken switch is on, its calls are spoken: the model says the
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
	prompt tunable[Prompt]
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

	in       ports.TextInput
	compacts chan struct{}
	stream   ports.StreamBroadcast
	whole    ports.TextBroadcast
	audio    ports.AudioBroadcast
	state    ports.StateBroadcast
	meter    meter
	spoken   spoken
}

// spoken is the switch that has a block's calls spoken; off until set.
type spoken struct{ on atomic.Bool }

func (s *spoken) SetActive(on bool) { s.on.Store(on) }

// NewThreaded builds a block that takes part in the thread, in turn after the
// blocks built on it before.
func NewThreaded(name string, thread *Thread, prompt Prompt) *Threaded {
	return &Threaded{
		name:     name,
		prompt:   newTunable(prompt),
		thread:   thread,
		seat:     thread.join(),
		Backoff:  []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		Attempts: 3,
		in:       make(ports.TextInput, 16),
		compacts: make(chan struct{}, 1),
		meter:    meter{name: name},
	}
}

// SetPrompt changes its message from the next call on.
func (b *Threaded) SetPrompt(prompt Prompt) { b.prompt.set(prompt) }

// Thread is the thread the block takes part in.
func (b *Threaded) Thread() *Thread { return b.thread }

// Compact asks the block's thread to compact now, between calls; the block
// shows working meanwhile and pays for the summary. A request while one waits
// is the same request.
func (b *Threaded) Compact() { request(b.compacts) }

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

// Spoken is the switch that has the block's calls spoken, off until set; a
// call already running keeps what it started with.
func (b *Threaded) Spoken() ports.Switchable { return &b.spoken }

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
			case <-b.compacts:
				compactNow(ctx, b.thread, b.name, &b.state, &b.meter, b.Backoff)
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
				b.stream.Send(ports.PieceClose)
				b.whole.Send(reply)
				b.thread.pass(ctx, b.seat, &b.meter, b.Backoff)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// call continues the thread with the block's message until a reply is kept,
// retrying a failure that may pass after each backoff and a bad reply at once.
func (b *Threaded) call(ctx context.Context) (string, error) {
	calls := b.thread.calls(b.name)
	message, constrain := b.prompt.get().message(b.thread.cued()), b.due(calls)
	speak := b.spoken.on.Load()
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
