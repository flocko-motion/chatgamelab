package engine

import (
	"context"
	"sync"
)

// Persist is injected by whoever embeds the engine. The engine ships two: one
// that keeps events in memory, for tests, and one that discards them, for a dev
// launcher. A store persists a session's spec, its state and its events, never
// its offers: those are live, built from the platform's key cascade for every
// launch and resume, and carry the keys.
//
// The engine never calls back out to whoever stores the session; it hands
// events to this interface.
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
