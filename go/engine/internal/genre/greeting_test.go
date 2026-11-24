package genre

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
)

func nextCall(t *testing.T, c *scriptedConn) string {
	t.Helper()
	select {
	case call := <-c.calls:
		return call
	case <-time.After(3 * time.Second):
		t.Fatal("the conversation was told nothing more")
	}
	return ""
}

// greetingTool writes a fixed line, or fails, and keeps what it was asked.
type greetingTool struct {
	asked chan string
	err   error
}

func (g *greetingTool) Query(_ context.Context, system, user string) (string, adapters.Usage, error) {
	g.asked <- system + "\n---\n" + user
	if g.err != nil {
		return "", adapters.Usage{}, g.err
	}
	return "Guten Tag! Setzen Sie sich doch.", adapters.Usage{}, nil
}

func greetingWiring(t *testing.T, live *scriptedLive, tool adapters.Tool) *Wiring {
	t.Helper()
	w, err := NewNPCLive(NPCLiveConfig{
		Live:         live,
		Tool:         tool,
		GalleryImage: mock.GalleryImage("office"),
		Scenario:     "You are a trainer interviewing an applicant.",
		GameStart:    "The trainer welcomes the applicant into her office.",
		InitPrompt:   "Begin.",
		Language:     "de",
		IdleAfter:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// The character speaks first: its first words, written before the game opens
// from how the game begins, are said once the conversation starts, after the
// opening instruction.
func TestTheCharacterSaysItsFirstWords(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	tool := &greetingTool{asked: make(chan string, 1)}
	w := greetingWiring(t, live, tool)
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)

	asked := <-tool.asked
	for _, want := range []string{"how the game begins", "The trainer welcomes the applicant", "Deutsch"} {
		if !strings.Contains(strings.ToLower(asked), strings.ToLower(want)) {
			t.Errorf("the first words were asked for without %q:\n%s", want, asked)
		}
	}

	nextInvitation(t, invites)
	connect(t, ctx, w)
	conn := nextConversation(t, live)
	conn.events <- adapters.LiveEvent{Kind: adapters.EventStarted}

	if got := nextCall(t, conn); got != "instruct: Begin." {
		t.Errorf("first the conversation was told %q, want the opening instruction", got)
	}
	if got := nextCall(t, conn); got != "say: Guten Tag! Setzen Sie sich doch." {
		t.Errorf("then it was told %q, want the first words to say", got)
	}
}

// Without its first words, as when the call fails, the game starts all the
// same, on the opening instruction alone, and the player hears of no error.
func TestAGameStartsWithoutItsFirstWords(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := newScriptedLive()
	tool := &greetingTool{asked: make(chan string, 8), err: errors.New("no answer")}
	w := greetingWiring(t, live, tool)
	errorsSeen := streamOf(w, "error")
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)

	nextInvitation(t, invites)
	connect(t, ctx, w)
	conn := nextConversation(t, live)
	conn.events <- adapters.LiveEvent{Kind: adapters.EventStarted}
	if got := nextCall(t, conn); got != "instruct: Begin." {
		t.Errorf("the conversation was told %q, want the opening instruction", got)
	}
	select {
	case call := <-conn.calls:
		t.Errorf("something was said without first words: %q", call)
	case e := <-errorsSeen:
		t.Errorf("the player was told of the failed first words: %q", e)
	case <-time.After(200 * time.Millisecond):
	}
}
