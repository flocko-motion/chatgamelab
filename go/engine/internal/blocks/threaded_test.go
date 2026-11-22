package blocks_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// threadedBlock is one block, subscribed to what it puts out.
type threadedBlock struct {
	*blocks.Threaded
	stream  <-chan ports.Piece
	whole   <-chan string
	failure <-chan ports.Failure
	states  <-chan ports.State
}

// startThreaded builds a block whose message is "first as <name>" right after
// the game's own message on the thread, "then as <name>" otherwise; the
// caller starts it. A threaded block's own message is the game's, so a block
// after another on the thread sends its first.
func startThreaded(t *testing.T, thread *blocks.Thread, name string) threadedBlock {
	t.Helper()
	b := blocks.NewThreaded(name, thread, blocks.Prompt{First: "first as " + name, Then: "then as " + name})
	b.Backoff = []time.Duration{0, 0}
	return threadedBlock{Threaded: b, stream: b.StreamOutPort(), whole: b.TextOutPort(), failure: b.FailureOutPort(), states: b.StateOutPort()}
}

func newThread(adapter adapters.Thread) *blocks.Thread {
	return blocks.NewThread("story-thread", adapter, "the system message", "NARRATION RULES must be respected: be kind")
}

// reply collects one streamed reply up to its close.
func reply(t *testing.T, stream <-chan ports.Piece) []ports.Piece {
	t.Helper()
	var pieces []ports.Piece
	for {
		piece := receive(t, stream, "a piece")
		pieces = append(pieces, piece)
		if piece == ports.PieceClose {
			return pieces
		}
	}
}

func joined(pieces []ports.Piece) string {
	var b strings.Builder
	for _, p := range pieces {
		b.WriteString(string(p))
	}
	return b.String()
}

// Two blocks on one thread: the second's call continues the conversation the
// first's call left, and the thread's state carries both. The text a block
// receives makes the call; its message is its prompt.
func TestTwoThreadedBlocksShareOneThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := newThread(mock.Thread{})
	outline := startThreaded(t, thread, "outline")
	expand := startThreaded(t, thread, "expand")
	outline.Start(ctx)
	expand.Start(ctx)

	outline.TextInPort() <- "The player crosses."
	plot := receive(t, outline.whole, "the outline")
	if plot != "[story turn 1] then as outline" {
		t.Fatalf("outline said %q", plot)
	}
	if got := joined(reply(t, outline.stream)); got != plot {
		t.Errorf("the stream said %q, the whole reply %q", got, plot)
	}

	expand.TextInPort() <- plot
	prose := receive(t, expand.whole, "the prose")
	if !strings.HasPrefix(prose, "[story turn 2]") {
		t.Errorf("expand said %q, want turn 2 of the same thread", prose)
	}

	state := thread.ExportState()
	for _, want := range []string{`"id":"mock-thread-2"`, `"model":"mock-story"`, `"block":"outline"`, `"input":"then as outline"`} {
		if !strings.Contains(state, want) {
			t.Errorf("exported state %s lacks %s", state, want)
		}
	}
}

// A restored thread continues where the stored one left off.
func TestRestoredThreadContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := newThread(mock.Thread{})
	b := startThreaded(t, first, "expand")
	b.Start(ctx)
	b.TextInPort() <- "one"
	receive(t, b.whole, "the first reply")
	stored := first.ExportState()
	cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resumed := newThread(mock.Thread{})
	resumed.RestoreState(stored)
	again := startThreaded(t, resumed, "expand")
	again.Start(ctx2)
	// The block made its first call before the session was stored, so it owes
	// whatever waits on that call a cycle as it starts.
	expectPhases(t, again.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	again.TextInPort() <- "two"
	if got := receive(t, again.whole, "the resumed reply"); !strings.HasPrefix(got, "[story turn 2]") {
		t.Errorf("a resumed thread said %q, want turn 2", got)
	}
}

// The thread runs one call at a time, in turn: the next turn's Outline waits
// for Expand, even when its input arrives first.
func TestThreadRunsInTurn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{}
	thread := newThread(adapter)
	outline := startThreaded(t, thread, "outline")
	expand := startThreaded(t, thread, "expand")
	outline.Start(ctx)
	expand.Start(ctx)

	outline.TextInPort() <- "turn 1"
	receive(t, outline.whole, "outline 1")
	// The next turn's outline is asked before this turn's expand.
	outline.TextInPort() <- "turn 2"
	time.Sleep(20 * time.Millisecond)
	expand.TextInPort() <- "expand 1"
	receive(t, expand.whole, "expand 1")
	receive(t, outline.whole, "outline 2")

	if got := adapter.inputs(); strings.Join(got, ",") != "then as outline,first as expand,first as outline" {
		t.Errorf("calls ran as %v, want in turn", got)
	}
}

// A stream that breaks partway is made again from the same continuation, and
// the pieces already sent are dropped first.
func TestThreadedRetriesABrokenStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{breaks: 1}
	thread := newThread(adapter)
	thread.RestoreState(`{"continuation":{"id":"resp_7"}}`)
	b := startThreaded(t, thread, "expand")
	b.Start(ctx)

	b.TextInPort() <- "go"
	pieces := reply(t, b.stream)
	want := []ports.Piece{"half ", ports.PieceDiscard, "whole ", "reply", ports.PieceClose}
	if joined(pieces) != joined(want) || len(pieces) != len(want) {
		t.Errorf("streamed %q, want %q", pieces, want)
	}
	if got := receive(t, b.whole, "the reply"); got != "whole reply" {
		t.Errorf("whole reply %q", got)
	}
	if from := adapter.continuations(); len(from) != 2 || from[0] != "resp_7" || from[1] != "resp_7" {
		t.Errorf("continued from %v, want resp_7 both times", from)
	}
	expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// A spoken reply broken partway drops its audio with the text: the retried
// reply speaks from its start.
func TestThreadedRetryDiscardsSpokenAudio(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{breaks: 1}
	b := startThreaded(t, newThread(adapter), "expand")
	b.Spoken().SetActive(true)
	audio := b.AudioOutPort()
	b.Start(ctx)

	b.TextInPort() <- "go"
	reply(t, b.stream)
	var got []ports.AudioChunk
	for range 3 {
		got = append(got, receive(t, audio, "a chunk"))
	}
	if len(got[0]) != 2 || got[0][0] != 1 || !got[1].Discard() || len(got[2]) != 2 || got[2][0] != 2 {
		t.Errorf("audio %v, want the broken chunk, the discard, the retried chunk", got)
	}
}

// Its Spoken switch on, a block's call on the shared thread is spoken: the
// transcript streams as text, the audio beside it, and the transcript is the
// reply the thread keeps. Off, the same call is silent.
func TestThreadedSpeaksWhileSpokenIsOn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := newThread(mock.Thread{Name: "mock-realtime"})
	outline := startThreaded(t, thread, "outline")
	expand := startThreaded(t, thread, "expand")
	audio := expand.AudioOutPort()
	usage := expand.UsageOutPort()
	outline.Start(ctx)
	expand.Start(ctx)

	expand.Spoken().SetActive(true)
	outline.TextInPort() <- "plot"
	receive(t, outline.whole, "the outline")
	expand.TextInPort() <- "NARRATE"
	prose := receive(t, expand.whole, "the prose")
	if got := joined(reply(t, expand.stream)); got != prose || !strings.HasPrefix(prose, "[story turn 2]") {
		t.Errorf("streamed %q, whole %q: the transcript is the reply", got, prose)
	}
	if chunk := receive(t, audio, "spoken audio"); len(chunk) == 0 {
		t.Error("the first chunk was empty")
	}
	if spent := receive(t, usage, "usage"); spent.AudioOutputTokens == 0 || spent.Model != "mock-realtime" {
		t.Errorf("usage = %+v, want audio billed on the speaking model", spent)
	}
	if state := thread.ExportState(); !strings.Contains(state, `"reply":"`+prose+`"`) {
		t.Errorf("the thread kept %s, want the transcript", state)
	}

	expand.Spoken().SetActive(false)
	outline.TextInPort() <- "plot"
	receive(t, outline.whole, "the outline")
	for len(audio) > 0 {
		<-audio
	}
	expand.TextInPort() <- "NARRATE"
	receive(t, expand.whole, "the silent prose")
	select {
	case chunk := <-audio:
		t.Errorf("spoken off, the call still spoke %d bytes", len(chunk))
	case <-time.After(20 * time.Millisecond):
	}
}

// The session switches the thread's model when its calls go spoken: the next
// call opens a fresh provider thread on the other model, seeded with the
// transcript, so the story goes on where it was.
func TestThreadSwitchesModelMidGame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var slot adapters.ThreadSlot
	slot.Set(mock.Thread{}, "offer")
	thread := newThread(&slot)
	// A continuation that counts apart from its transcript, so a continued
	// thread and a reseeded one say different turns.
	thread.RestoreState(`{"continuation":{"id":"mock-thread-7","offer":"offer","model":"mock-story"},` +
		`"transcript":[{"input":"a","reply":"b"},{"input":"c","reply":"d"}]}`)
	expand := startThreaded(t, thread, "expand")
	expand.Start(ctx)

	expand.TextInPort() <- "one"
	if got := receive(t, expand.whole, "the reply on mock-story"); !strings.HasPrefix(got, "[story turn 8]") {
		t.Fatalf("said %q, want the stored thread continued", got)
	}

	slot.Set(mock.Thread{Name: "mock-realtime"}, "offer")
	expand.Spoken().SetActive(true)
	expand.TextInPort() <- "two"
	if got := receive(t, expand.whole, "the reply on mock-realtime"); !strings.HasPrefix(got, "[story turn 4]") {
		t.Errorf("said %q, want a fresh thread seeded with the three exchanges", got)
	}
	state := thread.ExportState()
	// The mock marks the spoken reply its thread holds after the turn.
	if !strings.Contains(state, `"id":"mock-thread-4:1","offer":"offer","model":"mock-realtime"`) {
		t.Errorf("exported state %s, want a continuation made on mock-realtime", state)
	}
}

// A failure that will not pass is not made again: the text the block was
// given goes out on the failure output, naming a rejected key, the thread's
// turn is handed on, and the block is ready for the next.
func TestThreadedFailsOnAHardError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{hard: adapters.ErrKeyRejected}
	b := startThreaded(t, newThread(adapter), "expand")
	b.Start(ctx)

	b.TextInPort() <- "the outline"
	if got := receive(t, b.failure, "the failed text"); got.Input != "the outline" {
		t.Errorf("failure output carried %q, want the text the block was given", got)
	}
	failed := expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	if failed.Reason != ports.ReasonKeyRejected {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonKeyRejected)
	}

	// The turn came back round, so the next call is made.
	adapter.heal()
	b.TextInPort() <- "the next outline"
	if got := receive(t, b.whole, "the next reply"); got != "whole reply" {
		t.Errorf("after a failure, said %q", got)
	}
}

// Retries exhausted: the provider is unreachable.
func TestThreadedGivesUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{breaks: 10}
	b := startThreaded(t, newThread(adapter), "expand")
	b.Start(ctx)

	b.TextInPort() <- "go"
	receive(t, b.failure, "the failed text")
	failed := expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	if failed.Reason != ports.ReasonUnreachable {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonUnreachable)
	}
}

// The thread sends its System Message with every turn, for the adapter to
// open a provider thread with; the block's message is the game's own; and the
// constraints come with every call of a block at frequency 1, with every
// third at 3, counted per block from the transcript.
func TestConstraintFrequency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{}
	b := startThreaded(t, newThread(adapter), "expand")
	b.SetConstraintFrequency(3)
	b.Start(ctx)
	for i := range 4 {
		b.TextInPort() <- "go"
		receive(t, b.whole, fmt.Sprintf("reply %d", i))
	}

	var carried []bool
	for _, turn := range adapter.turns() {
		if turn.System != "the system message" || !turn.Directive {
			t.Errorf("turn %+v, want the System Message and the game's own message", turn)
		}
		carried = append(carried, turn.Constraints == "NARRATION RULES must be respected: be kind")
	}
	if fmt.Sprint(carried) != "[true false false true]" {
		t.Errorf("constraints carried %v, want on the first call and every third", carried)
	}
}

// A reply tidied after it streamed is replaced on the stream before it
// closes, and the whole reply is the tidied one.
func TestThreadedReplacesATidiedReply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := startThreaded(t, newThread(&scriptedThread{}), "expand")
	b.Clean = func(s string) string { return strings.ToUpper(s) }
	b.Start(ctx)

	b.TextInPort() <- "go"
	pieces := reply(t, b.stream)
	want := []ports.Piece{"whole ", "reply", ports.Replacement("WHOLE REPLY"), ports.PieceClose}
	if fmt.Sprint(pieces) != fmt.Sprint(want) {
		t.Errorf("streamed %q, want %q", pieces, want)
	}
	if got := receive(t, b.whole, "the reply"); got != "WHOLE REPLY" {
		t.Errorf("whole reply %q, want the tidied one", got)
	}
}

// A reply with nothing left once tidied is discarded and asked again; after
// three, the call fails.
func TestThreadedAsksAgainWhenNothingIsLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{}
	b := startThreaded(t, newThread(adapter), "expand")
	b.Clean = func(string) string { return "" }
	b.Start(ctx)

	b.TextInPort() <- "the outline"
	if got := receive(t, b.failure, "the failed text"); got.Input != "the outline" {
		t.Errorf("failure output carried %q", got)
	}
	failed := expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	if failed.Reason != ports.ReasonBadReply {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonBadReply)
	}
	if n := len(adapter.turns()); n != 3 {
		t.Errorf("%d calls, want 3", n)
	}
	var pieces []ports.Piece
	for range 9 {
		pieces = append(pieces, receive(t, b.stream, "a piece"))
	}
	attempt := fmt.Sprint([]ports.Piece{"whole ", "reply", ports.PieceDiscard})
	if fmt.Sprint(pieces[0:3]) != attempt || fmt.Sprint(pieces[3:6]) != attempt || fmt.Sprint(pieces[6:9]) != attempt {
		t.Errorf("streamed %q, want each attempt discarded", pieces)
	}
}

// scriptedThread answers "whole reply", after breaking partway the first
// breaks times, or fails with hard until healed.
type scriptedThread struct {
	mu     sync.Mutex
	breaks int
	hard   error
	calls  []adapters.ThreadTurn
}

func (s *scriptedThread) Model() string { return "fake-story" }

func (s *scriptedThread) Continue(_ context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	s.mu.Lock()
	s.calls = append(s.calls, turn)
	breaking := s.breaks > 0
	if breaking {
		s.breaks--
	}
	hard := s.hard
	s.mu.Unlock()

	if hard != nil {
		return adapters.ThreadReply{}, hard
	}
	if breaking {
		turn.Hear("half ")
		if turn.Speak {
			turn.HearAudio([]byte{1, 0})
		}
		return adapters.ThreadReply{}, transient()
	}
	turn.Hear("whole ")
	if turn.Speak {
		turn.HearAudio([]byte{2, 0})
	}
	turn.Hear("reply")
	return adapters.ThreadReply{
		Text:         "whole reply",
		Usage:        adapters.Usage{Model: "fake-story", InputTokens: 10, OutputTokens: 2},
		Continuation: adapters.Continuation{ID: "resp_next", Model: "fake-story"},
	}, nil
}

func (s *scriptedThread) heal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hard = nil
}

func (s *scriptedThread) turns() []adapters.ThreadTurn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]adapters.ThreadTurn(nil), s.calls...)
}

func (s *scriptedThread) inputs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, c := range s.calls {
		out = append(out, c.Input)
	}
	return out
}

func (s *scriptedThread) continuations() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, c := range s.calls {
		out = append(out, c.Continuation.ID)
	}
	return out
}
