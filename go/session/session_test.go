package session

import (
	"testing"

	"engine"
)

func TestGameSpec(t *testing.T) {
	theme := "noir"
	rule := "Keep it gentle."
	game := &gameFacts{
		Genre:        "npc",
		Name:         "Bridge",
		Description:  "Talk your way past a troll.",
		Scenario:     "A troll guards a bridge.",
		GameStart:    "Greet the traveller.",
		ImageStyle:   "woodcut",
		Theme:        &theme,
		StatusFields: `[{"name":"Mood","value":"grumpy"}]`,
		Voices:       map[string]string{"character": "female"},
	}
	spec := gameSpec(game, "de", Rule{Text: &rule})
	want := engine.SessionSpec{
		Genre: engine.GenreNPC, Title: "Bridge", Scenario: "A troll guards a bridge.",
		InitPrompt: "Greet the traveller.", ImageStyle: "woodcut", Theme: "noir", Language: "de",
		Guardrail: rule, Status: map[string]string{"Mood": "grumpy"},
	}
	if spec.Genre != want.Genre || spec.Title != want.Title || spec.Scenario != want.Scenario ||
		spec.InitPrompt != want.InitPrompt || spec.ImageStyle != want.ImageStyle || spec.Theme != want.Theme ||
		spec.Language != want.Language || spec.Guardrail != want.Guardrail || spec.Status["Mood"] != "grumpy" ||
		spec.Description != "Talk your way past a troll." || spec.Voices["character"] != engine.VoiceFemale {
		t.Errorf("spec %+v, want %+v", spec, want)
	}
}

func TestPlayerLanguage(t *testing.T) {
	cases := []struct{ profile, requested, want string }{
		{"fr", "de", "fr"},
		{"", "de", "de"},
		{"", "", "en"},
	}
	for _, c := range cases {
		if got := playerLanguage(c.profile, c.requested); got != c.want {
			t.Errorf("playerLanguage(%q, %q) = %q, want %q", c.profile, c.requested, got, c.want)
		}
	}
}
