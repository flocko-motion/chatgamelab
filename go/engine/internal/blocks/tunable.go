package blocks

import "sync"

// tunable is what a block runs on that may change while it runs, as a person
// edits a prompt during play: the block reads it at each call, so the next
// call takes the new value and a call already made keeps the one it began
// with.
type tunable[T any] struct {
	mu sync.Mutex
	v  T
}

func newTunable[T any](v T) tunable[T] { return tunable[T]{v: v} }

func (t *tunable[T]) get() T {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.v
}

func (t *tunable[T]) set(v T) {
	t.mu.Lock()
	t.v = v
	t.mu.Unlock()
}
