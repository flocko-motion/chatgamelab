package engine_test

import (
	"context"
	"encoding/json"
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

// A genre that takes no typed input must say so rather than swallow it.
func TestWrongInputKindIsRejected(t *testing.T) {
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

	if err := s.Say("typing at a voice genre"); err == nil {
		t.Error("expected Say to be refused by npc-live")
	}
	if err := s.Speak([]byte("persuasion")); err != nil {
		t.Errorf("Speak should be accepted by npc-live: %v", err)
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
