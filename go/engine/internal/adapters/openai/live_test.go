package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engine/internal/adapters"
)

// The shape a successful creation answers with, as documented. Both fields are
// nested a level down, and reading them from the top costs a session that has
// already been created and billed — which is how this was first found.
func TestReadCreatedSession(t *testing.T) {
	body := []byte(`{
	  "session": {"id": "live_123", "model": "gpt-live-1"},
	  "transport": {"type": "webrtc", "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\n"}
	}`)

	id, answer, err := readCreated(body)
	if err != nil {
		t.Fatalf("a documented response was refused: %v", err)
	}
	if id != "live_123" {
		t.Errorf("session id read as %q", id)
	}
	if answer == "" {
		t.Error("the SDP answer was not read, so a browser would get nothing to apply")
	}
}

// A response missing either half is reported with the body attached, because
// the next surprise should be readable rather than guessed at.
func TestReadCreatedSessionReportsWhatWasMissing(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"no id", `{"transport":{"sdp":"v=0"}}`, "session id"},
		{"no answer", `{"session":{"id":"live_1"}}`, "SDP answer"},
		{"neither", `{}`, "session id or SDP answer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := readCreated([]byte(tc.body))
			if err == nil {
				t.Fatal("an unusable response was accepted")
			}
			if !contains(err.Error(), tc.want) {
				t.Errorf("error does not name what was missing: %v", err)
			}
			if !contains(err.Error(), "body:") {
				t.Errorf("error carries no body to read: %v", err)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// A resume must survive a fork it cannot have.
//
// Forking continues a stored recording, and a recording exists only where the
// session closed gracefully. A dropped connection or a killed process leaves
// none — which are the very moments somebody wants to carry on from — so the
// provider answers "the stored session was not found" and the resume has to
// carry on by seeding a fresh conversation with what was said instead.
func TestResumeFallsBackWhenThereIsNoRecording(t *testing.T) {
	var asked []string
	var seeded createRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)

		if strings.HasSuffix(r.URL.Path, "/fork") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"The stored session was not found."}}`))
			return
		}

		_ = json.NewDecoder(r.Body).Decode(&seeded)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"session":{"id":"live_new"},"transport":{"sdp":"v=0 answer"}}`))
	}))
	defer server.Close()

	live := NewLive(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-live-1")
	live.sessions = server.URL

	id, answer, err := live.create(context.Background(), "sk-test", adapters.LiveConfig{
		Offer:      "v=0 offer",
		ResumeFrom: "live_gone",
		Scenario:   "You are the keeper of a bridge.",
		Guardrail:  "Suitable for a 13-year-old.",
		History: []adapters.Utterance{
			{Speaker: adapters.SpeakerCharacter, Text: "You shall not pass."},
			{Speaker: adapters.SpeakerPlayer, Text: "I have gold."},
		},
	}, "verse")
	if err != nil {
		t.Fatalf("a resume with no recording failed instead of falling back: %v", err)
	}
	if id != "live_new" || answer == "" {
		t.Errorf("got id %q answer %q", id, answer)
	}

	if len(asked) != 2 || !strings.HasSuffix(asked[0], "/fork") {
		t.Fatalf("expected a fork and then a fresh session, got %v", asked)
	}

	// The character has to remember, or the fallback costs the game what the
	// saving was meant to protect.
	if len(seeded.Session.Input) < 3 {
		t.Fatalf("the fresh session carried %d messages; want the guardrail and both speakers", len(seeded.Session.Input))
	}
	var said []string
	for _, m := range seeded.Session.Input {
		for _, part := range m.Content {
			said = append(said, m.Role+":"+part.Text)
		}
	}
	joined := strings.Join(said, " | ")
	if !strings.Contains(joined, "assistant:You shall not pass.") {
		t.Errorf("the character's own words were not carried: %s", joined)
	}
	if !strings.Contains(joined, "user:I have gold.") {
		t.Errorf("the player's words were not carried: %s", joined)
	}
	// The platform's constraint travels as its own developer message, never
	// folded into the scenario.
	if !strings.Contains(joined, "developer:Suitable for a 13-year-old.") {
		t.Errorf("the guardrail was not carried as a developer message: %s", joined)
	}
}

// createRequest is the body this adapter sends, for a test to read back.
type createRequest struct {
	Session struct {
		Input []message `json:"input"`
	} `json:"session"`
}
