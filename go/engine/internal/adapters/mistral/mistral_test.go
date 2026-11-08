package mistral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engine/internal/adapters"
)

func key(context.Context) (string, error) { return "test-key", nil }

// fake answers /conversations with reply and serves one file, recording the
// last conversation request it saw.
func fake(t *testing.T, reply string, seen *start) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/conversations":
			if err := json.NewDecoder(r.Body).Decode(seen); err != nil {
				t.Errorf("decode request: %v", err)
			}
			_, _ = w.Write([]byte(reply))
		case "/files/file-1/content":
			_, _ = w.Write([]byte("PNG"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestToolSendsPromptAndReportsUsage(t *testing.T) {
	var seen start
	srv := fake(t, `{"outputs":[{"role":"assistant","content":"in character"}],
		"usage":{"prompt_tokens":12,"completion_tokens":3}}`, &seen)

	tool := NewTool(key, "mistral-small-latest")
	tool.api.base = srv.URL

	text, used, err := tool.Query(context.Background(), "judge", "a line")
	if err != nil {
		t.Fatal(err)
	}
	if text != "in character" {
		t.Errorf("text = %q", text)
	}
	if used != (adapters.Usage{Model: "mistral-small-latest", InputTokens: 12, OutputTokens: 3}) {
		t.Errorf("usage = %+v", used)
	}
	if seen.Instructions != "judge" || len(seen.Inputs) != 1 || seen.Inputs[0].Content != "a line" || seen.Store {
		t.Errorf("request = %+v", seen)
	}
}

func TestToolReadsChunkedContent(t *testing.T) {
	var seen start
	srv := fake(t, `{"outputs":[{"role":"assistant","content":[
		{"type":"text","text":"in "},{"type":"text","text":"character"}]}]}`, &seen)

	tool := NewTool(key, "m")
	tool.api.base = srv.URL

	text, _, err := tool.Query(context.Background(), "s", "u")
	if err != nil {
		t.Fatal(err)
	}
	if text != "in character" {
		t.Errorf("text = %q", text)
	}
}

func TestImageDownloadsTheToolFile(t *testing.T) {
	var seen start
	srv := fake(t, `{"outputs":[{"role":"assistant","content":[
		{"type":"tool_file","tool":"image_generation","file_id":"file-1"}]}],
		"usage":{"prompt_tokens":40,"completion_tokens":5}}`, &seen)

	image := NewImage(key, "mistral-small-latest")
	image.api.base = srv.URL

	got, used, err := image.Generate(context.Background(), adapters.ImageRequest{Prompt: "a bridge"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "PNG" {
		t.Errorf("image = %q", got)
	}
	if used.Images != 1 || used.InputTokens != 40 {
		t.Errorf("usage = %+v", used)
	}
	if len(seen.Tools) != 1 || seen.Tools[0].Type != "image_generation" {
		t.Errorf("tools = %+v", seen.Tools)
	}
}

func TestImageWithoutFileFails(t *testing.T) {
	var seen start
	srv := fake(t, `{"outputs":[{"role":"assistant","content":"I cannot draw that."}]}`, &seen)

	image := NewImage(key, "m")
	image.api.base = srv.URL

	if _, _, err := image.Generate(context.Background(), adapters.ImageRequest{Prompt: "x"}); err == nil {
		t.Fatal("want an error when no file comes back")
	}
}
