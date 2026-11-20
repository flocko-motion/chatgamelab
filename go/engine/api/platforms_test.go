package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engine/api"
)

// The platform mounts this handler and renders what it returns, so the JSON
// shape is a contract: checked here as raw JSON, not through the Go types.
func TestPlatformsJSON(t *testing.T) {
	srv := httptest.NewServer(api.New(api.Options{Dev: true}).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/platforms")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content type %q", ct)
	}

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
	if err := json.NewDecoder(resp.Body).Decode(&platforms); err != nil {
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
	srv := httptest.NewServer(api.New(api.Options{}).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/platforms")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var platforms []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&platforms); err != nil {
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
