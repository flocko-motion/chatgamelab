package engine_test

import (
	"context"
	"testing"

	"engine"
)

// A prompt edited during play takes effect on its block's next call, is
// stored with the spec, and a resumed session runs on it.
func TestPromptEditedDuringPlayTakesEffect(t *testing.T) {
	store := &engine.MemoryPersist{}
	s := launchStored(t, store, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "edit"})
	waitForTurn(t, s, "[story turn 2]")

	if err := s.SetPrompt("expand", "TELL IT IN RHYME"); err != nil {
		t.Fatal(err)
	}
	if got := s.Prompts()["expand"]; got != "TELL IT IN RHYME" {
		t.Errorf("the session reports running on %q", got)
	}
	if err := s.Say("I cross the bridge"); err != nil {
		t.Fatal(err)
	}
	waitForTurn(t, s, "[story turn 4] TELL IT IN RHYME")
	s.Close()

	stored, err := store.LoadState(context.Background(), "edit")
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Spec.PromptOverride["expand"]; got != "TELL IT IN RHYME" {
		t.Errorf("stored override %q", got)
	}

	resumed := resume(t, store, "edit")
	if err := resumed.Say("I keep going"); err != nil {
		t.Fatal(err)
	}
	waitForTurn(t, resumed, "[story turn 6] TELL IT IN RHYME")
}

// A prompt the genre lacks is refused, and nothing is stored.
func TestUnknownPromptIsRefusedDuringPlay(t *testing.T) {
	s := launchStored(t, &engine.MemoryPersist{}, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "typo"})
	defer s.Close()
	if err := s.SetPrompt("expnad", "x"); err == nil {
		t.Error("a misspelled prompt was accepted")
	}
}

// A block's details list the prompts it runs on, each marked when it is no
// longer the genre's own.
func TestBlockDetailListsItsPrompts(t *testing.T) {
	s := launchStored(t, &engine.MemoryPersist{}, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "show"})
	defer s.Close()
	if err := s.SetPrompt("expand", "TELL IT IN RHYME"); err != nil {
		t.Fatal(err)
	}

	detail, ok := s.Inspect("expand")
	if !ok {
		t.Fatal("no detail for expand")
	}
	got := map[string]engine.BlockPrompt{}
	for _, p := range detail.Prompts {
		got[p.Name] = p
	}
	if p := got["expand"]; p.Text != "TELL IT IN RHYME" || !p.Edited {
		t.Errorf("expand's own prompt = %+v, want the edit, marked", p)
	}
	if p := got["opening-expand"]; p.Text == "" || p.Edited {
		t.Errorf("the opening's prompt = %+v, want the genre's own", p)
	}
	// Expand runs on the story thread, so it lists the thread's System
	// Message and its summary request too.
	if len(got) != 4 || got["game-master"].Text == "" || got["summary"].Text == "" {
		t.Errorf("expand lists %+v, want its two prompts and the thread's two", detail.Prompts)
	}

	// Outline is on the same thread: an edit there is the same prompt.
	outline, _ := s.Inspect("outline")
	names := map[string]bool{}
	for _, p := range outline.Prompts {
		names[p.Name] = true
	}
	if len(names) != 2 || !names["game-master"] || !names["summary"] {
		t.Errorf("outline lists %+v, want the thread's two", outline.Prompts)
	}
}

// Each genre lists its own prompts by name, as a launch that adapts them
// needs them.
func TestGenresListTheirPrompts(t *testing.T) {
	for _, info := range engine.GenreList() {
		if len(info.Prompts) == 0 {
			t.Errorf("%s lists no prompts", info.ID)
		}
		for _, p := range info.Prompts {
			if p.Name == "" || p.Text == "" {
				t.Errorf("%s lists an empty prompt: %+v", info.ID, p)
			}
		}
	}
}
