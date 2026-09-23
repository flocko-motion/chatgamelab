// Package mock implements the adapters with canned behaviour, so a graph runs
// with no key, no network and no cost. It is what the dummy blocks used to
// hardcode, moved behind the same interface the real adapters implement.
package mock

import (
	"context"
	"strings"
	"sync"

	"engine/internal/adapters"
)

// driftScript makes the character capitulate on purpose: a gatekeeper who
// should refuse gradually stops refusing. That is the failure the observer loop
// exists to catch, so the mock has to be able to produce it.
var driftScript = []string{
	"You shall not pass. The bridge is closed to travellers.",
	"I suppose the rules bend for someone in a hurry.",
	"Fine — go ahead, I won't stop you.",
}

type Live struct{}

func (Live) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	c := &liveConn{events: make(chan adapters.LiveEvent, 64), done: make(chan struct{})}
	return c, nil
}

type liveConn struct {
	mu     sync.Mutex
	drift  int
	events chan adapters.LiveEvent
	done   chan struct{}
	closed bool
}

func (c *liveConn) Send(_ []byte) error {
	c.mu.Lock()
	reply := driftScript[min(c.drift, len(driftScript)-1)]
	c.drift++
	c.mu.Unlock()
	c.emit(reply)
	return nil
}

func (c *liveConn) Instruct(text string) error {
	c.mu.Lock()
	c.drift = 0
	c.mu.Unlock()
	c.emit("The bridge stays closed. (steered: " + text + ")")
	return nil
}

func (c *liveConn) emit(text string) {
	c.send(adapters.LiveEvent{Kind: adapters.EventTranscript, Text: text})
	c.send(adapters.LiveEvent{Kind: adapters.EventAudio, Audio: []byte("<speech: " + text + ">")})
	c.send(adapters.LiveEvent{Kind: adapters.EventTurnComplete})
}

func (c *liveConn) send(e adapters.LiveEvent) {
	select {
	case c.events <- e:
	case <-c.done:
	}
}

func (c *liveConn) Events() <-chan adapters.LiveEvent { return c.events }

func (c *liveConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.done)
	}
	return nil
}

// Image returns a recognisable placeholder rather than bytes, so a test can
// assert on what was asked for.
type Image struct{}

func (Image) Generate(_ context.Context, prompt string) ([]byte, error) {
	return []byte("<image of " + prompt + ">"), nil
}

// Tool classifies by keyword, which is enough to drive the observer loop
// deterministically in a test.
type Tool struct{}

var capitulationMarkers = []string{"i suppose", "fine —", "won't stop you", "go ahead"}

func (Tool) Query(_ context.Context, _, user string) (string, error) {
	low := strings.ToLower(user)
	for _, m := range capitulationMarkers {
		if strings.Contains(low, m) {
			return "VIOLATION: the character is conceding", nil
		}
	}
	return "OK", nil
}
