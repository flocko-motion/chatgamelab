package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engine/internal/adapters"
)

// A look at a picture sends the words, then the picture inline, and asks for
// JSON of the schema.
func TestToolShowsAPicture(t *testing.T) {
	var seen map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&seen)
		_, _ = w.Write([]byte(`{"output":[{"content":[]},{"content":[{"text":"{\"ok\":true}"}]}],"usage":{}}`))
	}))
	defer srv.Close()
	old := ResponsesURL
	ResponsesURL = srv.URL
	defer func() { ResponsesURL = old }()

	png := []byte("\x89PNG\r\n\x1a\nrest")
	tool := NewTool(func(context.Context) (string, error) { return "k", nil }, "gpt-test")
	text, _, err := tool.QueryImage(context.Background(), "judge", "the picture:", png, adapters.Schema{Name: "v", Body: map[string]any{"type": "object"}})
	if err != nil || text != `{"ok":true}` {
		t.Fatalf("text %q, err %v", text, err)
	}
	input, _ := seen["input"].([]any)
	if len(input) != 1 {
		t.Fatalf("input = %#v", seen["input"])
	}
	parts, _ := input[0].(map[string]any)["content"].([]any)
	if len(parts) != 2 {
		t.Fatalf("parts = %#v", parts)
	}
	first, _ := parts[0].(map[string]any)
	second, _ := parts[1].(map[string]any)
	if first["type"] != "input_text" || first["text"] != "the picture:" || second["type"] != "input_image" || second["image_url"] != adapters.DataURL(png) {
		t.Errorf("parts = %#v", parts)
	}
	if seen["text"] == nil {
		t.Error("no format asked for")
	}
}
