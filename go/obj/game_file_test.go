package obj

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/uuid"
)

// The game file keeps the field names files exported before carry.
func TestGameFileFieldNames(t *testing.T) {
	theme := "dark"
	file := NewGameFile(Game{
		ID: uuid.New(), Name: "n", Description: "d", Public: true,
		SystemMessageScenario: "s", SystemMessageGameStart: "g", ImageStyle: "i",
		CSS: "c", StatusFields: "f", Genre: "adventure", Theme: &theme,
	})
	raw, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	want := []string{"css", "description", "genre", "id", "image_style", "name",
		"status_fields", "system_message_game_start", "system_message_scenario", "theme"}
	var got []string
	for k := range fields {
		got = append(got, k)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("fields %v, want %v", got, want)
	}
	if fields["system_message_scenario"] != "s" || fields["theme"] != "dark" {
		t.Errorf("values not carried: %v", fields)
	}
}
