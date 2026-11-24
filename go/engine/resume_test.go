package engine_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"engine"
	"engine/internal/adapters/mock"
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
	s, err := engine.Resume(context.Background(), id, store, mockOffers(), mock.GalleryImage("bridge"))
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
	// The reply streams; the history keeps it whole once it has closed. The
	// opening takes the thread's first two turns, the action the next two.
	waitForTurn(t, first, "[story turn 4]")
	first.Close()

	resumed := resume(t, store, "s1")
	if !hasText(resumed.History(), "[story turn 2]") || !hasText(resumed.History(), "[story turn 4]") {
		t.Errorf("the restored history lacks the opening and the first turn as utterances: %q", resumed.History())
	}
	if err := resumed.Say("I keep going"); err != nil {
		t.Fatal(err)
	}
	// Turn 6, not a second opening: the thread continued.
	waitForTurn(t, resumed, "[story turn 6] NARRATE the outline")
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

	_, err = engine.Resume(context.Background(), "s6", store, mockOffers(), nil)
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
	_, err := engine.Resume(context.Background(), "s3", store, mockOffers(), nil)
	if !errors.Is(err, engine.ErrStale) {
		t.Fatalf("got %v, want ErrStale", err)
	}
	if !strings.Contains(err.Error(), "a-block-that-left") {
		t.Errorf("error should name the missing block, got: %v", err)
	}
}

func TestResumeOfNothingIsUnknown(t *testing.T) {
	_, err := engine.Resume(context.Background(), "never", &engine.MemoryPersist{}, mockOffers(), nil)
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
	waitForTurn(t, s, "[story turn 2]")
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
	// JSON, so values with spaces arrive whole. The opening set the values,
	// and the turn went on from them.
	const afterTurn = `{"Gold":"[turn 3]","Health":"[turn 3]"}`
	waitFor(t, first, "props", afterTurn)
	first.Close()

	resumed := resume(t, store, "s4")
	if got := resumed.Snapshot().Props["Gold"]; got != "[turn 3]" {
		t.Errorf("restored props = %v, want the stored values", resumed.Snapshot().Props)
	}
	waitFor(t, resumed, "props", afterTurn)
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
		GalleryImage: mock.GalleryImage("bridge"),
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

// A session stored before the gate kept its own state still starts again on a
// resume: what was said shows the game had begun.
func TestResumeOfAnOlderStoredLiveGameStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	spec := engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "older", Persist: store,
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

	// As stored before the gate kept its state.
	stored, err := store.LoadState(ctx, "older")
	if err != nil {
		t.Fatal(err)
	}
	delete(stored.State.Blocks, "start-game")
	if err := store.SaveState(ctx, "older", stored); err != nil {
		t.Fatal(err)
	}

	resumed := resume(t, store, "older")
	select {
	case <-resumed.Ready():
	case <-time.After(3 * time.Second):
		t.Fatalf("the older session never started again: %+v", resumed.Journal())
	}
}

// A resumed live game starts again without showing its portrait again: the
// gate that waited for the portrait opens at once, and the player can connect
// and talk on.
func TestResumedLiveGameOpensWithoutANewPortrait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	spec := engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "live", Persist: store,
		Scenario: "You keep a bridge.", Guardrail: "Suitable for a 13-year-old.",
		GalleryImage: mock.GalleryImage("bridge"),
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
	select {
	case <-resumed.Ready():
	case <-time.After(3 * time.Second):
		t.Fatalf("the resumed game never started: %+v", resumed.Journal())
	}
	if !resumed.Snapshot().Started {
		t.Error("the snapshot says the resumed game has not started")
	}
	// The player presses resume: the conversation opens again.
	connected := make(chan error, 1)
	go func() {
		_, err := resumed.Connect(ctx, "v=0 mock offer")
		connected <- err
	}()
	select {
	case err := <-connected:
		if err != nil {
			t.Fatalf("connect after the resume: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("the resumed conversation never answered the offer: %+v", resumed.Journal())
	}
	if countStream(resumed.History(), "image") != 1 {
		t.Error("the resume showed the portrait again")
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

// waitForTurn waits until the history keeps a whole reply starting prefix.
func waitForTurn(t *testing.T, s *engine.Session, prefix string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !hasText(s.History(), prefix); {
		if time.Now().After(deadline) {
			t.Fatalf("the history never kept %q whole: %q", prefix, s.History())
		}
		time.Sleep(5 * time.Millisecond)
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

// A session resumed before its gallery image was shown shows it then: the
// picture is handed in again on resume, never stored.
func TestResumeShowsAGalleryImageNotYetShown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	first, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "late", Persist: store,
		Scenario: "You keep a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-first.Ready():
	case <-time.After(3 * time.Second):
		t.Fatal("the first session never started")
	}
	first.Close()

	resumed := resume(t, store, "late")
	for deadline := time.Now().Add(3 * time.Second); countStream(resumed.History(), "image") == 0; {
		if time.Now().After(deadline) {
			t.Fatalf("the resumed session never showed its gallery image: %+v", resumed.Journal())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// The journal is stored with the session, so a resumed session still shows
// what happened before: a session reported broken can be read back whole.
func TestResumeKeepsTheJournal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	first, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "journalled", Persist: store,
		Scenario: "You keep a bridge.", GalleryImage: mock.GalleryImage("bridge"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	if err := awaitConnection(ctx, first); err != nil {
		t.Fatal(err)
	}
	first.Close()

	resumed := resume(t, store, "journalled")
	entries := resumed.Journal()
	kinds := make([]string, len(entries))
	for i, e := range entries {
		kinds[i] = e.Source + " " + e.Kind
	}
	joined := strings.Join(kinds, ", ")
	for _, want := range []string{"session launched", "live-session opened", "session closed", "session resumed"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the resumed journal lacks %q: %s", want, joined)
		}
	}
	if strings.Index(joined, "session launched") > strings.Index(joined, "session resumed") {
		t.Error("the stored journal does not come before what happened after the resume")
	}
	if hasStream(resumed.History(), "journal") {
		t.Error("the journal leaked into the story's history")
	}
}

func hasStream(events []engine.SessionEvent, stream string) bool {
	return countStream(events, stream) > 0
}

// A restart while the game is still preparing, before its gate opened, does
// not leave it waiting: resumed, the first words are written, once, and the
// gate opens.
func TestResumeBeforeTheGateOpenedStillStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	first, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "early", Persist: store,
		Scenario: "You keep a bridge.",
		// Slow enough that the first words are not written yet when it closes.
		MockPaceMs: 400,
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if gateOpened(first.Journal()) {
		t.Fatal("the gate opened before the test could close the session")
	}
	first.Close()

	resumed := resume(t, store, "early")
	for deadline := time.Now().Add(5 * time.Second); !gateOpened(resumed.Journal()); {
		if time.Now().After(deadline) {
			t.Fatalf("the resumed session never opened its gate: %+v", resumed.Journal())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// gateOpened reports whether the conversation has heard the gate's cue.
func gateOpened(journal []engine.JournalEntry) bool {
	for _, e := range journal {
		if e.Source == "live-session" && e.Kind == "cued" {
			return true
		}
	}
	return false
}

// overlapWatch is a store that notes when two saves of a session overlap, as
// a store that numbers events in order cannot take.
type overlapWatch struct {
	engine.MemoryPersist
	busy    atomic.Int32
	overlap atomic.Bool
}

func (w *overlapWatch) SaveEvent(ctx context.Context, id string, e engine.SessionEvent) error {
	if w.busy.Add(1) > 1 {
		w.overlap.Store(true)
	}
	defer w.busy.Add(-1)
	time.Sleep(time.Millisecond) // a store's round trip, so overlaps show
	return w.MemoryPersist.SaveEvent(ctx, id, e)
}

// A session's saves never overlap: the history and the journal go to the
// store one at a time, so a store numbering them in order loses nothing,
// the gallery image saved at launch included.
func TestSavesOfASessionNeverOverlap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &overlapWatch{}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "ordered", Persist: store,
		Scenario: "You keep a bridge.", GalleryImage: mock.GalleryImage("bridge"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	if err := awaitConnection(ctx, s); err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "text", "")
	s.Close()

	if store.overlap.Load() {
		t.Error("two saves of the session overlapped")
	}
	if countStream(store.Events("ordered"), "image") != 1 {
		t.Error("the gallery image was not stored")
	}
}
