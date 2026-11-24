package api_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"engine"
	"engine/api"
)

// platformsJSON is the platform list as the platform answers it.
func platformsJSON(t *testing.T, a *api.API) []byte {
	t.Helper()
	blob, err := json.Marshal(a.PlatformList())
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

// The platform answers this list and the game editor renders it, so the JSON
// shape is a contract: checked here as raw JSON, not through the Go types.
func TestPlatformsJSON(t *testing.T) {
	blob := platformsJSON(t, api.New(api.Options{Dev: func() bool { return true }}))

	type model struct {
		Name   string            `json:"name"`
		Params map[string]string `json:"params"`
	}
	var platforms []struct {
		ID       string          `json:"id"`
		Name     string          `json:"name"`
		UserKeys *bool           `json:"userKeys"`
		Toggles  map[string]bool `json:"toggles"`
		Genres   []string        `json:"genres"`
		Tiers    []struct {
			ID     string           `json:"id"`
			Label  string           `json:"label"`
			Models map[string]model `json:"models"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(blob, &platforms); err != nil {
		t.Fatal(err)
	}
	if len(platforms) != 3 {
		t.Fatalf("got %d platforms, want openai, mistral and mock", len(platforms))
	}

	openai := platforms[0]
	if openai.ID != "openai" || openai.Name != "OpenAI" || openai.UserKeys == nil || !*openai.UserKeys {
		t.Errorf("openai header = %+v", openai)
	}
	for _, toggle := range []string{"images", "audio-out", "audio-in", "audio-live"} {
		if _, stated := openai.Toggles[toggle]; !stated {
			t.Errorf("toggle %q not stated", toggle)
		}
	}
	if len(openai.Genres) != 2 || openai.Genres[0] != "adventure" || openai.Genres[1] != "npc-live" {
		t.Errorf("genres = %v", openai.Genres)
	}

	if len(openai.Tiers) != 3 {
		t.Fatalf("tiers = %+v", openai.Tiers)
	}
	labels := map[string]string{"low": "Economy", "medium": "Balanced", "high": "High"}
	for _, tier := range openai.Tiers {
		if labels[tier.ID] != tier.Label {
			t.Errorf("tier %q labelled %q", tier.ID, tier.Label)
		}
	}
	image := openai.Tiers[2].Models["image"]
	if image.Name == "" || image.Params["size"] == "" || image.Params["quality"] != "high" {
		t.Errorf("high image = %+v", image)
	}

	mistral := platforms[1]
	if mistral.Toggles["audio-live"] || len(mistral.Genres) != 1 {
		t.Errorf("mistral = %+v, want no live voice and adventure only", mistral)
	}
	if mock := platforms[2]; mock.ID != "mock" || mock.UserKeys == nil || *mock.UserKeys {
		t.Errorf("mock = %+v, want no user keys", mock)
	}
}

// A production platform must not offer the mock.
func TestPlatformsLeavesMockOutOfProduction(t *testing.T) {
	var platforms []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(platformsJSON(t, api.New(api.Options{})), &platforms); err != nil {
		t.Fatal(err)
	}
	for _, p := range platforms {
		if p.ID == "mock" {
			t.Fatal("mock listed outside dev mode")
		}
	}
	if len(platforms) == 0 {
		t.Fatal("no platforms listed")
	}
}

// The game editor reads the themes and genres the platform answers, so their
// JSON shape is a contract.
func TestListsJSON(t *testing.T) {
	var themes []map[string]any
	remarshal(t, engine.Themes(), &themes)
	if len(themes) == 0 || themes[0]["id"] != "default" {
		t.Errorf("themes = %v, want the default first", themes)
	}

	var genres []struct {
		ID    string   `json:"id"`
		Needs []string `json:"needs"`
	}
	remarshal(t, engine.GenreList(), &genres)
	want := map[string][]string{"adventure": {}, "npc-live": {"audio-live"}}
	if len(genres) != len(want) {
		t.Fatalf("genres = %+v", genres)
	}
	for _, g := range genres {
		needs, known := want[g.ID]
		if !known || fmt.Sprint(g.Needs) != fmt.Sprint(needs) || g.Needs == nil {
			t.Errorf("genre %+v, want needs %v", g, needs)
		}
	}
}

func remarshal(t *testing.T, v, into any) {
	t.Helper()
	blob, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(blob, into); err != nil {
		t.Fatal(err)
	}
}
