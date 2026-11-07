// package: game / per-session serialization locks
// type:    logic
// job:     provides per-session mutexes so that AI calls on the same session are serialized and never run concurrently.
// limits:  does not perform any AI or database work; only guards ordering of calls made by session play.
package game

import (
	"sync"

	"github.com/google/uuid"
)

// sessionLocks provides per-session mutexes to serialize AI calls.
// Mistral's Conversations API does not support concurrent requests on the same
// conversation, and even for OpenAI the ExpandStory response ID must be
// persisted before the next ExecuteAction can use it for continuity.
//
// Usage: call Lock(sessionID) at the start of DoSessionAction and pass the
// returned unlock function into the ExpandStory goroutine so it can defer-unlock
// when it finishes.
var sessionLocks = &keyedMutex{locks: make(map[uuid.UUID]*lockEntry)}

type lockEntry struct {
	mu      sync.Mutex
	waiters int // number of goroutines waiting or holding the lock
}

type keyedMutex struct {
	mu    sync.Mutex // protects the map
	locks map[uuid.UUID]*lockEntry
}

// Lock acquires the per-session lock and returns an unlock function.
// The unlock function is safe to call exactly once.
func (km *keyedMutex) Lock(id uuid.UUID) func() {
	km.mu.Lock()
	entry, ok := km.locks[id]
	if !ok {
		entry = &lockEntry{}
		km.locks[id] = entry
	}
	entry.waiters++
	km.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		km.mu.Lock()
		entry.waiters--
		if entry.waiters == 0 {
			delete(km.locks, id)
		}
		km.mu.Unlock()
	}
}
