package engine

import (
	"context"
	"fmt"
	"sync"
)

// SchemaVersion is the shape of what a session stores. A stored session of
// another version is refused on resume: sessions are disposable, and the
// player starts anew.
const SchemaVersion = 2

// Stored is what a session keeps beside its event log: the spec with the
// player's preference, never offers, and the state its blocks export.
type Stored struct {
	Version int          `json:"version"`
	Spec    SessionSpec  `json:"spec"`
	State   SessionState `json:"state"`
}

// Persist stores sessions; whoever embeds the engine supplies it. The engine
// saves a session's state at launch and at checkpoints while it runs, and each
// event of its history as it happens: text, pictures, status and errors. Live
// audio, narration, phases and usage reports are never events here; usage is
// part of the state.
//
// A store keeps what it is given and reacts to none of it. Offers are never
// stored: they are live, built from the platform's key cascade for every
// launch and resume, and carry the keys.
type Persist interface {
	SaveEvent(ctx context.Context, sessionID string, e SessionEvent) error
	SaveState(ctx context.Context, sessionID string, stored Stored) error
	// LoadState answers ErrUnknown for a session it does not hold.
	LoadState(ctx context.Context, sessionID string) (Stored, error)
	// LoadHistory is every event saved, oldest first.
	LoadHistory(ctx context.Context, sessionID string) ([]SessionEvent, error)
}

// NoopPersist discards everything, for a launcher with nowhere to save to.
type NoopPersist struct{}

func (NoopPersist) SaveEvent(context.Context, string, SessionEvent) error { return nil }
func (NoopPersist) SaveState(context.Context, string, Stored) error       { return nil }

func (NoopPersist) LoadState(_ context.Context, id string) (Stored, error) {
	return Stored{}, unknownSession(id)
}

func (NoopPersist) LoadHistory(context.Context, string) ([]SessionEvent, error) { return nil, nil }

// MemoryPersist keeps sessions in memory, for tests and a dev launcher: a
// session resumes from it within one process.
type MemoryPersist struct {
	mu       sync.Mutex
	sessions map[string]*memorySession
}

type memorySession struct {
	stored *Stored
	events []SessionEvent
}

func (m *MemoryPersist) session(id string) *memorySession {
	if m.sessions == nil {
		m.sessions = map[string]*memorySession{}
	}
	s, ok := m.sessions[id]
	if !ok {
		s = &memorySession{}
		m.sessions[id] = s
	}
	return s
}

func (m *MemoryPersist) SaveEvent(_ context.Context, id string, e SessionEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.session(id)
	s.events = append(s.events, e)
	return nil
}

func (m *MemoryPersist) SaveState(_ context.Context, id string, stored Stored) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.session(id).stored = &stored
	return nil
}

func (m *MemoryPersist) LoadState(_ context.Context, id string) (Stored, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok || s.stored == nil {
		return Stored{}, unknownSession(id)
	}
	return *s.stored, nil
}

func (m *MemoryPersist) LoadHistory(_ context.Context, id string) ([]SessionEvent, error) {
	return m.Events(id), nil
}

// Events is what a session has saved so far, for a test to assert on.
func (m *MemoryPersist) Events(id string) []SessionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil
	}
	return append([]SessionEvent(nil), s.events...)
}

func unknownSession(id string) error {
	return &Refusal{Problem: ProblemUnknown, Message: fmt.Sprintf("there is no session %q", id)}
}
