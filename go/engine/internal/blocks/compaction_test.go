package blocks_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// recording passes each turn on to the thread adapter it wraps and keeps it,
// so a test sees what each call sent.
type recording struct {
	adapters.Thread
	mu    sync.Mutex
	turns []adapters.ThreadTurn
	// fail answers the summary request with this error instead, while set;
	// blank answers it with nothing.
	fail  error
	blank bool
}

func (r *recording) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	r.mu.Lock()
	r.turns = append(r.turns, turn)
	fail, blank := r.fail, r.blank
	r.mu.Unlock()
	if fail != nil && summarising(turn) {
		return adapters.ThreadReply{}, fail
	}
	if blank && summarising(turn) {
		reply, err := r.Thread.Continue(ctx, turn)
		reply.Text = "  "
		return reply, err
	}
	return r.Thread.Continue(ctx, turn)
}

func (r *recording) seen() []adapters.ThreadTurn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]adapters.ThreadTurn(nil), r.turns...)
}

func (r *recording) heal() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fail = nil
}

// summaryRequest is what the tests' threads ask for a summary with.
const summaryRequest = "summarise the conversation so far"

// summarising says the turn asks the model for a summary.
func summarising(turn adapters.ThreadTurn) bool { return turn.Input == summaryRequest }

// windowed is a thread on the adapter that compacts in the mock's context
// windows.
func windowed(adapter adapters.Thread) *blocks.Thread {
	thread := newThread(adapter)
	thread.Compaction = blocks.Compaction{Window: mock.Models.Window, Request: summaryRequest}
	return thread
}

// play has the block make calls until done says to stop, at most limit.
func play(t *testing.T, b threadedBlock, limit int, done func() bool) {
	t.Helper()
	for range limit {
		b.TextInPort() <- "the player acts"
		receive(t, b.whole, "a reply")
		reply(t, b.stream)
		settle(t, b)
		if done() {
			return
		}
	}
	t.Fatalf("still going after %d calls", limit)
}

// settle waits until the block is ready after its call: a compaction after
// the call is done. The block works once per call, so the ready it reports
// as it starts is passed over.
func settle(t *testing.T, b threadedBlock) {
	t.Helper()
	for receive(t, b.states, "the block's state").Phase != ports.PhaseWorking {
	}
	for receive(t, b.states, "the block's state").Phase != ports.PhaseReady {
	}
}

// storedState reads the thread's exported state.
func storedState(t *testing.T, thread *blocks.Thread) (summary string, from int, transcript int) {
	t.Helper()
	summary, from, exchanges := storedTranscript(t, thread)
	return summary, from, len(exchanges)
}

// storedTranscript reads the thread's exported state, its transcript whole.
func storedTranscript(t *testing.T, thread *blocks.Thread) (summary string, from int, transcript []adapters.Exchange) {
	t.Helper()
	var state struct {
		Summary    string              `json:"summary"`
		From       int                 `json:"from"`
		Transcript []adapters.Exchange `json:"transcript"`
	}
	if err := json.Unmarshal([]byte(thread.ExportState()), &state); err != nil {
		t.Fatalf("exported state: %v", err)
	}
	return state.Summary, state.From, state.Transcript
}

// A thread whose input passes half its model's window asks the model for the
// story so far, as the game's own message, and goes on from a fresh provider
// thread seeded with the System Message, that summary and the last six
// exchanges word for word. The transcript keeps every exchange; the summary
// is not one of them.
func TestThreadCompactsNearItsWindow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{}}
	thread := windowed(adapter)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)

	play(t, expand, 40, func() bool {
		s, _, _ := storedState(t, thread)
		return s != ""
	})
	turns := adapter.seen()
	request := turns[len(turns)-1]
	if !summarising(request) || !request.Directive || request.Speak || request.Schema != nil {
		t.Fatalf("the last call %+v, want a plain summary request as the game's own message", request)
	}
	if request.Continuation.ID == "" {
		t.Errorf("the summary request started a fresh thread, want it on the story thread itself")
	}

	summary, from, exchanges := storedTranscript(t, thread)
	kept := len(exchanges)
	if !strings.HasPrefix(summary, "[story turn") {
		t.Errorf("summary %q, want the model's reply to the request", summary)
	}
	if kept != len(turns)-1 || from != kept-6 {
		t.Errorf("transcript of %d from %d, want all %d story calls, the last six after the summary", kept, from, len(turns)-1)
	}

	expand.TextInPort() <- "the player acts on"
	receive(t, expand.whole, "the reply after compaction")
	reply(t, expand.stream)
	settle(t, expand)
	next := adapter.seen()[len(turns)]
	if next.Continuation.ID != "" {
		t.Errorf("the next call went on from %q, want a fresh provider thread", next.Continuation.ID)
	}
	if next.System != "the system message" || next.Summary != summary || len(next.Transcript) != 6 {
		t.Errorf("seeded with system %q, summary %q and %d exchanges, want the System Message, the summary and six",
			next.System, next.Summary, len(next.Transcript))
	}
	if got, want := next.Transcript, exchanges[kept-6:]; !reflect.DeepEqual(got, want) {
		t.Errorf("seeded with %+v, want the last six exchanges %+v", got, want)
	}
}

// A spoken thread whose input carried audio first goes on from a fresh
// provider thread seeded from its transcript, which holds every reply as
// text, without asking for a summary.
func TestSpokenThreadReseedsFromTextBeforeItSummarises(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{Name: "mock-realtime"}}
	thread := windowed(adapter)
	expand := startThreaded(t, thread, "expand")
	expand.Spoken().SetActive(true)
	expand.Start(ctx)

	reseeded := func() bool {
		turns := adapter.seen()
		return len(turns) > 1 && turns[len(turns)-1].Continuation.ID == ""
	}
	play(t, expand, 40, reseeded)
	turns := adapter.seen()
	for _, turn := range turns {
		if summarising(turn) {
			t.Fatalf("asked for a summary before reseeding from text")
		}
	}
	fresh := turns[len(turns)-1]
	if fresh.Summary != "" || len(fresh.Transcript) != len(turns)-1 {
		t.Errorf("reseeded with summary %q and %d exchanges, want the whole transcript, %d", fresh.Summary, len(fresh.Transcript), len(turns)-1)
	}

	// Without the audio, the thread grows on from text until it summarises.
	play(t, expand, 60, func() bool {
		s, _, _ := storedState(t, thread)
		return s != ""
	})
}

// A restored thread that cannot go on from its continuation seeds a fresh
// provider thread with the summary and the exchanges after it.
func TestRestoredThreadSeedsFromTheSummary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{}}
	thread := windowed(adapter)
	thread.RestoreState(`{"summary":"the story so far","from":2,"transcript":[` +
		`{"block":"expand","input":"a","reply":"b"},{"block":"expand","input":"c","reply":"d"},` +
		`{"block":"expand","input":"e","reply":"f"}]}`)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)

	expand.TextInPort() <- "the player acts"
	receive(t, expand.whole, "the reply")
	settle(t, expand)
	turn := adapter.seen()[0]
	if turn.Summary != "the story so far" || len(turn.Transcript) != 1 || turn.Transcript[0].Input != "e" {
		t.Errorf("seeded with summary %q and %+v, want the summary and the one exchange after it", turn.Summary, turn.Transcript)
	}
	if _, _, kept := storedState(t, thread); kept != 4 {
		t.Errorf("transcript of %d, want all four exchanges kept", kept)
	}
}

// A summary request that fails leaves the thread as it was: the game plays
// on, from the same continuation, and the thread asks again after its next
// call.
func TestFailedSummaryLeavesTheThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{}, fail: errors.New("responses api: 400: bad request")}
	thread := windowed(adapter)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)

	asked := func() bool { return slices.ContainsFunc(adapter.seen(), summarising) }
	play(t, expand, 40, asked)
	if s, _, _ := storedState(t, thread); s != "" {
		t.Fatalf("summary %q after a failed request, want none", s)
	}

	before := len(adapter.seen())
	stateBefore := thread.ExportState()
	expand.TextInPort() <- "the player acts on"
	if got := receive(t, expand.whole, "the reply after the failure"); got == "" {
		t.Fatalf("no reply after the failed summary")
	}
	reply(t, expand.stream)
	settle(t, expand)
	turns := adapter.seen()
	if turns[before].Continuation.ID == "" || !strings.Contains(stateBefore, turns[before].Continuation.ID) {
		t.Errorf("went on from %q, want the continuation the failure left", turns[before].Continuation.ID)
	}
	if len(turns) < before+2 || !summarising(turns[before+1]) {
		t.Fatalf("no new summary request after the next call")
	}

	adapter.heal()
	play(t, expand, 5, func() bool {
		s, _, _ := storedState(t, thread)
		return s != ""
	})
}

// A summary request answered with nothing leaves the thread as it was, and
// the game plays on.
func TestBlankSummaryLeavesTheThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{}, blank: true}
	thread := windowed(adapter)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)

	play(t, expand, 40, func() bool { return slices.ContainsFunc(adapter.seen(), summarising) })
	if s, from, _ := storedState(t, thread); s != "" || from != 0 {
		t.Fatalf("summary %q from %d after a blank reply, want none", s, from)
	}
	expand.TextInPort() <- "the player acts on"
	if got := receive(t, expand.whole, "the reply after the blank summary"); !strings.HasPrefix(got, "[story turn") {
		t.Errorf("the game said %q after a blank summary", got)
	}
}

// A block asks its thread to compact on demand: the thread summarises at
// once, whatever its fill, the block working meanwhile, and the next call
// goes on from a fresh provider thread seeded with the summary.
func TestBlockCompactsItsThreadOnDemand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &recording{Thread: mock.Thread{}}
	thread := windowed(adapter)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)
	play(t, expand, 2, func() bool { return len(adapter.seen()) == 2 })

	expand.Compact()
	settle(t, expand)
	if summary, from, kept := storedState(t, thread); summary == "" || from != 0 || kept != 2 {
		t.Fatalf("summary %q from %d of %d exchanges, want a summary keeping both", summary, from, kept)
	}
	if fill := thread.Fill(); fill != (blocks.ThreadFill{}) {
		t.Errorf("fill %+v after compaction, want none until the next call", fill)
	}

	expand.TextInPort() <- "the player acts on"
	receive(t, expand.whole, "the reply after compaction")
	turns := adapter.seen()
	if next := turns[len(turns)-1]; next.Continuation.ID != "" || next.Summary == "" {
		t.Errorf("went on from %q with summary %q, want a fresh thread seeded with the summary", next.Continuation.ID, next.Summary)
	}
}

// A thread reports how full it is: its last call's input, and the window of
// the model the call ran on.
func TestThreadReportsItsFill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := windowed(mock.Thread{})
	if fill := thread.Fill(); fill != (blocks.ThreadFill{}) {
		t.Errorf("fill %+v before a call, want none", fill)
	}
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)
	play(t, expand, 1, func() bool { return true })
	if fill := thread.Fill(); fill.Tokens <= 0 || fill.Window != mock.Window {
		t.Errorf("fill %+v, want the call's input in the mock's window of %d", fill, mock.Window)
	}
}
