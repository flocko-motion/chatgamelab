package genre

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
)

const (
	testScenario  = "A troll guards a bridge over a misty gorge."
	testGuardrail = "Suitable for a 10-year-old."
)

// recordingThread runs the mock thread and keeps every turn it was given.
type recordingThread struct {
	mock.Thread
	mu    sync.Mutex
	turns []adapters.ThreadTurn
}

func (r *recordingThread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	r.mu.Lock()
	r.turns = append(r.turns, turn)
	r.mu.Unlock()
	return r.Thread.Continue(ctx, turn)
}

func (r *recordingThread) seen() []adapters.ThreadTurn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]adapters.ThreadTurn(nil), r.turns...)
}

// recordingTool runs the mock tool and keeps every call, by system prompt.
type recordingTool struct {
	mock.Tool
	mu    sync.Mutex
	calls [][2]string
}

func (r *recordingTool) Query(ctx context.Context, system, user string) (string, adapters.Usage, error) {
	r.mu.Lock()
	r.calls = append(r.calls, [2]string{system, user})
	r.mu.Unlock()
	return r.Tool.Query(ctx, system, user)
}

func (r *recordingTool) seen() [][2]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][2]string(nil), r.calls...)
}

// play starts a wiring, plays the opening and the actions, and waits for each
// one's prose.
func play(t *testing.T, w *Wiring, actions ...string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening's prose")
	for _, action := range actions {
		w.Say(action)
		receiveLine(t, text, "the prose for "+action)
	}
}

// The System Message opens the story thread: the game engine prompt, the
// scenario, how the game starts, and the constraints. The opening is the
// Initialization Prompt, the game's own message; Expand's message is its
// prompt alone.
func TestAdventureSystemMessage(t *testing.T) {
	thread := &recordingThread{}
	w := adventure(t, AdventureConfig{
		Thread: thread, Scenario: testScenario, Guardrail: testGuardrail,
		StartingMessage: "The player wakes on the bridge.",
	})
	play(t, w, "I cross")

	turns := thread.seen()
	if len(turns) != 4 {
		t.Fatalf("%d calls on the thread, want the opening's two and the turn's two", len(turns))
	}
	want := w.Prompts[PromptGameMaster] + "\n\nThe scenario:\n" + testScenario +
		"\n\nHow to start the game:\nThe player wakes on the bridge." +
		"\n\nNARRATION RULES must be respected: " + testGuardrail
	for i, turn := range turns {
		if turn.System != want {
			t.Errorf("call %d's System Message:\n%s\nwant:\n%s", i, turn.System, want)
		}
	}
	opening, openingExpand, action, expand := turns[0], turns[1], turns[2], turns[3]
	if opening.Input != w.Prompts[PromptOpening] || !opening.Directive {
		t.Errorf("the opening sent %+v, want the Initialization Prompt as the game's own message", opening)
	}
	if openingExpand.Input != w.Prompts[PromptOpeningExpand] || expand.Input != w.Prompts[PromptExpand] || !expand.Directive {
		t.Errorf("Expand sent %q then %q, want its prompts alone", openingExpand.Input, expand.Input)
	}
	if action.Directive || !strings.Contains(action.Input, `"input":"[mock-tool] I cross"`) {
		t.Errorf("the action sent %+v, want the rephrased action as the player's", action)
	}

	w = adventure(t, AdventureConfig{Thread: &recordingThread{}, Scenario: testScenario})
	if system := systemMessage(w.Prompts[PromptGameMaster], testScenario, "", ""); strings.Contains(system, "How to start") ||
		strings.Contains(system, "NARRATION RULES") {
		t.Errorf("without a starting message or constraints, the System Message reads:\n%s", system)
	}
}

// The constraints never share a message with designer text where the
// provider has a message of their own for them: every call's constraints are
// the youth-protection rule alone, and no message the game sends carries it.
// Only the System Message holds both, as one text.
func TestAdventureScenarioAndGuardrailStayApart(t *testing.T) {
	thread := &recordingThread{}
	w := adventure(t, AdventureConfig{
		Thread: thread, Scenario: testScenario, Guardrail: testGuardrail,
		StartingMessage: "The player wakes on the bridge.",
	})
	play(t, w, "I cross", "I wait")

	for i, turn := range thread.seen() {
		if turn.Constraints != "NARRATION RULES must be respected: "+testGuardrail {
			t.Errorf("call %d carried constraints %q, want the rule alone", i, turn.Constraints)
		}
		if strings.Contains(turn.Input, testGuardrail) {
			t.Errorf("call %d folded the rule into its message: %q", i, turn.Input)
		}
		if strings.Contains(turn.Constraints, "troll") || strings.Contains(turn.Constraints, "wakes") {
			t.Errorf("call %d's constraints carry designer text: %q", i, turn.Constraints)
		}
	}
}

// The constraint frequency is set per block by name: at 2, Expand's calls
// carry the constraints every other time, counted from its own calls, while
// the opening and Outline's calls carry them every time. An unknown block or
// a frequency out of range refuses the wiring.
func TestAdventureConstraintFrequency(t *testing.T) {
	thread := &recordingThread{}
	w := adventure(t, AdventureConfig{
		Thread: thread, Guardrail: testGuardrail,
		ConstraintFrequency: map[string]int{BlockExpand: 2},
	})
	play(t, w, "one", "two", "three")

	var expand, outline []bool
	for _, turn := range thread.seen() {
		carried := turn.Constraints != ""
		if turn.Directive && strings.HasPrefix(turn.Input, "NARRATE") {
			expand = append(expand, carried)
		} else {
			outline = append(outline, carried)
		}
	}
	if fmt.Sprint(expand) != "[true false true false]" {
		t.Errorf("Expand carried the constraints %v, want every other call", expand)
	}
	if fmt.Sprint(outline) != "[true true true true]" {
		t.Errorf("Outline carried the constraints %v, want every call", outline)
	}

	for _, frequencies := range []map[string]int{{"rephrase": 2}, {BlockExpand: 0}, {BlockOutline: 11}} {
		if _, err := NewAdventure(withMocks(AdventureConfig{ConstraintFrequency: frequencies})); err == nil {
			t.Errorf("frequencies %v were accepted", frequencies)
		}
	}
}

// Preparation condenses a long scenario and adapts the image style to the
// youth-protection rule, before the gate opens; the picture prompt is built
// in v1's order from both, the scene frame around it.
func TestAdventurePreparationShapesThePicture(t *testing.T) {
	tool, image := &recordingTool{}, &recordingImage{}
	long := strings.Repeat("The gorge is deep and the troll is old. ", 15)
	w := adventure(t, AdventureConfig{
		Tool: tool, Image: image, Scenario: long, Guardrail: testGuardrail,
		ImageStyle: "Holzschnitt", Description: "Talk your way past a troll.",
	})
	play(t, w)

	image.expect(t, strings.Join([]string{
		"You are generating scene illustrations for a text-adventure game.",
		"The game idea is: Talk your way past a troll.",
		"The game scenario is: [mock-tool] Scenario:\n" + long,
		"The current scene is: [outline 1] " + w.Prompts[PromptOpening],
		"The visual should show: [image_prompt 1] " + w.Prompts[PromptOpening],
		"The artistic style should be: [mock-tool] Image Style: Holzschnitt\n\nWorkshop Constraints:\n" + testGuardrail,
		"Important: Scenery only, do not depict the player character.",
	}, "\n"))

	asked := map[string]bool{}
	for _, call := range tool.seen() {
		asked[call[0]] = true
	}
	if !asked[w.Prompts[PromptCondenseScenario]] || !asked[w.Prompts[PromptImageStyle]] {
		t.Errorf("preparation asked %v, want both calls", tool.seen())
	}
}

// recordingImage runs the mock image adapter and keeps every prompt.
type recordingImage struct {
	mu      sync.Mutex
	prompts []string
}

func (r *recordingImage) Generate(ctx context.Context, ask adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	r.mu.Lock()
	r.prompts = append(r.prompts, ask.Prompt)
	r.mu.Unlock()
	return mock.Image{}.Generate(ctx, ask)
}

// expect waits for a prompt that is want.
func (r *recordingImage) expect(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		prompts := append([]string(nil), r.prompts...)
		r.mu.Unlock()
		for _, prompt := range prompts {
			if prompt == want {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	t.Fatalf("no picture was asked for as\n%s\nasked:\n%s", want, strings.Join(r.prompts, "\n---\n"))
}

// A short scenario is not condensed, and the picture prompt carries it
// whole; the image style is adapted all the same, whatever the offers allow.
func TestAdventureShortScenarioIsNotCondensed(t *testing.T) {
	tool := &recordingTool{}
	w := adventure(t, AdventureConfig{Tool: tool, Scenario: testScenario})
	play(t, w)
	waitForImagePrompt(t, w, "The game scenario is: "+testScenario+"\n")
	adapted := false
	for _, call := range tool.seen() {
		if call[0] == w.Prompts[PromptCondenseScenario] {
			t.Error("a short scenario was condensed")
		}
		adapted = adapted || call[0] == w.Prompts[PromptImageStyle]
	}
	if !adapted {
		t.Error("the image style was not adapted")
	}
}

// The picture prompt's parts each go only when there is something to say,
// and the style falls back to Adventure's own.
func TestImagePromptOrder(t *testing.T) {
	build := imagePrompt("first line\nlast line", "", "the scenario")
	got := build(blocks.Parts{Text: "a bridge", Secondary: "The player crosses."})
	want := "first line\nThe game scenario is: the scenario\nThe current scene is: The player crosses.\n" +
		"The visual should show: a bridge\nThe artistic style should be: " + AdventureImageStyle + "\nlast line"
	if got != want {
		t.Errorf("prompt:\n%s\nwant:\n%s", got, want)
	}
}

// What the player types before the opening's prose is done is held, and sent
// once it is: the world has the initiative.
func TestAdventurePlayerWaitsForTheOpening(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := &recordingThread{Thread: mock.Thread{Pace: 100 * time.Millisecond}}
	w := adventure(t, AdventureConfig{Thread: thread})
	text := utterances(streamOf(w, "text"))
	w.Graph.Start(ctx)
	w.Say("I act at once")

	receiveLine(t, text, "the opening's prose")
	receiveLine(t, text, "the turn's prose")
	turns := thread.seen()
	if len(turns) != 4 || !strings.Contains(turns[2].Input, "I act at once") {
		t.Errorf("calls ran as %v, want the opening's two before the action", inputs(turns))
	}
}

func inputs(turns []adapters.ThreadTurn) []string {
	out := make([]string, len(turns))
	for i, turn := range turns {
		out[i] = turn.Input
	}
	return out
}

// errHard is a failure that will not pass, so no retry slows a test.
var errHard = errors.New("the provider refused the request")

// failingTool fails the calls whose system prompt it names.
type failingTool struct {
	mock.Tool
	system string
}

func (f failingTool) Query(ctx context.Context, system, user string) (string, adapters.Usage, error) {
	if system == f.system {
		return "", adapters.Usage{}, errHard
	}
	return f.Tool.Query(ctx, system, user)
}

// failingThread fails the calls whose message passes the test.
type failingThread struct {
	mock.Thread
	fails func(adapters.ThreadTurn) bool
}

func (f failingThread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	if f.fails(turn) {
		return adapters.ThreadReply{}, errHard
	}
	return f.Thread.Continue(ctx, turn)
}

type failingImage struct{}

func (failingImage) Generate(context.Context, adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	return nil, adapters.Usage{}, errHard
}

// sinkNamed is the arrivals of the sink of that name.
func sinkNamed(t *testing.T, w *Wiring, name string) chan string {
	t.Helper()
	for _, sink := range w.Sinks() {
		if sink.NodeName() == name {
			return sink.Arrivals()
		}
	}
	t.Fatalf("no sink named %s", name)
	return nil
}

// Every block that calls a model leads its failure where the genre says: a
// failed call during play reaches an error sink with its code and the text
// the block was given, and the game goes on; a failed preparation call
// reaches the problem sink.
func TestAdventureFailureEdges(t *testing.T) {
	rephrase := AdventurePrompts[PromptRephrase]
	for _, tc := range []struct {
		name string
		cfg  AdventureConfig
		// sink is where the failure goes; want what it receives first.
		sink, want string
	}{
		{
			name: "rephrase",
			cfg:  AdventureConfig{Tool: failingTool{system: strings.TrimSuffix(rephrase, "{{with .Language}}\nKeep the response in {{.}}.{{end}}")}},
			sink: "turn-failed", want: `{"code":"turn_failed","text":"I cross"}`,
		},
		{
			name: "outline",
			cfg: AdventureConfig{Thread: failingThread{fails: func(turn adapters.ThreadTurn) bool {
				return turn.Schema != nil && !turn.Directive
			}}},
			sink: "turn-failed", want: `{"code":"turn_failed","text":"[mock-tool] I cross"}`,
		},
		{
			name: "expand",
			cfg: AdventureConfig{Thread: failingThread{fails: func(turn adapters.ThreadTurn) bool {
				return strings.HasPrefix(turn.Input, "NARRATE the outline")
			}}},
			sink: "turn-failed", want: `{"code":"turn_failed","text":"[outline 3] [mock-tool] I cross"}`,
		},
		{
			name: "image",
			cfg:  AdventureConfig{Image: failingImage{}},
			sink: "image-failed", want: `{"code":"image_failed","text":"You are generating scene illustrations`,
		},
		{
			name: "image-style",
			cfg:  AdventureConfig{Tool: failingTool{system: AdventurePrompts[PromptImageStyle]}},
			sink: "problem", want: `{"node":"image-style","reason":"failed",`,
		},
		{
			name: "condense-scenario",
			cfg:  AdventureConfig{Tool: failingTool{system: AdventurePrompts[PromptCondenseScenario]}, Scenario: strings.Repeat("long ", 120)},
			sink: "problem", want: `{"node":"condense-scenario","reason":"failed",`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			w := adventure(t, tc.cfg)
			if err := w.Graph.Validate(); err != nil {
				t.Fatal(err)
			}
			failed := sinkNamed(t, w, tc.sink)
			w.Graph.Start(ctx)
			w.Say("I cross")
			if got := receiveLine(t, failed, "the failure"); !strings.HasPrefix(got, tc.want) {
				t.Errorf("%s received %q, want it to start %q", tc.sink, got, tc.want)
			}
		})
	}
}

// An opening that failed leaves Expand on its turn prompt: the opening prompt
// follows the opening's cue only.
func TestAdventureExpandAfterAFailedOpening(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := &recordingThread{}
	w := adventure(t, AdventureConfig{Thread: failingThread{Thread: mock.Thread{}, fails: func(turn adapters.ThreadTurn) bool {
		if turn.Schema != nil && turn.Directive {
			return true
		}
		_, _ = thread.Continue(ctx, turn)
		return false
	}}})
	text, failures := utterances(streamOf(w, "text")), sinkNamed(t, w, "turn-failed")
	w.Graph.Start(ctx)
	receiveLine(t, failures, "the failed opening")
	w.Say("I cross")
	receiveLine(t, text, "the turn's prose")
	var expands []string
	for _, turn := range thread.seen() {
		if turn.Schema == nil {
			expands = append(expands, turn.Input)
		}
	}
	if len(expands) != 1 || expands[0] != w.Prompts[PromptExpand] {
		t.Errorf("Expand sent %q, want its turn prompt", expands)
	}
}

// After a failed turn the game goes on: the next action runs through.
func TestAdventureGoesOnAfterAFailedTurn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var once sync.Once
	failFirst := func(turn adapters.ThreadTurn) bool {
		failed := false
		if turn.Schema != nil && !turn.Directive {
			once.Do(func() { failed = true })
		}
		return failed
	}
	w := adventure(t, AdventureConfig{Thread: failingThread{fails: failFirst}})
	text, failures := utterances(streamOf(w, "text")), sinkNamed(t, w, "turn-failed")
	w.Graph.Start(ctx)
	receiveLine(t, text, "the opening's prose")
	w.Say("I cross")
	receiveLine(t, failures, "the failed turn")
	w.Say("I cross again")
	if got := receiveLine(t, text, "the next turn's prose"); !strings.HasPrefix(got, "[story turn 4]") {
		t.Errorf("the next turn said %q", got)
	}
}
