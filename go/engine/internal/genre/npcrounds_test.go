package genre

import (
	"context"
	"strings"
	"testing"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
)

const testCharacter = "You are the bridge keeper. Do not concede passage."

func npcRounds(t testing.TB, cfg NPCRoundsConfig) *Wiring {
	t.Helper()
	if cfg.Thread == nil {
		cfg.Thread = mock.Thread{}
	}
	if cfg.Hearing == nil {
		cfg.Hearing = mock.Hearing{}
	}
	if cfg.Tool == nil {
		cfg.Tool = mock.Tool{}
	}
	if cfg.Scenario == "" {
		cfg.Scenario = testCharacter
	}
	if cfg.GalleryImage == nil {
		cfg.GalleryImage = mock.GalleryImage("bridge keeper")
	}
	w, err := NewNPCRounds(cfg)
	if err != nil {
		t.Fatalf("wiring npc-rounds: %v", err)
	}
	return w
}

// The player sees the cover, then the character speaks first, from how the
// game begins; each line the player types is answered on the one thread, as
// the player's own message.
func TestNPCRoundsFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	thread := &recordingThread{}
	w := npcRounds(t, NPCRoundsConfig{Thread: thread, GameStart: "A traveller walks up to the bridge.", Guardrail: testGuardrail})
	image, text, said := streamOf(w, "image"), utterances(streamOf(w, "text")), streamOf(w, "said")
	w.Graph.Start(ctx)

	receiveLine(t, image, "the cover")
	receiveLine(t, text, "the character's first words")
	opening := thread.seen()[0]
	if !opening.Directive || opening.Input != NPCRoundsPrompts[PromptNPCOpening] {
		t.Errorf("the opening was %+v, want the opening cue as the game's own message", opening)
	}
	if !strings.Contains(opening.System, testCharacter) || !strings.Contains(opening.System, "A traveller walks up to the bridge.") {
		t.Errorf("the System Message lacks the scenario or how the game begins:\n%s", opening.System)
	}
	if strings.Contains(opening.System, testGuardrail) && opening.Constraints == "" {
		t.Error("the guardrail went into the System Message only")
	}

	w.Say("Let me pass.")
	if got := receiveLine(t, said, "the player's line"); got != "Let me pass." {
		t.Errorf("said %q", got)
	}
	reply := receiveLine(t, text, "the character's answer")
	turn := thread.seen()[1]
	if turn.Directive || turn.Input != "Let me pass." {
		t.Errorf("the player's turn was %+v, want their words as their own message", turn)
	}
	if !strings.Contains(reply, "Let me pass.") {
		t.Errorf("the answer %q was not to the player's words", reply)
	}
}

// With every call go the guardrail and the character's line, at the
// character's constraint frequency; the opening always carries them.
func TestNPCRoundsConstraintsGoWithTheCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	thread := &recordingThread{}
	w := npcRounds(t, NPCRoundsConfig{Thread: thread, Guardrail: testGuardrail,
		ConstraintFrequency: map[string]int{BlockCharacter: 2}})
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening")
	for _, line := range []string{"one", "two", "three"} {
		w.Say(line)
		receiveLine(t, text, "the answer to "+line)
	}
	turns := thread.seen()
	for i, want := range []bool{true, false, true, false} {
		carried := strings.Contains(turns[i].Constraints, testGuardrail) &&
			strings.Contains(turns[i].Constraints, NPCRoundsPrompts[PromptHoldTheLine])
		if carried != want {
			t.Errorf("call %d carried the constraints: %v, want %v (%q)", i, carried, want, turns[i].Constraints)
		}
	}
	if strings.Contains(turns[0].Constraints, testCharacter) {
		t.Error("the designer's scenario shares a message with the guardrail")
	}
}

// A spoken turn is heard, shown as the player's line and answered; with
// audio-out the answer is spoken too. Nothing heard starts no turn.
func TestNPCRoundsSpeaks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := npcRounds(t, NPCRoundsConfig{})
	text, said, audio := utterances(streamOf(w, "text")), streamOf(w, "said"), streamOf(w, "audio")
	voiceFailed := sinkNamed(t, w, "voice-failed")
	for _, block := range w.Switches[adapters.ToggleAudioOut] {
		block.SetActive(true)
	}
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening")
	receiveLine(t, audio, "the opening spoken")

	if err := w.Speak(mock.SilentClip(1)); err != nil {
		t.Fatal(err)
	}
	receiveLine(t, voiceFailed, "the failed voice")

	if err := w.Speak(mock.SpokenClip(1.5)); err != nil {
		t.Fatal(err)
	}
	if got := receiveLine(t, said, "the spoken line"); got != "[mock-hearing] 1.5 s of speech" {
		t.Errorf("said %q", got)
	}
	if got := receiveLine(t, text, "the answer"); !strings.Contains(got, "[mock-hearing] 1.5 s of speech") {
		t.Errorf("the answer %q was not to what was heard", got)
	}

	for _, block := range w.Switches[adapters.ToggleAudioIn] {
		block.SetActive(false)
	}
	if err := w.Speak(mock.SpokenClip(1)); err != ErrVoiceOff {
		t.Errorf("voice off, Speak answered %v", err)
	}
}

// NPC-Rounds has no observer: the guardrail goes with the calls, so the
// conversation takes neither injections nor a director.
func TestNPCRoundsHasNoObserver(t *testing.T) {
	w := npcRounds(t, NPCRoundsConfig{})
	if len(w.Observers) != 0 || len(w.Injectable) != 0 || w.Live != nil {
		t.Errorf("observers %v, injectable %v, live %v", w.Observers, w.Injectable, w.Live)
	}
	if _, ok := w.OnThread[BlockCharacter]; !ok {
		t.Error("the character is on no thread, so it cannot compact")
	}
}
