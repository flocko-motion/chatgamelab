package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"engine"
)

// The public surface has to be enough on its own: launch, drive, read events.
// If a test needs internal/, the boundary is in the wrong place.
func TestAdventureThroughPublicAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:   engine.GenreAdventure,
		ID:      "test-adventure",
		Persist: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Say("I try to cross the bridge"); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"text": false, "audio": false, "image": false, "props": false}
	deadline := time.After(3 * time.Second)
	for remaining(want) > 0 {
		select {
		case e := <-s.Events():
			want[e.Stream] = true
		case <-deadline:
			t.Fatalf("never saw every stream, still missing %d: %v", remaining(want), want)
		}
	}

	if got := len(store.Snapshot()); got < 4 {
		t.Errorf("persist saw %d events, want at least 4", got)
	}
}

// A live session takes both typed and spoken input, because the model does.
func TestLiveGenreAcceptsBothInputKinds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:     engine.GenreNPCLive,
		Guardrail: "refuse passage",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Speak([]byte("persuasion")); err != nil {
		t.Errorf("Speak should be accepted by npc-live: %v", err)
	}
	if err := s.Say("let me through"); err != nil {
		t.Errorf("Say should be accepted by npc-live: %v", err)
	}
}

// A genre that genuinely takes no input of a kind must say so rather than
// swallow it: input vanishing into a genre would surface as a mute character
// rather than as an error.
//
// Adventure will gain audio input once it wires a transcription block, which is
// what v1 does today. Until then it is the genre with one input kind.
func TestUnsupportedInputKindIsRejected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Speak([]byte("spoken at a typed genre")); err == nil {
		t.Error("expected Speak to be refused by adventure")
	}
	if err := s.Say("I try to cross the bridge"); err != nil {
		t.Errorf("Say should be accepted by adventure: %v", err)
	}
}

func TestUnknownGenreRefused(t *testing.T) {
	_, err := engine.Launch(context.Background(), engine.SessionSpec{Genre: "quiz"})
	if err == nil {
		t.Fatal("expected an error for an unknown genre")
	}
}

func remaining(m map[string]bool) int {
	n := 0
	for _, seen := range m {
		if !seen {
			n++
		}
	}
	return n
}

// A real platform without an injected key resolver must refuse to launch,
// rather than discovering the problem on the first model call.
func TestRealPlatformNeedsKeyFunc(t *testing.T) {
	_, err := engine.Launch(context.Background(), engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		Platform: engine.PlatformOpenAI,
	})
	if err == nil {
		t.Fatal("expected a launch error when no Keys function is injected")
	}
}

// The key must not be reachable from a serialised spec, because that blob is
// what gets persisted.
func TestSpecMarshalsWithoutSecrets(t *testing.T) {
	spec := engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		Platform: engine.PlatformOpenAI,
		Keys:     func(context.Context) (string, error) { return "sk-should-not-appear", nil },
	}
	blob, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "sk-should-not-appear") || strings.Contains(string(blob), "Keys") {
		t.Errorf("serialised spec leaked the key path: %s", blob)
	}
}

// The portrait is init work that nothing waits on, so a failure there costs the
// picture and not the conversation. What gates is a wiring decision, and this is
// the behaviour that decision buys.
func TestPortraitFailureDoesNotStopTheSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "no-portrait",
		Platform: engine.PlatformOpenAI,
		Scenario: "a bridge keeper",
		// A key resolver that fails stands in for any init work that cannot
		// complete: the image call never gets a key.
		Keys: func(context.Context) (string, error) {
			return "", errors.New("no key for you")
		},
	})
	if err != nil {
		t.Fatalf("a failed portrait must not stop the session: %v", err)
	}
	defer s.Close()

	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened, though nothing was wired to hold it")
	}
}

// The portrait arrives without anything triggering it, because preparation made
// it before play began.
func TestPortraitArrivesWithoutBeingTriggered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "portrait",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(3 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "image" {
				if !strings.Contains(e.Value, "keeper of a bridge") {
					t.Errorf("portrait was not made from the scenario: %s", e.Value)
				}
				return
			}
		case <-deadline:
			t.Fatal("no portrait, and nobody spoke to trigger one")
		}
	}
}

// Every block reports what it is doing on the session stream. This is the
// debugging surface: which block is busy, and for how long, is how a turn is
// read — and the same data a graph view draws.
func TestBlocksReportTheirPhases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "phases",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Speak([]byte("persuasion")); err != nil {
		t.Fatal(err)
	}

	working := map[string]bool{}
	deadline := time.After(3 * time.Second)
	for !working["observer"] {
		select {
		case e := <-s.Events():
			if e.Stream != "state" {
				continue
			}
			var report engine.BlockState
			if err := json.Unmarshal([]byte(e.Value), &report); err != nil {
				t.Fatalf("state event was not a block report: %q", e.Value)
			}
			if report.Phase == "working" {
				working[report.Node] = true
			}
		case <-deadline:
			t.Fatalf("no observer working report; saw %v", working)
		}
	}

	// The live session is busy for the whole conversation, which is what
	// distinguishes it on a graph view from a block that flickers per turn.
	if !working["live-session"] {
		t.Error("the live session never reported working")
	}
}

// The gate holds the player's inputs until preparation finishes, and reports
// that it has opened — which is what a client needs before offering a control
// that will actually work.
func TestGateOpeningIsVisibleOnTheStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "gated",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(3 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream != "state" {
				continue
			}
			var report engine.BlockState
			if err := json.Unmarshal([]byte(e.Value), &report); err != nil {
				t.Fatal(err)
			}
			if report.Node == "start-game" && report.Phase == "ready" {
				return
			}
		case <-deadline:
			t.Fatal("the gate never reported opening, so a client cannot know when to let anyone play")
		}
	}
}

// Words typed while the game is still preparing are kept and delivered when it
// starts, because someone who typed while a portrait rendered meant to say it.
func TestInputDuringPreparationIsNotLost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "early",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Say("hello?"); err != nil {
		t.Fatal(err)
	}

	// The character answers, which it can only do if the held line was
	// delivered once the gate opened.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "text" && strings.Contains(e.Value, "shall not pass") {
				return
			}
		case <-deadline:
			t.Fatal("nothing typed before the gate opened ever reached the game")
		}
	}
}
