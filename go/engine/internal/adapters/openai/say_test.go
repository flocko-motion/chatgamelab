package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

// A fake GPT-Live socket: it starts the session, speaks two chunks once it is
// asked to say a line, then falls silent; on close it reports the usage.
func TestSayCollectsTheVoiceUntilItFallsQuiet(t *testing.T) {
	var start, line map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("authorization %q", got)
		}
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer ws.CloseNow()
		ctx := r.Context()
		send := func(event map[string]any) {
			blob, _ := json.Marshal(event)
			_ = ws.Write(ctx, websocket.MessageText, blob)
		}
		for {
			_, data, err := ws.Read(ctx)
			if err != nil {
				return
			}
			var event map[string]any
			_ = json.Unmarshal(data, &event)
			switch event["type"] {
			case "session.start":
				start = event
				send(map[string]any{"type": "session.started"})
			case "session.instructions.append":
				line = event
				for _, chunk := range []string{"ab", "cd"} {
					send(map[string]any{"type": "session.output_audio.delta", "delta": base64.StdEncoding.EncodeToString([]byte(chunk))})
				}
				send(map[string]any{"type": "session.usage.updated", "usage": map[string]any{"seconds": 2}})
			case "session.close":
				send(map[string]any{"type": "session.closed", "reason": "close_requested", "usage": map[string]any{"seconds": 3}})
				return
			}
		}
	}))
	defer srv.Close()
	old := LiveSocketURL
	LiveSocketURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	defer func() { LiveSocketURL = old }()

	live := NewLive(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-live-1")
	pcm, used, err := live.Say(context.Background(), adapters.SayRequest{Voice: "voice_abc", Instructions: "Read it.", Line: "Hello there."})
	if err != nil {
		t.Fatalf("Say: %v", err)
	}
	if string(pcm) != "abcd" {
		t.Errorf("audio %q, want both chunks", pcm)
	}
	if used.AudioSeconds != 3 || used.Model != "gpt-live-1" {
		t.Errorf("usage %+v, want the final 3 s on gpt-live-1", used)
	}
	session, _ := start["session"].(map[string]any)
	voice := session["audio"].(map[string]any)["output"].(map[string]any)["voice"]
	if id, _ := voice.(map[string]any)["id"].(string); id != "voice_abc" {
		t.Errorf("voice %v, want the made voice by id", voice)
	}
	if content, _ := line["content"].(string); !strings.Contains(content, "Hello there.") {
		t.Errorf("the line was not asked for: %v", line)
	}
}
