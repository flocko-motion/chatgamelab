package engine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"engine"
)

// A resumed session continues its provider-side thread rather than starting a
// new one. Without this, every restart silently loses the conversation the
// model is holding.
func TestResumeContinuesTheThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Say("I try to cross the bridge"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, first, "text", "[prose turn 1]")

	state := first.State()
	first.Close()

	if len(state.Blocks) == 0 {
		t.Fatal("nothing was exported; a threaded block must carry state")
	}

	resumed, err := engine.Resume(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s1"}, state)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()

	if err := resumed.Say("I keep going"); err != nil {
		t.Fatal(err)
	}
	// Turn 2, not turn 1: the thread continued.
	waitFor(t, resumed, "text", "[prose turn 2]")
}

// A fresh launch with the same spec starts over, which is what makes the test
// above meaningful.
func TestFreshLaunchDoesNotContinue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s2"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	s.Say("I try to cross the bridge")
	waitFor(t, s, "text", "[prose turn 1]")
}

// State naming a block the wiring no longer has means the wiring changed under
// a stored session. That has to fail rather than half-restore.
func TestResumeRefusesUnknownBlock(t *testing.T) {
	_, err := engine.Resume(context.Background(),
		engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s3"},
		engine.SessionState{Blocks: map[string]string{"a-block-that-left": "resp-9"}})
	if err == nil {
		t.Fatal("expected a resume against a changed wiring to fail")
	}
	if !strings.Contains(err.Error(), "a-block-that-left") {
		t.Errorf("error should name the missing block, got: %v", err)
	}
}

func waitFor(t *testing.T, s *engine.Session, stream, prefix string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == stream && strings.HasPrefix(e.Value, prefix) {
				return
			}
		case <-deadline:
			t.Fatalf("never saw %s event starting %q", stream, prefix)
		}
	}
}

// Status values are not a handle to something a provider remembers: if the
// engine loses them, they are gone, and the player's status bar resets. So they
// have to survive a resume too.
func TestResumeRestoresStatusValues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spec := engine.SessionSpec{
		Genre:  engine.GenreAdventure,
		ID:     "s4",
		Status: map[string]string{"Health": "Wounded", "Gold": "12"},
	}

	first, err := engine.Launch(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	first.Say("I press on")
	waitFor(t, first, "props", "map[Gold:12 Health:Wounded")
	state := first.State()
	first.Close()

	// Relaunch with no seed at all: whatever the player sees must come from the
	// restored state rather than from the spec.
	resumed, err := engine.Resume(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s4"}, state)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()

	waitFor(t, resumed, "props", "map[Gold:12 Health:Wounded")
}
