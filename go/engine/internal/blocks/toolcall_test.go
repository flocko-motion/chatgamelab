package blocks_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// The block hands its prompt and the player's text to the tool, passes the
// reply on and bills what the call cost.
func TestToolCallPassesTheReplyOn(t *testing.T) {
	tool := &scriptedTool{reply: "The player tries to cross."}
	b, out, states, usage := startToolCall(t, tool)

	b.TextInPort() <- "I cross"

	if got := receive(t, out, "the reply"); got != "The player tries to cross." {
		t.Errorf("passed on %q", got)
	}
	if calls := tool.seen(); len(calls) != 1 || calls[0] != [2]string{"rephrase it", "I cross"} {
		t.Errorf("the tool was asked %v", calls)
	}
	used := receive(t, usage, "the usage")
	if used.Node != "rephrase" || used.Model != "fake-tool" || used.InputTokens != 7 || used.OutputTokens != 5 {
		t.Errorf("billed %+v", used)
	}
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// A failure that may pass is asked again, and the player never hears of it.
func TestToolCallRetriesTransientErrors(t *testing.T) {
	tool := &scriptedTool{reply: "fine", errs: []error{transient(), transient(), transient(), transient()}}
	b, out, states, _ := startToolCall(t, tool)

	b.TextInPort() <- "I cross"

	if got := receive(t, out, "the reply after four failures"); got != "fine" {
		t.Errorf("passed on %q", got)
	}
	if n := len(tool.seen()); n != 5 {
		t.Errorf("the tool was asked %d times, want 5", n)
	}
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// Five attempts and still failing: the text goes out on the failure output,
// the block is ready again and says the provider is unreachable.
func TestToolCallGivesUpAfterFiveAttempts(t *testing.T) {
	tool := &scriptedTool{reply: "never", errs: []error{transient(), transient(), transient(), transient(), transient(), transient()}}
	b, out, states, _ := startToolCall(t, tool)
	failure := b.FailureOutPort()

	b.TextInPort() <- "I cross"

	if got := receive(t, failure, "the failed text"); got.Input != "I cross" {
		t.Errorf("the failure output carried %q, want the text the block was given", got)
	}
	failed := expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	if failed.Reason != ports.ReasonUnreachable {
		t.Errorf("reason %q, want %q", failed.Reason, ports.ReasonUnreachable)
	}
	if n := len(tool.seen()); n != 5 {
		t.Errorf("the tool was asked %d times, want 5", n)
	}
	select {
	case v, open := <-out:
		if open {
			t.Errorf("a failed call passed on %q", v)
		}
	default:
	}
}

// A failure that will not pass is not asked again: the call fails at once,
// naming a rejected key apart from anything else, and the block goes on with
// the next text.
func TestToolCallFailsAtOnceOnHardErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		reason string
	}{
		"rejected key":  {fmt.Errorf("responses api: 401: %w", adapters.ErrKeyRejected), ports.ReasonKeyRejected},
		"unknown model": {errors.New("responses api: 404 model not found"), ports.ReasonFailed},
		"no platform":   {adapters.ErrUnset, ports.ReasonFailed},
	} {
		t.Run(name, func(t *testing.T) {
			tool := &scriptedTool{reply: "fine", errs: []error{tc.err}}
			b, out, states, _ := startToolCall(t, tool)
			failure := b.FailureOutPort()

			b.TextInPort() <- "I cross"

			receive(t, failure, "the failed text")
			failed := expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
			if failed.Reason != tc.reason {
				t.Errorf("reason %q, want %q", failed.Reason, tc.reason)
			}
			if failed.Message != tc.err.Error() {
				t.Errorf("message %q, want the error %q", failed.Message, tc.err.Error())
			}
			if n := len(tool.seen()); n != 1 {
				t.Errorf("the tool was asked %d times, want 1", n)
			}

			b.TextInPort() <- "I wait"
			if got := receive(t, out, "the next reply"); got != "fine" {
				t.Errorf("after a failure, passed on %q", got)
			}
		})
	}
}

func startToolCall(t *testing.T, tool adapters.Tool) (*blocks.ToolCall, <-chan string, <-chan ports.State, <-chan ports.Usage) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	b := blocks.NewToolCall("rephrase", tool, "rephrase it")
	b.Backoff = make([]time.Duration, len(b.Backoff))
	out, states, usage := b.TextOutPort(), b.StateOutPort(), b.UsageOutPort()
	b.Start(ctx)
	return b, out, states, usage
}

// expectPhases reads the block's states in order and returns the last.
func expectPhases(t *testing.T, states <-chan ports.State, want ...ports.Phase) ports.State {
	t.Helper()
	var last ports.State
	for _, phase := range want {
		last = receive(t, states, string(phase))
		if last.Phase != phase {
			t.Fatalf("phase %q, want %q", last.Phase, phase)
		}
	}
	return last
}

func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatalf("never received %s", what)
	}
	var zero T
	return zero
}

func transient() error { return fmt.Errorf("responses api: 503: %w", adapters.ErrTransient) }

// scriptedTool fails with errs in turn, then answers reply.
type scriptedTool struct {
	reply string
	errs  []error

	mu    sync.Mutex
	calls [][2]string
}

func (s *scriptedTool) Query(_ context.Context, system, user string) (string, adapters.Usage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, [2]string{system, user})
	if len(s.calls) <= len(s.errs) {
		return "", adapters.Usage{}, s.errs[len(s.calls)-1]
	}
	return s.reply, adapters.Usage{Model: "fake-tool", InputTokens: 7, OutputTokens: 5}, nil
}

func (s *scriptedTool) seen() [][2]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][2]string(nil), s.calls...)
}

// A kept call answers once and remembers: restored, it gives its reply again
// without asking the tool, and asked anew it answers the same.
func TestKeptToolCallAnswersFromMemoryAfterARestart(t *testing.T) {
	tool := &countingTool{}
	first := blocks.NewToolCall("greeting", tool, "write")
	first.Keep = true
	ctx, cancel := context.WithCancel(context.Background())
	out := first.TextOutPort()
	first.Start(ctx)
	first.TextInPort() <- "brief"
	reply := <-out
	cancel()
	saved := first.ExportState()
	if saved != reply || tool.calls != 1 {
		t.Fatalf("kept %q after %d calls, want the reply %q after 1", saved, tool.calls, reply)
	}

	again := blocks.NewToolCall("greeting", tool, "write")
	again.Keep = true
	again.RestoreState(saved)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	out2 := again.TextOutPort()
	again.Start(ctx2)
	if got := <-out2; got != reply {
		t.Errorf("restored, it gave %q, want %q", got, reply)
	}
	again.TextInPort() <- "brief"
	if got := <-out2; got != reply {
		t.Errorf("asked anew, it gave %q, want %q", got, reply)
	}
	if tool.calls != 1 {
		t.Errorf("the tool was asked %d times, want once", tool.calls)
	}
}

// Without Keep a call keeps nothing, so a resumed session answers only what
// it is sent anew.
func TestToolCallKeepsNothingUnlessAsked(t *testing.T) {
	b := blocks.NewToolCall("rephrase", &countingTool{}, "write")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := b.TextOutPort()
	b.Start(ctx)
	b.TextInPort() <- "an action"
	<-out
	if s := b.ExportState(); s != "" {
		t.Errorf("an unkept call exported %q", s)
	}
}

type countingTool struct {
	mu    sync.Mutex
	calls int
}

func (c *countingTool) Query(_ context.Context, _, user string) (string, adapters.Usage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return "reply to " + user, adapters.Usage{}, nil
}
