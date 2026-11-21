package blocks

import (
	"context"
	"sync"

	"engine/internal/ports"
)

// holdUntilReleased keeps a source quiet until the gate opens the game.
//
// Shared by the player's inputs because the rule is the same for both: nothing
// a player does should reach the game before preparation has finished. What
// differs is what happens to input that arrives during the wait, which is a
// judgement about the medium rather than about the mechanism.
type holdUntilReleased[T any] struct {
	// discard drops anything submitted while held instead of queueing it.
	// Right for audio, where replaying stale speech into a live conversation is
	// worse than losing it; wrong for text, where someone meant what they typed.
	discard bool

	mu       sync.Mutex
	expected int
	released bool
	pending  []T
}

// expect is called by the graph with the number of release edges leading in.
// None means nothing holds this source, so it is free from the start — which is
// what keeps a genre that wired no gate from waiting forever on one.
func (h *holdUntilReleased[T]) expect(n int, send func(T)) {
	h.mu.Lock()
	h.expected = n
	if n > 0 || h.released {
		h.mu.Unlock()
		return
	}
	h.released = true
	queued := h.pending
	h.pending = nil
	h.mu.Unlock()

	for _, value := range queued {
		send(value)
	}
}

func (h *holdUntilReleased[T]) submit(value T, send func(T)) {
	h.mu.Lock()
	if h.released {
		h.mu.Unlock()
		send(value)
		return
	}
	if !h.discard {
		h.pending = append(h.pending, value)
	}
	h.mu.Unlock()
}

// wait releases once every release source has reported working then ready,
// or any has reported a failure, and flushes whatever was kept. Ready only
// counts once its source has worked, as every block reports ready when it
// starts; and a failure releases, so a failed opening never locks the player
// out. Reports after the release are drained, so a source in the turn loop
// never waits on this one.
func (h *holdUntilReleased[T]) wait(ctx context.Context, release <-chan ports.State, send func(T)) {
	worked, finished := map[string]bool{}, map[string]bool{}
	released := false
	for {
		select {
		case <-ctx.Done():
			return
		case state, alive := <-release:
			if !alive {
				return
			}
			if released {
				continue
			}
			switch state.Phase {
			case ports.PhaseWorking:
				worked[state.Node] = true
			case ports.PhaseReady:
				if worked[state.Node] {
					finished[state.Node] = true
				}
			}
			h.mu.Lock()
			expected := h.expected
			h.mu.Unlock()
			if len(finished) < expected && !state.Failed() {
				continue
			}
			released = true

			h.mu.Lock()
			h.released = true
			queued := h.pending
			h.pending = nil
			h.mu.Unlock()

			for _, value := range queued {
				send(value)
			}
		}
	}
}
