package blocks

import (
	"context"
	"errors"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// ToolCall is a single-shot text transformation: its role (rephrase,
// translate, condense, …) is the system prompt it is built with, and the text
// it receives is the user message.
//
// Its output is structural, so it has no switch. A failure that may pass is
// retried; one that will not, or an exhausted retry, sends the text on the
// failure output, and the block goes on with the next.
type ToolCall struct {
	failing
	name, system string
	tool         adapters.Tool

	// Backoff is the wait before each retry of a failure that may pass, so a
	// call is made len(Backoff)+1 times at most. Set before Start.
	Backoff []time.Duration

	in    ports.TextInput
	out   ports.TextBroadcast
	state ports.StateBroadcast
	meter meter
}

func NewToolCall(name string, tool adapters.Tool, system string) *ToolCall {
	// A missing adapter is a wiring mistake; it surfaces where the graph is built.
	if tool == nil {
		panic("blocks: " + name + " needs a tool adapter")
	}
	return &ToolCall{
		name:    name,
		system:  system,
		tool:    tool,
		Backoff: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
		in:      make(ports.TextInput, 16),
		meter:   meter{name: name},
	}
}

func (b *ToolCall) NodeName() string             { return b.name }
func (b *ToolCall) TextInPort() chan<- string    { return b.in }
func (b *ToolCall) TextOutPort() <-chan string   { return b.out.Subscribe() }
func (b *ToolCall) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

// UsageOutPort reports what the calls have cost.
func (b *ToolCall) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

// StateOutPort reports what this block is doing, with the provider's reason
// after a failed call.
func (b *ToolCall) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *ToolCall) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.failure.Close(); b.state.Close(); b.meter.usage.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		for {
			select {
			case <-ctx.Done():
				return
			case text, open := <-b.in:
				if !open {
					return
				}

				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				reply, err := b.query(ctx, text)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					b.fail(&b.state, b.name, text, err)
					continue
				}
				b.out.Send(reply)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// query asks the tool, retrying a failure that may pass after each backoff.
func (b *ToolCall) query(ctx context.Context, text string) (string, error) {
	for attempt := 0; ; attempt++ {
		reply, used, err := b.tool.Query(ctx, b.system, text)
		if err == nil {
			b.meter.bill(used)
			return reply, nil
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

// failing is a block's failure output: each call that failed for good, with
// the input it was given. The completeness check refuses it unwired.
type failing struct{ failure ports.FailureBroadcast }

// FailureOutPort carries each call that failed for good.
func (f *failing) FailureOutPort() <-chan ports.Failure { return f.failure.Subscribe() }

// RequiredOutputs has the failure output: what a failure leads to is the
// genre's to say.
func (f *failing) RequiredOutputs() []ports.Kind { return []ports.Kind{ports.KindFailure} }

// fail sends the failure, then reports the block ready again with the
// provider's reason.
func (f *failing) fail(state *ports.StateBroadcast, name, input string, err error) {
	why := reason(err)
	f.failure.Send(ports.Failure{Node: name, Input: input, Reason: why, Message: err.Error()})
	state.Send(ports.State{Node: name, Phase: ports.PhaseReady, Reason: why, Message: err.Error()})
}

// errExhausted is a failure that might have passed, still there after the
// last attempt: the provider is unreachable.
type errExhausted struct{ error }

func (e errExhausted) Unwrap() error { return e.error }

func reason(err error) string {
	var exhausted errExhausted
	switch {
	case errors.As(err, &exhausted):
		return ports.ReasonUnreachable
	case errors.Is(err, adapters.ErrBadReply):
		return ports.ReasonBadReply
	case errors.Is(err, adapters.ErrKeyRejected):
		return ports.ReasonKeyRejected
	}
	return ports.ReasonFailed
}

// sleep waits d, and reports false if the context ended first.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
