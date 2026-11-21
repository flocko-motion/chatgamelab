package blocks_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// threadedBlock is one started block and what it puts out.
type threadedBlock struct {
	*blocks.Threaded
	stream <-chan ports.Piece
	whole  <-chan string
	states <-chan ports.State
}

func startThreaded(t *testing.T, thread *blocks.Thread, name string) threadedBlock {
	t.Helper()
	b := blocks.NewThreaded(name, thread, "act as "+name)
	b.Backoff = []time.Duration{0, 0}
	tb := threadedBlock{Threaded: b, stream: b.StreamOutPort(), whole: b.TextOutPort(), states: b.StateOutPort()}
	return tb
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

// Outline and Expand are two blocks on one thread: Expand's call continues
// the conversation Outline's call left, and the thread's state carries both.
func TestTwoThreadedBlocksShareOneThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := blocks.NewThread("story-thread", mock.Thread{})
	outline := startThreaded(t, thread, "outline")
	expand := startThreaded(t, thread, "expand")
	outline.Start(ctx)
	expand.Start(ctx)

	outline.TextInPort() <- "The player crosses."
	plot := receive(t, outline.whole, "the outline")
	if plot != "[story turn 1] The player crosses." {
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
	for _, want := range []string{`"id":"mock-thread-2"`, `"model":"mock-story"`, `"input":"The player crosses."`} {
		if !strings.Contains(state, want) {
			t.Errorf("exported state %s lacks %s", state, want)
		}
	}
}

// A restored thread continues where the stored one left off.
func TestRestoredThreadContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := blocks.NewThread("story-thread", mock.Thread{})
	b := startThreaded(t, first, "expand")
	b.Start(ctx)
	b.TextInPort() <- "one"
	receive(t, b.whole, "the first reply")
	stored := first.ExportState()
	cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resumed := blocks.NewThread("story-thread", mock.Thread{})
	resumed.RestoreState(stored)
	again := startThreaded(t, resumed, "expand")
	again.Start(ctx2)
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
	thread := blocks.NewThread("story-thread", adapter)
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

	if got := adapter.inputs(); strings.Join(got, ",") != "turn 1,expand 1,turn 2" {
		t.Errorf("calls ran as %v, want in turn", got)
	}
}

// A stream that breaks partway is made again from the same continuation, and
// the pieces already sent are dropped first.
func TestThreadedRetriesABrokenStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{breaks: 1}
	thread := blocks.NewThread("story-thread", adapter)
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

// A failure that will not pass ends the game, naming a rejected key.
func TestThreadedFailsOnAHardError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{hard: adapters.ErrKeyRejected}
	b := startThreaded(t, blocks.NewThread("story-thread", adapter), "expand")
	b.Start(ctx)

	b.TextInPort() <- "go"
	failed := expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseFailed)
	if failed.Reason != ports.ReasonKeyRejected {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonKeyRejected)
	}
}

// Retries exhausted: the provider is unreachable.
func TestThreadedGivesUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedThread{breaks: 10}
	b := startThreaded(t, blocks.NewThread("story-thread", adapter), "expand")
	b.Start(ctx)

	b.TextInPort() <- "go"
	failed := expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseFailed)
	if failed.Reason != ports.ReasonUnreachable {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonUnreachable)
	}
}

// scriptedThread answers "whole reply", after breaking partway the first
// breaks times, or fails with hard.
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
	s.mu.Unlock()

	if s.hard != nil {
		return adapters.ThreadReply{}, s.hard
	}
	if breaking {
		turn.Hear("half ")
		return adapters.ThreadReply{}, transient()
	}
	turn.Hear("whole ")
	turn.Hear("reply")
	return adapters.ThreadReply{
		Text:         "whole reply",
		Usage:        adapters.Usage{Model: "fake-story", InputTokens: 10, OutputTokens: 2},
		Continuation: adapters.Continuation{ID: "resp_next", Model: "fake-story"},
	}, nil
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
