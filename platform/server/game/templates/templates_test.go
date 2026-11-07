package templates

import (
	"strings"
	"testing"

	"cgl/obj"
)

func TestGetTemplateLeavesNoPlaceholders(t *testing.T) {
	for name, gameStart := range map[string]string{"with game start": "Der Spieler steht vor einer Villa.", "without game start": ""} {
		t.Run(name, func(t *testing.T) {
			game := &obj.Game{SystemMessageScenario: "Ein Einbruch.", SystemMessageGameStart: gameStart}
			got, err := GetTemplate(game, "de")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(got, "{{") {
				t.Errorf("unreplaced placeholder in system prompt:\n%s", got)
			}
			if hasStart := strings.Contains(got, "How to start the game:"); hasStart != (gameStart != "") {
				t.Errorf("game start section present = %v, want %v", hasStart, gameStart != "")
			}
		})
	}
}

func TestPromptNarrateOpeningScene(t *testing.T) {
	rules := "keine Gewalt"
	got := PromptNarrateOpeningScene("de", &rules)

	for _, want := range []string{"Deutsch", "4-6 sentences", "Do not list choices", "explain it plainly", "NARRATION RULES must be respected: keine Gewalt"} {
		if !strings.Contains(got, want) {
			t.Errorf("opening prompt missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "%!") {
		t.Errorf("format error in opening prompt:\n%s", got)
	}
	if got == PromptNarratePlotOutline("de", &rules) {
		t.Error("opening prompt must differ from the regular narration prompt")
	}
}
