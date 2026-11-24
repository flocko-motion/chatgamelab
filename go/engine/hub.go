package engine

import (
	"sync"
	"time"
)

// subscriberBuffer is how many events a subscriber may fall behind before it
// is cut off: generous, as narrated audio arrives in many small frames.
const subscriberBuffer = 4096

// hub hands every event of a session to each subscriber, each on a channel of
// its own, so two pages never take events from each other. It never waits: an
// event nobody subscribes to is only kept in the history, and a subscriber
// whose channel is full is cut off, its channel closed, rather than skipped
// over, as an event missed in the middle would garble what it shows. A page
// cut off reconnects and rebuilds from the history.
type hub struct {
	mu   sync.Mutex
	subs map[*subscriber]struct{}
	// first is the subscription opened with the session, for a caller in the
	// same process; the first socket's subscription ends it. Ended, it stays
	// readable, so what it holds still reaches whoever reads it.
	first *subscriber
	// alone is when the last socket left, or the session began; zero while
	// one is subscribed.
	alone  time.Time
	closed bool
}

type subscriber struct {
	ch chan SessionEvent
	// socket says the subscription is a page's, which keeps the session in memory.
	socket bool
}

func newHub(now time.Time) *hub {
	h := &hub{subs: map[*subscriber]struct{}{}, alone: now}
	h.first = &subscriber{ch: make(chan SessionEvent, subscriberBuffer)}
	h.subs[h.first] = struct{}{}
	return h
}

// subscribe opens a socket's subscription; cancel ends it, once.
func (h *hub) subscribe() (<-chan SessionEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub := &subscriber{ch: make(chan SessionEvent, subscriberBuffer), socket: true}
	if h.closed {
		close(sub.ch)
		return sub.ch, func() {}
	}
	h.dropLocked(h.first)
	h.subs[sub] = struct{}{}
	h.alone = time.Time{}
	return sub.ch, func() { h.unsubscribe(sub) }
}

func (h *hub) unsubscribe(sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dropLocked(sub)
}

// dropLocked ends one subscription, and notes when the last socket left.
func (h *hub) dropLocked(sub *subscriber) {
	if _, ok := h.subs[sub]; !ok {
		return
	}
	delete(h.subs, sub)
	close(sub.ch)
	if sub.socket && h.sockets() == 0 {
		h.alone = time.Now()
	}
}

func (h *hub) sockets() int {
	n := 0
	for sub := range h.subs {
		if sub.socket {
			n++
		}
	}
	return n
}

// publish hands an event to every subscriber, without waiting on any.
func (h *hub) publish(ev SessionEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs {
		select {
		case sub.ch <- ev:
		default:
			// Behind by a whole buffer: whoever reads it has gone, or
			// cannot keep up, and rebuilds once reconnected.
			h.dropLocked(sub)
		}
	}
}

// aloneSince is when the last socket left; false while one is subscribed.
func (h *hub) aloneSince() (time.Time, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.alone, !h.alone.IsZero()
}

// close ends every subscription, as the session ends.
func (h *hub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for sub := range h.subs {
		delete(h.subs, sub)
		close(sub.ch)
	}
}

// firstEvents is the subscription opened with the session.
func (h *hub) firstEvents() <-chan SessionEvent { return h.first.ch }
