package blocks

import (
	"context"
	"sync"

	"engine/internal/ports"
)

// firedMarker is what a fired source exports. The engine never interprets a
// block's state, so the value only has to be recognisable to this block.
const firedMarker = "fired"

// OnceText emits one configured line at startup and then closes. It is the init
// stem in its smallest form: whatever hangs off it runs exactly once per
// session, which is how a genre says "generate this portrait once" with an edge
// instead of with a flag inside the block doing the work.
type OnceText struct {
	name string
	text string

	out   ports.TextBroadcast
	state ports.StateBroadcast

	// Guarded because ExportState is called from the caller's goroutine while
	// the block's own loop is running — a checkpoint never stops the graph.
	mu    sync.Mutex
	fired bool
}

func NewOnceText(name, text string) *OnceText {
	return &OnceText{name: name, text: text}
}

func (b *OnceText) NodeName() string           { return b.name }
func (b *OnceText) TextOutPort() <-chan string { return b.out.Subscribe() }

// StateOutPort reports the one piece of work this block has. A genre that must
// not start play before the stem has run wires this to the gate.
func (b *OnceText) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// ExportState records that the line has gone out, not the line itself: the text
// is configuration and arrives again with the spec.
func (b *OnceText) ExportState() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fired {
		return firedMarker
	}
	return ""
}

func (b *OnceText) RestoreState(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fired = s == firedMarker
}

func (b *OnceText) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		b.mu.Lock()
		already := b.fired
		b.mu.Unlock()

		// Reported even on a resumed session, where there is nothing left to
		// send: a gating block owes the gate a working-then-ready cycle, or the
		// game never starts.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})

		// A resumed session stays silent, because whatever this line triggered
		// the first time has already been paid for and persisted. Firing again
		// would spend money to produce a different result and overwrite the one
		// the player was shown.
		if !already {
			select {
			case <-ctx.Done():
				return
			default:
			}
			b.out.Send(b.text)
			b.mu.Lock()
			b.fired = true
			b.mu.Unlock()
		}

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}()
}
