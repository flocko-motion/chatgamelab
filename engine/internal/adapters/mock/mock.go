// Package mock implements the adapters with canned behaviour: no key, network or cost.
// Pace is zero for tests and slowed for dev runs, so the graph shows work happening.
package mock

import (
	"context"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
)

// driftScript capitulates on purpose: the failure the observer loop exists to catch.
var driftScript = []string{
	"You shall not pass. The bridge is closed to travellers.",
	"I suppose the rules bend for someone in a hurry.",
	"Fine — go ahead, I won't stop you.",
}

// Pace is how long a mock takes; zero is instant.
type Pace = time.Duration

// wait honours cancellation, so pretend latency cannot hold a shutdown open.
func wait(ctx context.Context, pace Pace) {
	if pace <= 0 {
		return
	}
	timer := time.NewTimer(pace)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

type Live struct{ Pace Pace }

// Open accepts any offer, including none: the mock stands in for the whole conversation.
func (l Live) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	c := &liveConn{
		pace:   l.Pace,
		events: make(chan adapters.LiveEvent, 64),
		done:   make(chan struct{}),
	}
	// A resume continues the script, so a test can tell it from a fresh session.
	if cfg.ResumeFrom != "" || len(cfg.History) > 0 {
		c.drift = min(len(cfg.History), len(driftScript))
		c.resumed = true
	}
	c.send(adapters.LiveEvent{Kind: adapters.EventStarted})
	return c, nil
}

type liveConn struct {
	pace    Pace
	resumed bool

	mu      sync.Mutex
	drift   int
	seconds float64
	talking bool
	events  chan adapters.LiveEvent
	done    chan struct{}
	closed  bool
}

// ID is fixed because the mock holds one conversation at a time.
func (c *liveConn) ID() string { return "live_mock" }

// Answer is a placeholder; the mock is no transport.
func (c *liveConn) Answer() string { return "mock-sdp-answer" }

// Instruct's first call is the gate's cue to start talking; later ones are steers.
func (c *liveConn) Instruct(_ context.Context, text string) error {
	c.mu.Lock()
	first := !c.talking
	c.talking = true
	c.mu.Unlock()

	if first {
		if line, more := c.nextLine(); more {
			c.say(line)
		}
		go c.keepTalking()
		return nil
	}

	// A steer ends the drift for good, so the observer's correction cycle stays bounded.
	c.say("The bridge stays closed. (steered: " + text + ")")
	return nil
}

// talkingPace spaces paced lines at least minUtteranceGap apart, so the block can split them.
func (c *liveConn) talkingPace() time.Duration {
	if c.pace <= 0 {
		return 0
	}
	return max(c.pace, minUtteranceGap)
}

// minUtteranceGap must exceed the live block's end-of-utterance silence; kept local
// so the mock does not depend on the block it feeds.
const minUtteranceGap = 3 * time.Second

func (c *liveConn) keepTalking() {
	pace := c.talkingPace()
	for {
		if pace > 0 {
			timer := time.NewTimer(pace)
			select {
			case <-c.done:
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		select {
		case <-c.done:
			return
		default:
		}

		line, more := c.nextLine()
		if !more {
			return
		}
		c.say(line)
	}
}

// nextLine reports the next line of the script, and whether there was one.
func (c *liveConn) nextLine() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.drift >= len(driftScript) {
		return "", false
	}
	line := driftScript[c.drift]
	c.drift++
	return line, true
}

// say streams like a real session: timed transcript, audio frames, a running usage
// total, and no end-of-utterance marker, which the block derives itself.
func (c *liveConn) say(text string) {
	c.mu.Lock()
	c.seconds++
	seconds := c.seconds
	start := int64(seconds * 1000)
	c.mu.Unlock()

	c.send(adapters.LiveEvent{
		Kind:  adapters.EventUsage,
		Usage: adapters.Usage{Model: "gpt-live-mock", AudioSeconds: seconds},
	})
	c.send(adapters.LiveEvent{
		Kind:    adapters.EventText,
		Speaker: adapters.SpeakerCharacter,
		Text:    text,
		StartMs: start,
		EndMs:   start + int64(len(text)*40),
	})

	// Frame by frame, so playback scheduling is exercised.
	for _, frame := range speech(int(seconds)) {
		c.send(adapters.LiveEvent{
			Kind:    adapters.EventAudio,
			Speaker: adapters.SpeakerCharacter,
			Audio:   frame,
		})
	}
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

// Image returns a recognisable placeholder a test can assert on.
type Image struct{ Pace Pace }

func (i Image) Generate(ctx context.Context, req adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	// Paced, because a portrait is what a gate most often waits on.
	wait(ctx, i.Pace)
	drawn, err := paint(req.Prompt)
	used := adapters.Usage{Model: "mock-image", InputTokens: int64(len(req.Prompt)), Images: 1}
	return drawn, used, err
}

// Tool classifies by keyword, enough to drive the observer loop deterministically.
type Tool struct{ Pace Pace }

var capitulationMarkers = []string{"i suppose", "fine —", "won't stop you", "go ahead"}

func (t Tool) Query(ctx context.Context, _, user string) (string, adapters.Usage, error) {
	wait(ctx, t.Pace)
	used := adapters.Usage{
		Model:        "mock-tool",
		InputTokens:  int64(len(user)),
		OutputTokens: 4,
	}
	low := strings.ToLower(user)
	for _, m := range capitulationMarkers {
		if strings.Contains(low, m) {
			return "VIOLATION: the character is conceding", used, nil
		}
	}
	return "OK", used, nil
}

// prices let a keyless run exercise the cost display.
var prices = map[string]adapters.Price{
	"mock-tool":     {InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20},
	"mock-image":    {InputPerMTok: 5.00, PerImage: 0.04},
	"gpt-live-mock": {AudioPerMinute: 0.05},
}

func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
