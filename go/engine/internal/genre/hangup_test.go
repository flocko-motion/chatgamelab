package genre

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// scriptedLive plays the provider: each conversation it opens is handed to
// the test, which says what happens in it.
type scriptedLive struct {
	opened chan *scriptedConn
	fail   chan error
}

func newScriptedLive() *scriptedLive {
	return &scriptedLive{opened: make(chan *scriptedConn, 8), fail: make(chan error, 8)}
}

func (l *scriptedLive) Open(_ context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	select {
	case err := <-l.fail:
		return nil, err
	default:
	}
	c := &scriptedConn{cfg: cfg, events: make(chan adapters.LiveEvent, 8), calls: make(chan string, 8)}
	l.opened <- c
	return c, nil
}

type scriptedConn struct {
	cfg    adapters.LiveConfig
	events chan adapters.LiveEvent
	// calls is what the conversation was told, in order: "instruct: …", "say: …".
	calls chan string
}

func (c *scriptedConn) ID() string     { return "live_scripted" }
func (c *scriptedConn) Answer() string { return "v=0 answer" }
func (c *scriptedConn) Instruct(_ context.Context, text string) error {
	c.calls <- "instruct: " + text
	return nil
}
func (c *scriptedConn) Inform(_ context.Context, text string) error {
	c.calls <- "inform: " + text
	return nil
}
func (c *scriptedConn) SayAloud(_ context.Context, text string) error {
	c.calls <- "say: " + text
	return nil
}
func (c *scriptedConn) Events() <-chan adapters.LiveEvent { return c.events }
func (c *scriptedConn) Close() error                      { return nil }

func (c *scriptedConn) hangUp(reason string) {
	c.events <- adapters.LiveEvent{Kind: adapters.EventClosed, CloseReason: reason}
}

func hangupWiring(t *testing.T, live *scriptedLive) *Wiring {
	t.Helper()
	w, err := NewNPCLive(NPCLiveConfig{
		Live:         live,
		GalleryImage: mock.GalleryImage("bridge"),
		Tool:         mock.Tool{},
		Scenario:     "You are the keeper of a bridge.",
		InitPrompt:   "Begin.",
		IdleAfter:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func nextInvitation(t *testing.T, invites <-chan ports.State) ports.State {
	t.Helper()
	select {
	case s := <-invites:
		return s
	case <-time.After(5 * time.Second):
		t.Fatal("no invitation came")
	}
	return ports.State{}
}

func nextConversation(t *testing.T, live *scriptedLive) *scriptedConn {
	t.Helper()
	select {
	case c := <-live.opened:
		return c
	case <-time.After(5 * time.Second):
		t.Fatal("no conversation was opened")
	}
	return nil
}

// A conversation the provider hangs up is not the end of the game: the page
// is invited to connect again at once, and the new conversation carries on
// from what was said.
func TestAHungUpConversationCarriesOnFromTheTranscript(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	w := hangupWiring(t, live)
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)

	if got := nextInvitation(t, invites); got.Reason != "" {
		t.Errorf("the first invitation waits for a press: %q", got.Reason)
	}
	connect(t, ctx, w)
	first := nextConversation(t, live)
	first.events <- adapters.LiveEvent{Kind: adapters.EventText, Speaker: adapters.SpeakerCharacter, Text: "You shall not pass."}
	first.hangUp("remote_hangup")

	if got := nextInvitation(t, invites); got.Reason != "" {
		t.Fatalf("after a hang-up the invitation waits for a press (%q); the page should connect at once", got.Reason)
	}
	if got := w.Live.Standing(); got != blocks.StandingAwaiting {
		t.Errorf("standing %q after a hang-up, want awaiting so a reloaded page connects", got)
	}
	connect(t, ctx, w)
	second := nextConversation(t, live)
	if len(second.cfg.History) == 0 || !strings.Contains(second.cfg.History[0].Text, "shall not pass") {
		t.Errorf("the new conversation does not carry on from the transcript: %+v", second.cfg.History)
	}
}

// A provider that keeps hanging up before anything is said is not called
// again and again: after a few, the conversation waits for the player.
func TestRepeatedHangUpsWaitForThePlayer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	w := hangupWiring(t, live)
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)
	nextInvitation(t, invites)

	for i := 0; ; i++ {
		connect(t, ctx, w)
		nextConversation(t, live).hangUp("connection_lost")
		got := nextInvitation(t, invites)
		if got.Reason == blocks.InvitePress {
			if i < 2 {
				t.Errorf("waited for the player after %d hang-ups, want a few tries first", i+1)
			}
			return
		}
		if i > 10 {
			t.Fatal("the conversation never stopped connecting again by itself")
		}
	}
}

// A conversation that cannot be opened leaves the game waiting for the
// player's press, not finished: the next press opens one.
func TestAFailedOpenWaitsForThePlayer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	w := hangupWiring(t, live)
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)
	nextInvitation(t, invites)

	live.fail <- adapters.ErrNoCredit
	if _, err := w.Live.Connect(ctx, "v=0 offer"); !errors.Is(err, adapters.ErrNoCredit) {
		t.Fatalf("connect: %v, want the open's error", err)
	}
	if got := nextInvitation(t, invites); got.Reason != blocks.InvitePress {
		t.Errorf("after a failed open the invitation is %q, want one that waits for a press", got.Reason)
	}
	connect(t, ctx, w)
	nextConversation(t, live)
}

// After a hang-up, what is put into the conversation reaches the new one, not
// the one the provider closed.
func TestInjectionsReachTheConversationAfterAHangUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	w := hangupWiring(t, live)
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)
	nextInvitation(t, invites)

	connect(t, ctx, w)
	first := nextConversation(t, live)
	first.hangUp("remote_hangup")
	nextInvitation(t, invites)
	connect(t, ctx, w)
	second := nextConversation(t, live)

	if err := w.Inject("observer", ports.InjectBackground, "She is from the 1950s."); err != nil {
		t.Fatal(err)
	}
	select {
	case call := <-second.calls:
		if call != "inform: She is from the 1950s." {
			t.Errorf("the new conversation was told %q", call)
		}
	case call := <-first.calls:
		t.Fatalf("the closed conversation was told %q", call)
	case <-time.After(3 * time.Second):
		t.Fatal("the injection reached no conversation")
	}
}
