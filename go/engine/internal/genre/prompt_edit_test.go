package genre

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"engine/internal/adapters/mock"
)

// Every prompt of a genre belongs to a block of its graph, so each can be
// shown with its block and edited during play.
func TestEveryPromptHasItsBlock(t *testing.T) {
	for name, w := range map[string]*Wiring{
		"adventure": mustAdventure(t, nil),
		"npc-live":  mustNPCLive(t, mockNPCConfig("")),
		"npc-duo":   mustNPCDuo(t, mockDuoConfig("")),
	} {
		t.Run(name, func(t *testing.T) {
			if got, want := slices.Sorted(maps.Keys(w.PromptUses)), slices.Sorted(maps.Keys(w.Prompts)); !slices.Equal(got, want) {
				t.Errorf("prompts with a block %v, want every prompt %v", got, want)
			}
			for prompt, use := range w.PromptUses {
				if len(use.Blocks) == 0 {
					t.Errorf("%s belongs to no block", prompt)
				}
				for _, block := range use.Blocks {
					if _, found := w.Graph.Inspect(block); !found {
						t.Errorf("%s belongs to %q, which is no block of the graph", prompt, block)
					}
				}
			}
			if got, want := slices.Sorted(maps.Keys(w.Defaults)), slices.Sorted(maps.Keys(w.Prompts)); !slices.Equal(got, want) {
				t.Errorf("defaults %v, want one per prompt %v", got, want)
			}
		})
	}
}

// An edited prompt is filled like the genre's own, and one whose template
// does not fill is refused, leaving the prompt as it was.
func TestEditedPromptIsFilled(t *testing.T) {
	w := adventure(t, AdventureConfig{Language: "de"})
	if err := w.SetPrompt(PromptRephrase, "Say it as a pirate, in {{.Language}}."); err != nil {
		t.Fatal(err)
	}
	if got := w.CurrentPrompts()[PromptRephrase]; got != "Say it as a pirate, in Deutsch." {
		t.Errorf("rephrase runs on %q", got)
	}
	if err := w.SetPrompt(PromptRephrase, "{{.Nothing}}"); err == nil {
		t.Error("a template that does not fill was accepted")
	}
	if got := w.CurrentPrompts()[PromptRephrase]; got != "Say it as a pirate, in Deutsch." {
		t.Errorf("a refused edit changed rephrase to %q", got)
	}
}

// A System Message edited during play leads the next call on a fresh
// provider thread, seeded with what the thread holds.
func TestEditedSystemMessageStartsAFreshThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	story := &threadLog{}
	w := adventure(t, AdventureConfig{Thread: story})
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening's prose")

	if err := w.SetPrompt(PromptGameMaster, "You are a terse game master."); err != nil {
		t.Fatal(err)
	}
	before := len(story.seen())
	w.Say("I walk on")
	receiveLine(t, text, "a turn's prose")

	next := story.seen()[before]
	if next.Continuation.ID != "" {
		t.Errorf("the call after the edit went on from %q, want a fresh provider thread", next.Continuation.ID)
	}
	if !strings.HasPrefix(next.System, "You are a terse game master.") {
		t.Errorf("the call after the edit ran on system %q", next.System)
	}
	if len(next.Transcript) == 0 {
		t.Error("the fresh thread was seeded with nothing; the story so far is lost")
	}
}

// A guardrail edited during play reaches the story thread's next call: its
// System Message, on a fresh provider thread, and the rule the call carries.
func TestEditedGuardrailReachesTheNextCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	story := &threadLog{}
	w := adventure(t, AdventureConfig{Thread: story, Guardrail: "No violence."})
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening's prose")

	w.SetGuardrail("Gentle fights are fine.")
	if got := w.CurrentGuardrail(); got != "Gentle fights are fine." {
		t.Errorf("the wiring keeps to %q", got)
	}
	before := len(story.seen())
	w.Say("I draw my sword")
	receiveLine(t, text, "a turn's prose")

	next := story.seen()[before]
	if next.Continuation.ID != "" {
		t.Errorf("the call after the edit went on from %q, want a fresh provider thread", next.Continuation.ID)
	}
	want := "NARRATION RULES must be respected: Gentle fights are fine."
	if !strings.HasSuffix(next.System, want) || strings.Contains(next.System, "No violence.") {
		t.Errorf("the System Message after the edit ends %q", next.System[max(0, len(next.System)-80):])
	}
	if next.Constraints != want {
		t.Errorf("the call after the edit carried %q", next.Constraints)
	}
}

// Every block that keeps to the guardrail is a block of the graph.
func TestGuardrailHasItsBlocks(t *testing.T) {
	for name, w := range map[string]*Wiring{
		"adventure": mustAdventure(t, nil),
		"npc-live":  mustNPCLive(t, mockNPCConfig("No violence.")),
		"npc-duo":   mustNPCDuo(t, mockDuoConfig("No violence.")),
	} {
		t.Run(name, func(t *testing.T) {
			if len(w.GuardrailUse.Blocks) == 0 || w.GuardrailUse.Apply == nil {
				t.Fatal("no block keeps to the guardrail")
			}
			for _, block := range w.GuardrailUse.Blocks {
				if _, found := w.Graph.Inspect(block); !found {
					t.Errorf("%q keeps to the guardrail, and is no block of the graph", block)
				}
			}
			w.SetGuardrail("Be kind.")
			if got := w.CurrentGuardrail(); got != "Be kind." {
				t.Errorf("the wiring keeps to %q", got)
			}
		})
	}
}

func mockDuoConfig(guardrail string) NPCDuoConfig {
	return NPCDuoConfig{
		Tool: mock.Tool{}, Live: mock.Live{}, Guardrail: guardrail,
		Scenario:  "You are the bridge keeper." + DuoSeparator + "You are the ferryman.",
		GameStart: "Both wait at the river.",
	}
}

func mustNPCDuo(t *testing.T, cfg NPCDuoConfig) *Wiring {
	t.Helper()
	w, err := NewNPCDuo(cfg)
	if err != nil {
		t.Fatalf("wiring npc-duo: %v", err)
	}
	return w
}
