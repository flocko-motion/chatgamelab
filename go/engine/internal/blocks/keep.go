package blocks

import (
	"context"
	"sync"

	"engine/internal/ports"
)

// Keep asks once for a value and keeps it: it sends its request on its text
// output, keeps the answer that comes back on its text input, and hands it on
// as a prop under its key. A resumed session hands on what it kept without
// asking; one stored before the answer came asks again. It reports working
// until it holds the value, so a gate wired to it waits for it.
//
// An empty request asks nothing: the keep is ready at once and hands on
// nothing.
type Keep struct {
	name, key, request string

	in    ports.TextInput
	ask   ports.TextBroadcast
	out   ports.PropsBroadcast
	state ports.StateBroadcast

	// Guarded because ExportState is called from the caller's goroutine while
	// the block's own loop is running.
	mu    sync.Mutex
	value *string
}

func NewKeep(name, key, request string) *Keep {
	return &Keep{name: name, key: key, request: request, in: make(ports.TextInput, 1)}
}

func (b *Keep) NodeName() string                   { return b.name }
func (b *Keep) TextInPort() chan<- string          { return b.in }
func (b *Keep) TextOutPort() <-chan string         { return b.ask.Subscribe() }
func (b *Keep) PropsOutPort() <-chan ports.PropMap { return b.out.Subscribe() }

// StateOutPort reports working until the value is held.
func (b *Keep) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// Origin marks the keep as starting its work itself: the answer flowing back
// into it is its own request's.
func (b *Keep) Origin() {}

// ExportState is the value, once held.
func (b *Keep) ExportState() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.value == nil {
		return ""
	}
	return *b.value
}

func (b *Keep) RestoreState(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.value = &s
}

func (b *Keep) Start(ctx context.Context) {
	go func() {
		defer func() { b.ask.Close(); b.out.Close(); b.state.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		// Reported whatever there is to do: a gating block owes the gate a
		// working-then-ready cycle.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})

		b.mu.Lock()
		held := b.value
		b.mu.Unlock()
		switch {
		case held != nil:
			b.out.Send(ports.PropMap{b.key: *held})
		case b.request != "":
			b.ask.Send(b.request)
			select {
			case <-ctx.Done():
				return
			case value, open := <-b.in:
				if !open {
					return
				}
				b.mu.Lock()
				b.value = &value
				b.mu.Unlock()
				b.out.Send(ports.PropMap{b.key: value})
			}
		}
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		// The value is kept; a later answer is drained so its sender never waits.
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-b.in:
				if !open {
					return
				}
			}
		}
	}()
}
