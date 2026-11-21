package genre

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"engine/internal/adapters/mock"
)

// Adventure has its eight prompts, each under its own name.
func TestAdventureHasItsPrompts(t *testing.T) {
	want := []string{
		"condense-scenario", "expand", "game-master", "image-style",
		"opening", "opening-expand", "rephrase", "scene",
	}
	if got := slices.Sorted(maps.Keys(AdventurePrompts)); !slices.Equal(got, want) {
		t.Errorf("Adventure's prompts are %v, want %v", got, want)
	}
	w := mustAdventure(t, nil)
	if got := slices.Sorted(maps.Keys(w.Prompts)); !slices.Equal(got, want) {
		t.Errorf("the wiring runs on %v, want %v", got, want)
	}
}

// withMocks runs the config on the mock platform's adapters, with pictures
// allowed, unless the config sets adapters of its own.
func withMocks(cfg AdventureConfig) AdventureConfig {
	if cfg.Tool == nil {
		cfg.Tool = mock.Tool{}
	}
	if cfg.Thread == nil {
		cfg.Thread = mock.Thread{}
	}
	if cfg.Image == nil {
		cfg.Image = mock.Image{}
	}
	return cfg
}

func adventure(t testing.TB, cfg AdventureConfig) *Wiring {
	t.Helper()
	w, err := NewAdventure(withMocks(cfg))
	if err != nil {
		t.Fatalf("wiring adventure: %v", err)
	}
	return w
}

// The language is named in itself in the prompts that
// write for the player.
func TestAdventurePromptsNameThePlayersLanguage(t *testing.T) {
	w := adventure(t, AdventureConfig{Language: "de"})
	for name, want := range map[string]string{
		PromptRephrase:      "\nKeep the response in Deutsch.",
		PromptExpand:        "NARRATE the outline into prose in the players language (Deutsch). ",
		PromptOpeningExpand: "NARRATE the opening scene into prose in the players language (Deutsch). ",
	} {
		if !strings.Contains(w.Prompts[name], want) {
			t.Errorf("%s lacks %q: %q", name, want, w.Prompts[name])
		}
	}

	w = adventure(t, AdventureConfig{})
	if got := w.Prompts[PromptRephrase]; strings.Contains(got, "Keep the response") {
		t.Errorf("without a language, rephrase still names one: %q", got)
	}
	if got := w.Prompts[PromptExpand]; !strings.HasPrefix(got, "NARRATE the outline into prose in the players language. ") {
		t.Errorf("without a language, expand reads %q", got)
	}
}

// game-master shows an action in the shape Outline sends it, with the game's
// own status fields.
func TestGameMasterShowsTheGamesStatus(t *testing.T) {
	w := adventure(t, AdventureConfig{Status: map[string]string{"Health": "100", "Gold": "5"}})
	want := "A player action arrives like this:\n" + `{"input":"drink the potion","status":{"Gold":"5","Health":"100"}}` + "\n"
	if !strings.Contains(w.Prompts[PromptGameMaster], want) {
		t.Errorf("game-master lacks %q:\n%s", want, w.Prompts[PromptGameMaster])
	}

	w = adventure(t, AdventureConfig{})
	want = "A player action arrives like this:\n" + `{"input":"drink the potion"}` + "\n"
	if !strings.Contains(w.Prompts[PromptGameMaster], want) {
		t.Errorf("without status fields game-master lacks %q:\n%s", want, w.Prompts[PromptGameMaster])
	}
}

// The opening asks for initial status values only when the game has status
// fields, and carries the dice it is given.
func TestOpeningFollowsTheGame(t *testing.T) {
	const statusSentence = "Set the status fields to good initial values for the scenario."

	w := adventure(t, AdventureConfig{Status: map[string]string{"Health": "100"}, Dice: []int{7, 42, 99, 1, 63}})
	opening := w.Prompts[PromptOpening]
	if !strings.HasPrefix(opening, "Start the game. Generate the opening scene. "+statusSentence+` The plot outline in "outline" must`) {
		t.Errorf("opening with status fields reads %q", opening)
	}
	if !strings.Contains(opening, "\n\nRandom values for this playthrough: 7, 42, 99, 1, 63 (each 1-100). ") {
		t.Errorf("opening lacks the dice: %q", opening)
	}

	w = adventure(t, AdventureConfig{})
	opening = w.Prompts[PromptOpening]
	if strings.Contains(opening, statusSentence) {
		t.Errorf("opening without status fields asks for them: %q", opening)
	}
	if strings.Contains(opening, "Random values") {
		t.Errorf("opening without dice mentions them: %q", opening)
	}
}

// A spec replaces a prompt by name; the override may use the same fields,
// and a misspelt name or a broken template refuses the wiring.
func TestAdventurePromptOverride(t *testing.T) {
	w := adventure(t, AdventureConfig{Language: "fr", PromptOverride: map[string]string{
		PromptRephrase: "Say it as a pirate, in {{.Language}}.",
		PromptExpand:   "Tell it in verse.",
	}})
	if got := w.Prompts[PromptRephrase]; got != "Say it as a pirate, in Français." {
		t.Errorf("rephrase override reads %q", got)
	}
	if got := w.Prompts[PromptExpand]; got != "Tell it in verse." {
		t.Errorf("expand override reads %q", got)
	}
	if w.Prompts[PromptGameMaster] == "" {
		t.Error("overriding two prompts dropped the others")
	}

	for _, override := range []map[string]string{
		{"outline": "x"},
		{PromptExpand: "Tell it {{.Rhyme}}."},
		{PromptExpand: "Tell it {{."},
	} {
		if _, err := NewAdventure(withMocks(AdventureConfig{PromptOverride: override})); err == nil {
			t.Errorf("override %v was accepted", override)
		}
	}
}

// Outline's schema describes its fields to the model.
func TestOutlineSchemaDescribesItsFields(t *testing.T) {
	properties := OutlineSchema(map[string]string{"Health": "100"}).Body["properties"].(map[string]any)
	for field, want := range map[string]string{
		"outline":      "Plot outline, just the raw plot - no coloring",
		"status":       "Updated status fields after the action",
		"image_prompt": "Vivid description of the scene for image generation",
	} {
		if got := properties[field].(map[string]any)["description"]; got != want {
			t.Errorf("%s is described as %q, want %q", field, got, want)
		}
	}
}

// Languages are named as the platform lists them.
func TestAdventureNamesLanguagesAsThePlatformDoes(t *testing.T) {
	for tag, want := range map[string]string{"fr": "Français", "bar": "Bayrisch", "no": "Norsk", "xx": "xx"} {
		w := adventure(t, AdventureConfig{Language: tag})
		if got := w.Prompts[PromptRephrase]; !strings.HasSuffix(got, "Keep the response in "+want+".") {
			t.Errorf("%s: rephrase ends %q, want the language named %s", tag, got, want)
		}
	}
}
