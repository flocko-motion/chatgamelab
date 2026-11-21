package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"engine"
)

// launchStored launches on a memory store, as a platform launches on its tables.
func launchStored(t *testing.T, store engine.Persist, spec engine.SessionSpec) *engine.Session {
	t.Helper()
	spec.Persist = store
	s, err := engine.Launch(context.Background(), spec, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func resume(t *testing.T, store engine.Persist, id string) *engine.Session {
	t.Helper()
	s, err := engine.Resume(context.Background(), id, store, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

// A session stored and resumed continues its provider-side thread and shows
// the conversation so far. Without this, every restart silently loses the
// conversation the model is holding.
func TestResumeContinuesTheThread(t *testing.T) {
	store := &engine.MemoryPersist{}
	first := launchStored(t, store, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s1"})
	if err := first.Say("I try to cross the bridge"); err != nil {
		t.Fatal(err)
	}
	// The reply streams; the history keeps it whole once it has closed.
	waitFor(t, first, "audio", "<narration of [story turn 1]")
	for deadline := time.Now().Add(3 * time.Second); !hasText(first.History(), "[story turn 1]"); {
		if time.Now().After(deadline) {
			t.Fatalf("the history never kept the first turn whole: %q", first.History())
		}
		time.Sleep(5 * time.Millisecond)
	}
	first.Close()

	resumed := resume(t, store, "s1")
	if !hasText(resumed.History(), "[story turn 1]") {
		t.Errorf("the restored history lacks the first turn as one utterance: %q", resumed.History())
	}
	if err := resumed.Say("I keep going"); err != nil {
		t.Fatal(err)
	}
	// Turn 2, not turn 1: the thread continued.
	waitFor(t, resumed, "audio", "<narration of [story turn 2]")
}

// hasText reports whether the history holds a text event starting with prefix.
func hasText(events []engine.SessionEvent, prefix string) bool {
	for _, e := range events {
		if e.Stream == "text" && strings.HasPrefix(e.Value, prefix) {
			return true
		}
	}
	return false
}

// The stored spec is the session's: a resume needs nothing but the ID.
func TestResumeKeepsTheSpecAndPreference(t *testing.T) {
	store := &engine.MemoryPersist{}
	first := launchStored(t, store, engine.SessionSpec{
		Genre: engine.GenreAdventure, ID: "s5", Title: "The Bridge", Theme: "noir", Language: "de",
	})
	first.Close()

	resumed := resume(t, store, "s5")
	topology := resumed.Topology()
	if topology.Title != "The Bridge" || topology.Theme != "noir" || topology.Language != "de" {
		t.Errorf("topology = %+v, want the stored title, theme and language", topology)
	}
	if pref := resumed.Preference(); pref == nil || pref.Offer != "mock" {
		t.Errorf("preference = %+v, want the one settled at launch", pref)
	}
}

// A session stored by another version of the engine is refused, so the
// playback-engine can offer to start anew.
func TestResumeRefusesAnotherVersion(t *testing.T) {
	store := &engine.MemoryPersist{}
	first := launchStored(t, store, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s6"})
	first.Close()

	stored, err := store.LoadState(context.Background(), "s6")
	if err != nil {
		t.Fatal(err)
	}
	stored.Version = engine.SchemaVersion - 1
	if err := store.SaveState(context.Background(), "s6", stored); err != nil {
		t.Fatal(err)
	}

	_, err = engine.Resume(context.Background(), "s6", store, mockOffers())
	if !errors.Is(err, engine.ErrStale) {
		t.Fatalf("got %v, want ErrStale", err)
	}
}

// State naming a block the wiring no longer has means the wiring changed under
// a stored session. That has to fail rather than half-restore.
func TestResumeRefusesUnknownBlock(t *testing.T) {
	store := &engine.MemoryPersist{}
	if err := store.SaveState(context.Background(), "s3", engine.Stored{
		Version: engine.SchemaVersion,
		Spec:    engine.SessionSpec{Genre: engine.GenreAdventure},
		State:   engine.SessionState{Blocks: map[string]string{"a-block-that-left": "{}"}},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := engine.Resume(context.Background(), "s3", store, mockOffers())
	if !errors.Is(err, engine.ErrStale) {
		t.Fatalf("got %v, want ErrStale", err)
	}
	if !strings.Contains(err.Error(), "a-block-that-left") {
		t.Errorf("error should name the missing block, got: %v", err)
	}
}

func TestResumeOfNothingIsUnknown(t *testing.T) {
	_, err := engine.Resume(context.Background(), "never", &engine.MemoryPersist{}, mockOffers())
	if !errors.Is(err, engine.ErrUnknown) {
		t.Fatalf("got %v, want ErrUnknown", err)
	}
}

// A fresh launch with the same spec starts over, which is what makes the
// resume tests meaningful.
func TestFreshLaunchDoesNotContinue(t *testing.T) {
	s := launchStored(t, &engine.MemoryPersist{}, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s2"})
	defer s.Close()

	s.Say("I try to cross the bridge")
	waitFor(t, s, "audio", "<narration of [story turn 1]")
}

// Status values are not a handle to something a provider remembers: if the
// engine loses them, they are gone, and the player's status bar resets. So they
// have to survive a resume too.
func TestResumeRestoresStatusValues(t *testing.T) {
	store := &engine.MemoryPersist{}
	first := launchStored(t, store, engine.SessionSpec{
		Genre:  engine.GenreAdventure,
		ID:     "s4",
		Status: map[string]string{"Health": "Wounded", "Gold": "12"},
	})
	first.Say("I press on")
	waitFor(t, first, "props", "map[Gold:12 Health:Wounded")
	first.Close()

	resumed := resume(t, store, "s4")
	if got := resumed.Snapshot().Props["Gold"]; got != "12" {
		t.Errorf("restored props = %v, want the stored values", resumed.Snapshot().Props)
	}
	waitFor(t, resumed, "props", "map[Gold:12 Health:Wounded")
}

// A live conversation open before a restart is not re-attached: it comes back
// paused, for the player to resume with one press, and the character does not
// greet again.
func TestResumedConversationWaitsPaused(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	spec := engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "live", Persist: store,
		Scenario: "You keep a bridge.", Guardrail: "Suitable for a 13-year-old.",
	}
	first, err := engine.Launch(ctx, spec, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	if err := awaitConnection(ctx, first); err != nil {
		t.Fatal(err)
	}
	waitFor(t, first, "text", "")
	first.Close()

	resumed := resume(t, store, "live")
	if got := resumed.Snapshot().Conversation; got != "paused" {
		t.Errorf("conversation = %q, want paused", got)
	}
	if countStream(resumed.History(), "image") == 0 {
		t.Error("the portrait was not kept")
	}
}

func textOf(events []engine.SessionEvent) string {
	var b strings.Builder
	for _, e := range events {
		if e.Stream == "text" {
			b.WriteString(e.Value)
		}
	}
	return b.String()
}

func countStream(events []engine.SessionEvent, stream string) int {
	n := 0
	for _, e := range events {
		if e.Stream == stream {
			n++
		}
	}
	return n
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
