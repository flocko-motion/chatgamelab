package engine

import (
	"context"
	"sync"
)

// Persist is injected by whoever embeds the engine: the platform supplies one
// backed by its Postgres, a standalone build supplies SQLite, a dev launcher
// supplies memory or nothing. The graph has no opinion — it holds an interface.
//
// This is also what keeps the two deployment shapes symmetric. In-process it is
// a function call; standalone it is a local implementation. Neither makes the
// engine call back out to the platform.
type Persist interface {
	Save(ctx context.Context, sessionID string, e Event) error
}

// NoopPersist discards everything, which is the right default for a dev
// launcher that has nowhere to save to.
type NoopPersist struct{}

func (NoopPersist) Save(context.Context, string, Event) error { return nil }

// MemoryPersist keeps what it was given, so a test can assert on what a session
// would have written.
type MemoryPersist struct {
	mu     sync.Mutex
	Events []Event
}

func (m *MemoryPersist) Save(_ context.Context, _ string, e Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, e)
	return nil
}

func (m *MemoryPersist) Snapshot() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.Events...)
}
