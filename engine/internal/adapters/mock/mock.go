// Package mock implements the adapters with canned behaviour, so a graph runs
// with no key, no network and no cost. It is what the dummy blocks used to
// hardcode, moved behind the same interface the real adapters implement.
//
// Every adapter here takes a Pace, because a mock has two jobs that want
// opposite speeds. A test wants answers instantly and gets them at the zero
// value. Somebody watching a dev run wants to see the pipeline work, and a
// block that finishes between two frames shows them nothing — a graph where
// every node is always ready is a graph that teaches nothing about what a turn
// actually costs in time.
package mock

import (
	"context"
	"strings"
	"sync"
	"time"

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

// Pace is how long a mock takes to do its work. Zero is instant, which is what
// a test wants; a dev run sets something a person can follow.
type Pace = time.Duration

// wait sleeps for the configured pace, or returns early if the caller gave up.
// A mock that ignored cancellation would hold a shutdown open for its own
// pretend latency.
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

// Open answers any offer, including none: the mock stands in for the whole
// conversation rather than for a transport, which is what lets a genre be
// exercised with no browser, no network and no key.
func (l Live) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	c := &liveConn{
		pace:   l.Pace,
		events: make(chan adapters.LiveEvent, 64),
		done:   make(chan struct{}),
	}
	// A resumed conversation carries on rather than starting over, so the mock
	// picks the script up where the last one left it. A test can then tell a
	// real resume from a fresh session by what the character says next.
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

// ID names the conversation, which is what a resume has to quote. It is fixed
// because the mock holds one at a time.
func (c *liveConn) ID() string { return "live_mock" }

// Answer is what the browser would apply to complete its connection. The mock
// is not a transport, so it says so rather than pretending to be one.
func (c *liveConn) Answer() string { return "mock-sdp-answer" }

// Instruct both opens the conversation and steers it, because appended
// instructions are the only thing that reaches a live session from this side.
// The first one is the gate's cue, which starts the character talking; every
// one after it is a correction, which pulls the drift back to the beginning.
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

	// A steer puts the character back in role and leaves it there. Rewinding the
	// script would make it drift again, and a mock that can be steered forever
	// would let the cycle the observer bounds run away in a dev run.
	c.say("The bridge stays closed. (steered: " + text + ")")
	return nil
}

// keepTalking walks the drift script once and then listens, which makes the
// character capitulate on purpose: a keeper who should hold the line gradually
// stops holding it. That is the failure the observer loop exists to catch.
//
// It stops at the end of the script rather than repeating its last line. A mock
// that talked forever would not be imitating a live model — one speaks and then
// waits — and it would bill a dev run for a conversation nobody is having.
// A character speaking at all needs to pause between lines, or the live block
// never sees the silence it uses to tell one utterance from the next and the
// whole conversation runs together. So a paced mock waits at least that long,
// whatever it was configured with.
func (c *liveConn) talkingPace() time.Duration {
	if c.pace <= 0 {
		return 0
	}
	return max(c.pace, minUtteranceGap)
}

// minUtteranceGap must exceed the silence the live block treats as the end of
// an utterance, or a paced mock runs every line into the one before it. Kept
// here rather than imported, because a mock reaching into the block it feeds
// would couple the two in the wrong direction.
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

// say streams one utterance the way a real session does: a transcript carrying
// its interval on the session timeline, audio frames, and a running usage
// total. Nothing marks the end — the block derives that, and a mock supplying a
// boundary would be testing something the API does not do.
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

	// Streamed frame by frame rather than as one blob, so playback scheduling
	// is exercised the way real audio will exercise it.
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

// Image returns a recognisable placeholder rather than bytes, so a test can
// assert on what was asked for.
type Image struct{ Pace Pace }

func (i Image) Generate(ctx context.Context, req adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	// A picture is the slowest thing in a session and the one a gate most often
	// waits on, so a mock that returned instantly would hide the wait a real
	// portrait makes a player sit through.
	wait(ctx, i.Pace)
	drawn, err := paint(req.Prompt)
	used := adapters.Usage{Model: "mock-image", InputTokens: int64(len(req.Prompt)), Images: 1}
	return drawn, used, err
}

// Tool classifies by keyword, which is enough to drive the observer loop
// deterministically in a test.
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

// Prices for the mock models, so a keyless run still exercises the cost display
// with numbers that look like money.
var prices = map[string]adapters.Price{
	"mock-tool":     {InputPerMTok: 0.20, CachedInputPerMTok: 0.02, OutputPerMTok: 1.20},
	"mock-image":    {InputPerMTok: 5.00, PerImage: 0.04},
	"gpt-live-mock": {AudioPerMinute: 0.05},
}

func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
