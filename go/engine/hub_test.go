package engine

import (
	"testing"
	"time"
)

func drained(ch <-chan SessionEvent) (got []string, closed bool) {
	for {
		select {
		case ev, open := <-ch:
			if !open {
				return got, true
			}
			got = append(got, ev.Value)
		default:
			return got, false
		}
	}
}

// Two pages on one session each get every event: neither takes events from
// the other, which is what a reload's old socket used to do.
func TestTwoSubscribersEachGetEveryEvent(t *testing.T) {
	h := newHub(time.Now())
	a, _ := h.subscribe()
	b, _ := h.subscribe()
	for _, v := range []string{"one", "two", "three"} {
		h.publish(SessionEvent{Stream: "text", Value: v})
	}
	for name, ch := range map[string]<-chan SessionEvent{"first page": a, "second page": b} {
		if got, _ := drained(ch); len(got) != 3 {
			t.Errorf("%s got %v, want all three events", name, got)
		}
	}
}

// Nobody subscribed: publishing never waits.
func TestPublishingWithoutSubscribersNeverWaits(t *testing.T) {
	h := newHub(time.Now())
	_, cancel := h.subscribe()
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 3 * subscriberBuffer {
			h.publish(SessionEvent{Stream: "text", Value: "x"})
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishing waited on nobody")
	}
}

// A subscriber a whole buffer behind is cut off, its channel closed, rather
// than skipped over; the others go on.
func TestASubscriberFallingBehindIsCutOff(t *testing.T) {
	h := newHub(time.Now())
	slow, _ := h.subscribe()
	fast, cancelFast := h.subscribe()
	defer cancelFast()

	for i := range subscriberBuffer + 1 {
		h.publish(SessionEvent{Stream: "text", Value: "x"})
		if i%100 == 0 {
			drained(fast)
		}
	}
	got, closed := drained(slow)
	if !closed {
		t.Fatal("the subscriber that fell behind was not cut off")
	}
	if len(got) != subscriberBuffer {
		t.Errorf("it kept %d events before the cut, want its whole buffer, %d, and nothing skipped", len(got), subscriberBuffer)
	}
	h.publish(SessionEvent{Stream: "text", Value: "after"})
	if got, closed := drained(fast); closed || len(got) == 0 {
		t.Errorf("the subscriber keeping up lost its events (%v, closed %t)", got, closed)
	}
}

// The subscription opened with the session sees events from the start; the
// first page's subscription ends it, so nobody keeps filling it.
func TestTheFirstSubscriptionEndsWithTheFirstPage(t *testing.T) {
	h := newHub(time.Now())
	h.publish(SessionEvent{Stream: "text", Value: "early"})
	first := h.firstEvents()
	if got, _ := drained(first); len(got) != 1 {
		t.Fatalf("the first subscription missed the early event: %v", got)
	}
	_, cancel := h.subscribe()
	defer cancel()
	if _, closed := drained(first); !closed {
		t.Error("the first subscription stayed open once a page subscribed")
	}
}

// A session knows since when no page has been subscribed, for whoever
// unloads sessions nobody is playing.
func TestAloneSinceTheLastPageLeft(t *testing.T) {
	started := time.Now().Add(-time.Hour)
	h := newHub(started)
	if since, alone := h.aloneSince(); !alone || !since.Equal(started) {
		t.Errorf("a new session is alone since %v (%t), want since it began", since, alone)
	}
	_, a := h.subscribe()
	_, b := h.subscribe()
	if _, alone := h.aloneSince(); alone {
		t.Error("alone while pages are subscribed")
	}
	a()
	if _, alone := h.aloneSince(); alone {
		t.Error("alone while one page is still subscribed")
	}
	b()
	b() // a second cancel is harmless
	if since, alone := h.aloneSince(); !alone || time.Since(since) > time.Second {
		t.Errorf("not alone, or not since now, once the last page left: %v %t", since, alone)
	}
}
