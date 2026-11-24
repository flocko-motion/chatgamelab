package engine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"engine"
	"engine/internal/adapters/mock"
)

// NPC-Rounds plays end to end on the mock: the cover first, then the
// character's opening words, a typed and a spoken turn answered aloud, and a
// resume that carries on the one conversation.
func TestNPCRoundsEndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &engine.MemoryPersist{}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre:     engine.SubgenreNPCRounds,
		ID:           "npc-rounds",
		Persist:      store,
		Scenario:     "You are the bridge keeper. Do not concede passage.",
		InitPrompt:   "A traveller walks up to the bridge.",
		Guardrail:    "Suitable for a 10-year-old.",
		GalleryImage: mock.GalleryImage("bridge keeper"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	firstText := ""
	deadline := time.After(5 * time.Second)
	for firstText == "" {
		select {
		case e := <-s.Events():
			seen[e.Stream] = true
			if e.Stream == "text" && e.Value != "" {
				firstText = e.Value
			}
		case <-deadline:
			t.Fatalf("the character never spoke first: saw %v", seen)
		}
	}
	waitForTurn(t, s, "[story turn 1] The conversation begins now.")
	if !hasStream(s.History(), "image") {
		t.Error("the cover was not shown")
	}

	if err := s.Say("Let me pass."); err != nil {
		t.Fatal(err)
	}
	waitForTurn(t, s, "[story turn 2] Let me pass.")
	if err := s.Speak(spokenClip()); err != nil {
		t.Fatal(err)
	}
	waitForTurn(t, s, "[story turn 3] [mock-hearing]")
	if !hasStream(s.History(), "said") {
		t.Error("the player's lines were not kept as theirs")
	}
	// The mock offer allows audio-out by default, so the answers are spoken.
	waitFor(t, s, "audio", "")
	s.Close()

	resumed := resume(t, store, "npc-rounds")
	if !hasText(resumed.History(), "[story turn 1]") || !hasText(resumed.History(), "[story turn 3]") {
		t.Errorf("the resumed history lacks the conversation: %q", textOf(resumed.History()))
	}
	if err := resumed.Say("Please?"); err != nil {
		t.Fatal(err)
	}
	// Turn 4, not a second opening: the conversation carried on.
	waitForTurn(t, resumed, "[story turn 4] Please?")
	if strings.Count(textOf(resumed.History()), "The conversation begins now.") != 1 {
		t.Error("the resumed game opened a second time")
	}
}
