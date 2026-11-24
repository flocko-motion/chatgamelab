package engine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"engine"
	"engine/internal/adapters/mock"
)

// When the provider's safety filter ends the conversation, the game is over:
// final, with the reason as the last line of its history, read back on a
// resume and never played on.
func TestTheSafetyFilterEndsTheGame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "filtered", Persist: store,
		Scenario: "You keep a bridge.", GalleryImage: mock.GalleryImage("bridge"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	if err := awaitConnection(ctx, s); err != nil {
		t.Fatal(err)
	}
	if err := s.Inject("observer", "direction", mock.ContentFiltered); err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "over", string(engine.OverContentPolicy))

	if over := s.Snapshot().Over; over == nil || over.Reason != engine.OverContentPolicy {
		t.Errorf("the snapshot says over %+v, want content_policy", over)
	}
	history := s.History()
	if last := history[len(history)-1]; last.Stream != "over" || last.Value != string(engine.OverContentPolicy) {
		t.Errorf("the last line of the history is %+v, want the reason the game is over", last)
	}
	var refusal *engine.Refusal
	if _, err := s.Connect(ctx, "v=0 offer", ""); !errors.As(err, &refusal) || refusal.Problem != engine.ProblemOver {
		t.Errorf("connecting to a game that is over: %v, want refused as over", err)
	}
	s.Close()

	// Read back, the game is still over, with its history, and nothing runs.
	resumed := resume(t, store, "filtered")
	if over := resumed.Snapshot().Over; over == nil || over.Reason != engine.OverContentPolicy {
		t.Fatalf("the resumed snapshot says over %+v, want content_policy", over)
	}
	if !resumed.Snapshot().Started {
		t.Error("a game that is over reads as not yet started")
	}
	if got := resumed.History(); len(got) == 0 || got[len(got)-1].Stream != "over" {
		t.Errorf("the resumed history does not end with the reason: %+v", got)
	}
	select {
	case e := <-resumed.Events():
		if e.Stream == "connect" {
			t.Error("a game that is over invited the player to connect")
		}
	case <-time.After(300 * time.Millisecond):
	}
	if err := resumed.Inject("observer", "direction", "Go on."); !errors.As(err, &refusal) || refusal.Problem != engine.ProblemOver {
		t.Errorf("injecting into a game that is over: %v, want refused as over", err)
	}
}
