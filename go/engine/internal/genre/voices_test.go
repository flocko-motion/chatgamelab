package genre

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// castVoices are the built-in voices the tests cast from: two languages, both
// genders, and one voice whose gender is not stated.
var castVoices = []adapters.BuiltinVoice{
	{Name: "ann", Language: "en", Accent: "British", Gender: "feminine"},
	{Name: "bob", Language: "en", Accent: "British", Gender: "masculine"},
	{Name: "kit", Language: "en", Accent: "Irish"},
	{Name: "uta", Language: "de", Accent: "German", Gender: "feminine"},
	{Name: "udo", Language: "de", Accent: "German", Gender: "masculine"},
}

// refusingTool refuses every call, as a provider that is down for good.
type refusingTool struct{ mock.Tool }

func (refusingTool) QuerySchema(context.Context, string, string, adapters.Schema) (string, adapters.Usage, error) {
	return "", adapters.Usage{}, errors.New("refused")
}

// lastPickTool answers a schema as the mock does, but picks the last of an
// enum's values: a cast another tool would not repeat.
type lastPickTool struct{ mock.Tool }

func (lastPickTool) QuerySchema(_ context.Context, _, _ string, schema adapters.Schema) (string, adapters.Usage, error) {
	reply := map[string]any{}
	props, _ := schema.Body["properties"].(map[string]any)
	for name, p := range props {
		prop, _ := p.(map[string]any)
		reply[name] = "x"
		if values, ok := prop["enum"].([]any); ok && len(values) > 0 {
			reply[name] = values[len(values)-1]
		}
	}
	out, err := json.Marshal(reply)
	return string(out), adapters.Usage{}, err
}

// duoVoices launches a scene and answers the voice each character's
// conversation opens with, the first's and the second's.
func duoVoices(t *testing.T, cfg NPCDuoConfig) (first, second string) {
	t.Helper()
	first, second, _ = duoVoicesFrom(t, cfg, nil)
	return first, second
}

// duoVoicesFrom is duoVoices on a scene restored from stored, if any, and
// answers what the scene stores.
func duoVoicesFrom(t *testing.T, cfg NPCDuoConfig, stored map[string]string) (first, second string, state map[string]string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &recordingLive{opened: make(chan adapters.LiveConfig, 2)}
	cfg.Live = recorder
	cfg.GalleryImage = mock.GalleryImage("lighthouse")
	cfg.Scenario = "A grumpy lighthouse keeper." + DuoSeparator + "A cheerful tax inspector."
	cfg.BuiltinVoices = castVoices
	w, err := NewNPCDuo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		if err := w.Graph.RestoreState(stored); err != nil {
			t.Fatal(err)
		}
	}
	// The browser connects once invited, as it does in play.
	lives := []*blocks.LiveSession{w.Live, w.Partner}
	invites := []<-chan ports.State{w.Live.Invitations(), w.Partner.Invitations()}
	w.Graph.Start(ctx)
	voices := map[string]string{}
	for i, live := range lives {
		nextInvitation(t, invites[i])
		if _, err := live.Connect(ctx, "v=0 mock offer"); err != nil {
			t.Fatalf("connect: %v", err)
		}
		select {
		case cfg := <-recorder.opened:
			voices[cfg.Scenario[:len("A grumpy")]] = cfg.Voice
		case <-time.After(2 * time.Second):
			t.Fatal("a conversation was never opened")
		}
	}
	return voices["A grumpy"], voices["A cheerf"], w.Graph.ExportState()
}

// Each character is cast in a voice of the gender the game chose for it,
// one that speaks the game's language.
func TestTheDuoIsCastInTheChosenGenders(t *testing.T) {
	first, second := duoVoices(t, NPCDuoConfig{
		Tool:         mock.Tool{},
		Language:     "de",
		VoiceChoices: map[string]VoiceChoice{VoiceA: VoiceMale, VoiceB: VoiceFemale},
	})
	if first != "udo" || second != "uta" {
		t.Errorf("cast A in %q and B in %q, want udo (male, German) and uta (female, German)", first, second)
	}
}

// Without a cast, a character takes the first voice of its gender in the
// game's language, and one left to any a voice the other does not have.
func TestAFailedDuoCastFallsBackWithinTheChosenGenders(t *testing.T) {
	first, second := duoVoices(t, NPCDuoConfig{
		Tool:         refusingTool{},
		Language:     "de",
		VoiceChoices: map[string]VoiceChoice{VoiceA: VoiceFemale},
	})
	if first != "uta" || second != "udo" {
		t.Errorf("fell back to %q and %q, want uta (female, German) and udo (the next German voice)", first, second)
	}
}

// A resumed scene keeps its cast: the characters speak in the voices they
// were cast in, though a new cast would pick others.
func TestAResumedDuoKeepsItsCast(t *testing.T) {
	cfg := NPCDuoConfig{Tool: lastPickTool{}, VoiceChoices: map[string]VoiceChoice{VoiceA: VoiceMale, VoiceB: VoiceFemale}}
	first, second, stored := duoVoicesFrom(t, cfg, nil)
	if first != "udo" || second != "uta" {
		t.Fatalf("cast A in %q and B in %q, want the last of each gender: udo and uta", first, second)
	}
	cfg.Tool = mock.Tool{}
	again, againSecond, _ := duoVoicesFrom(t, cfg, stored)
	if again != first || againSecond != second {
		t.Errorf("resumed in %q and %q, want the cast kept: %q and %q", again, againSecond, first, second)
	}
}

// npcVoice launches NPC-Live with a voice style on a key that makes voices,
// and answers the voice its conversation opens with and how many voices
// were made.
func npcVoice(t *testing.T, tool adapters.Tool, choice VoiceChoice) (voice string, made int32) {
	t.Helper()
	voice, made, _ = npcVoiceFrom(t, tool, choice, nil)
	return voice, made
}

// npcVoiceFrom is npcVoice on a game restored from stored, if any, and
// answers what the game stores.
func npcVoiceFrom(t *testing.T, tool adapters.Tool, choice VoiceChoice, stored map[string]string) (voice string, made int32, state map[string]string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voices := &adapters.VoicesSlot{}
	maker := &mock.Voices{}
	voices.Set(maker, &adapters.MemoryVoices{})
	recorder := &recordingLive{opened: make(chan adapters.LiveConfig, 1)}
	w, err := NewNPCLive(NPCLiveConfig{
		Live:          recorder,
		GalleryImage:  mock.GalleryImage("bridge"),
		Tool:          tool,
		Scenario:      "You are the keeper of a bridge.",
		VoiceStyle:    "deep, slow and a little hoarse",
		Voices:        voices,
		BuiltinVoices: castVoices,
		Language:      "de",
		VoiceChoices:  map[string]VoiceChoice{VoiceCharacter: choice},
		InitPrompt:    "Begin. Greet whoever has arrived.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		if err := w.Graph.RestoreState(stored); err != nil {
			t.Fatal(err)
		}
	}
	invites := w.Live.Invitations()
	w.Graph.Start(ctx)
	nextInvitation(t, invites)
	if _, err := w.Live.Connect(ctx, "v=0 mock offer"); err != nil {
		t.Fatalf("connect: %v", err)
	}
	select {
	case cfg := <-recorder.opened:
		return cfg.Voice, maker.Made.Load(), w.Graph.ExportState()
	case <-time.After(5 * time.Second):
		t.Fatal("the conversation was never opened")
	}
	return "", 0, nil
}

// A character set to a gender speaks in the built-in voice of that gender
// the brief finds fitting, and no voice is made for it, voice style or not.
func TestTheNPCSpeaksInABuiltInVoiceOfTheChosenGender(t *testing.T) {
	voice, made := npcVoice(t, mock.Tool{}, VoiceMale)
	if voice != "udo" {
		t.Errorf("speaks in %q, want udo, the first male voice in German", voice)
	}
	if made != 0 {
		t.Errorf("made %d voices, want none", made)
	}
}

// Without a brief, the character takes the first voice of its gender in the
// game's language.
func TestAFailedNPCBriefFallsBackWithinTheChosenGender(t *testing.T) {
	voice, made := npcVoice(t, refusingTool{}, VoiceFemale)
	if voice != "uta" || made != 0 {
		t.Errorf("speaks in %q having made %d voices, want uta and none", voice, made)
	}
}

// A character set to custom speaks in a voice made from the voice style.
func TestACustomNPCSpeaksInItsMadeVoice(t *testing.T) {
	voice, made := npcVoice(t, mock.Tool{}, VoiceCustom)
	if !strings.HasPrefix(voice, "voice_mock_") || made != 1 {
		t.Errorf("speaks in %q having made %d voices, want one made voice", voice, made)
	}
}

// narratorVoice plays an Adventure's opening, narrated, and answers the
// voice its first spoken turn asked for, and what the game stores.
func narratorVoice(t *testing.T, cfg AdventureConfig, stored map[string]string) (string, map[string]string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := &threadLog{}
	cfg.Thread = log
	cfg.Scenario = "A bridge over a chasm."
	cfg.SpeechVoices = castVoices
	w := adventure(t, cfg)
	if err := w.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		if err := w.Graph.RestoreState(stored); err != nil {
			t.Fatal(err)
		}
	}
	narrate(w, true)
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	if stored != nil {
		w.Say("I look around")
	}
	receiveLine(t, text, "the narrated prose")
	for _, turn := range log.seen() {
		if turn.Speak {
			return turn.Voice, w.Graph.ExportState()
		}
	}
	t.Fatal("no turn was spoken")
	return "", nil
}

// The narrator is cast before the opening, in a voice of the gender the
// game chose that speaks its language, and speaks in it from the first turn.
func TestTheNarratorIsCastInTheChosenGender(t *testing.T) {
	voice, _ := narratorVoice(t, AdventureConfig{
		Language:     "de",
		VoiceChoices: map[string]VoiceChoice{VoiceNarrator: VoiceMale},
	}, nil)
	if voice != "udo" {
		t.Errorf("narrated in %q, want udo, the first male voice in German", voice)
	}
}

// Without a cast the narrator takes the first voice of its gender in the
// game's language; left to any, the speaking model's own voice.
func TestAFailedNarratorCastFallsBack(t *testing.T) {
	voice, _ := narratorVoice(t, AdventureConfig{
		Tool:         refusingTool{},
		Language:     "de",
		VoiceChoices: map[string]VoiceChoice{VoiceNarrator: VoiceFemale},
	}, nil)
	if voice != "uta" {
		t.Errorf("narrated in %q, want uta, the first female voice in German", voice)
	}
	voice, _ = narratorVoice(t, AdventureConfig{Tool: refusingTool{}}, nil)
	if voice != "" {
		t.Errorf("narrated in %q, want the speaking model's own voice", voice)
	}
}

// A resumed Adventure keeps its narrator, though a new cast would pick
// another.
func TestAResumedAdventureKeepsItsNarrator(t *testing.T) {
	cfg := AdventureConfig{Tool: lastPickTool{}, VoiceChoices: map[string]VoiceChoice{VoiceNarrator: VoiceFemale}}
	first, stored := narratorVoice(t, cfg, nil)
	if first != "uta" {
		t.Fatalf("narrated in %q, want uta, the last female voice", first)
	}
	cfg.Tool = mock.Tool{}
	again, _ := narratorVoice(t, cfg, stored)
	if again != first {
		t.Errorf("resumed narrating in %q, want the cast kept: %q", again, first)
	}
}

// Left to any, a character without a brief speaks in the tier's own voice.
func TestAFailedNPCBriefLeftToAnyKeepsTheTiersVoice(t *testing.T) {
	voice, made := npcVoice(t, refusingTool{}, VoiceAny)
	if voice != "" || made != 0 {
		t.Errorf("speaks in %q having made %d voices, want the tier's own and none", voice, made)
	}
}

// A resumed NPC-Live game keeps the voice its character was cast in, though
// a new brief would cast another.
func TestAResumedNPCKeepsItsCast(t *testing.T) {
	first, _, stored := npcVoiceFrom(t, lastPickTool{}, VoiceFemale, nil)
	if first != "ann" {
		t.Fatalf("speaks in %q, want ann, the last female voice", first)
	}
	again, _, _ := npcVoiceFrom(t, mock.Tool{}, VoiceFemale, stored)
	if again != first {
		t.Errorf("resumed in %q, want the cast kept: %q", again, first)
	}
}
