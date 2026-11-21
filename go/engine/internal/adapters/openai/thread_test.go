package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
)

// fakeResponses answers the Responses API as a stream, recording each request.
type fakeResponses struct {
	mu       sync.Mutex
	requests []map[string]any
	// answer writes the response to the nth request.
	answer func(n int, w http.ResponseWriter)
}

func (f *fakeResponses) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("unreadable request: %v", err)
		}
		f.mu.Lock()
		f.requests = append(f.requests, body)
		n := len(f.requests)
		f.mu.Unlock()
		f.answer(n, w)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// streamed writes a documented stream: deltas, then the completed response.
func streamed(w http.ResponseWriter, id string, pieces ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\""+id+"\"}}\n\n")
	for _, p := range pieces {
		delta, _ := json.Marshal(map[string]string{"type": "response.output_text.delta", "delta": p})
		fmt.Fprintf(w, "event: response.output_text.delta\ndata: %s\n\n", delta)
	}
	fmt.Fprintf(w, "event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":%q,"+
		"\"usage\":{\"input_tokens\":120,\"output_tokens\":30,\"input_tokens_details\":{\"cached_tokens\":100}}}}\n\n", id)
}

func testThread(srv *httptest.Server) *Thread {
	th := NewThread(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-5-mini")
	th.url = srv.URL
	return th
}

func TestThreadStreamsAndContinues(t *testing.T) {
	fake := &fakeResponses{answer: func(n int, w http.ResponseWriter) {
		streamed(w, fmt.Sprintf("resp_%d", n), "The bridge ", "sways.")
	}}
	th := testThread(fake.serve(t))

	var pieces []string
	reply, err := th.Continue(context.Background(), adapters.ThreadTurn{
		System:       "You are a game master.",
		Input:        "The player crosses.",
		Continuation: adapters.Continuation{ID: "resp_0", Model: "gpt-5-mini", Made: time.Now()},
		Piece:        func(p string) { pieces = append(pieces, p) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "The bridge sways." || strings.Join(pieces, "|") != "The bridge |sways." {
		t.Errorf("reply %q, pieces %q", reply.Text, pieces)
	}
	if reply.Continuation.ID != "resp_1" || reply.Continuation.Model != "gpt-5-mini" || reply.Continuation.Made.IsZero() {
		t.Errorf("continuation = %+v", reply.Continuation)
	}
	if reply.Usage.InputTokens != 20 || reply.Usage.CachedInputTokens != 100 || reply.Usage.OutputTokens != 30 {
		t.Errorf("usage = %+v, want cached input apart", reply.Usage)
	}

	sent := fake.requests[0]
	input, _ := json.Marshal(sent["input"])
	if sent["previous_response_id"] != "resp_0" || string(input) != `[{"content":"The player crosses.","role":"user"}]` {
		t.Errorf("request = %v, input %s, want the turn continued from resp_0", sent, input)
	}
	// The continued thread holds the System Message already: it is sent once.
	if _, sent := sent["instructions"]; sent {
		t.Errorf("request = %v, want no instructions", sent)
	}
	if sent["store"] != true || sent["stream"] != true {
		t.Errorf("request = %v, want store and stream", sent)
	}
}

// The first call opens the thread with the System Message as a developer
// message, not as instructions, which OpenAI drops over a previous response.
// The constraints go in a developer message of their own, right before the
// call's message, and the game's own message is a developer message too.
func TestThreadOpensWithTheSystemMessage(t *testing.T) {
	fake := &fakeResponses{answer: func(n int, w http.ResponseWriter) { streamed(w, "resp_1", "{}") }}
	th := testThread(fake.serve(t))

	_, err := th.Continue(context.Background(), adapters.ThreadTurn{
		System:      "You are a game master. The scenario: a bridge.",
		Input:       "Start the game.",
		Directive:   true,
		Constraints: "NARRATION RULES must be respected: keep it gentle",
	})
	if err != nil {
		t.Fatal(err)
	}
	sent := fake.requests[0]
	if _, has := sent["instructions"]; has {
		t.Errorf("request = %v, want the System Message as a message", sent)
	}
	got, _ := json.Marshal(sent["input"])
	want := `[{"content":"You are a game master. The scenario: a bridge.","role":"developer"},` +
		`{"content":"NARRATION RULES must be respected: keep it gentle","role":"developer"},` +
		`{"content":"Start the game.","role":"developer"}]`
	if string(got) != want {
		t.Errorf("input = %s\nwant %s", got, want)
	}
}

// A player's action on a continued thread goes as the user's message, its
// constraints in a developer message of their own before it, and a call
// without constraints sends none.
func TestThreadConstraintsTravelApart(t *testing.T) {
	fake := &fakeResponses{answer: func(n int, w http.ResponseWriter) { streamed(w, fmt.Sprintf("resp_%d", n), "ok") }}
	th := testThread(fake.serve(t))
	from := adapters.Continuation{ID: "resp_0", Model: "gpt-5-mini", Made: time.Now()}

	for _, turn := range []adapters.ThreadTurn{
		{Input: `{"input":"I cross"}`, Constraints: "NARRATION RULES must be respected: no gore", Continuation: from},
		{Input: `{"input":"I wait"}`, Continuation: from},
	} {
		if _, err := th.Continue(context.Background(), turn); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := json.Marshal(fake.requests[0]["input"])
	if string(first) != `[{"content":"NARRATION RULES must be respected: no gore","role":"developer"},`+
		`{"content":"{\"input\":\"I cross\"}","role":"user"}]` {
		t.Errorf("constrained input = %s", first)
	}
	second, _ := json.Marshal(fake.requests[1]["input"])
	if string(second) != `[{"content":"{\"input\":\"I wait\"}","role":"user"}]` {
		t.Errorf("unconstrained input = %s", second)
	}
}

// A previous response OpenAI no longer has is taken over by a fresh thread
// seeded with the transcript, so the session goes on.
func TestThreadReseedsWhenTheResponseIsGone(t *testing.T) {
	fake := &fakeResponses{answer: func(n int, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":"previous_response_not_found","message":"Previous response with id 'resp_old' not found."}}`)
			return
		}
		streamed(w, "resp_new", "Again.")
	}}
	th := testThread(fake.serve(t))

	reply, err := th.Continue(context.Background(), adapters.ThreadTurn{
		System:       "rules",
		Input:        "next",
		Continuation: adapters.Continuation{ID: "resp_old", Model: "gpt-5-mini", Made: time.Now()},
		Transcript:   seedTranscript,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Continuation.ID != "resp_new" {
		t.Errorf("continuation = %+v", reply.Continuation)
	}
	assertSeeded(t, fake.requests[1])
}

// A continuation older than OpenAI's retention is not even tried.
func TestThreadReseedsAfterRetention(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) { streamed(w, "resp_new", "Hello.") }}
	th := testThread(fake.serve(t))

	_, err := th.Continue(context.Background(), adapters.ThreadTurn{
		System:       "rules",
		Input:        "next",
		Continuation: adapters.Continuation{ID: "resp_old", Model: "gpt-5-mini", Made: time.Now().Add(-31 * 24 * time.Hour)},
		Transcript:   seedTranscript,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("%d requests, want one fresh call", len(fake.requests))
	}
	assertSeeded(t, fake.requests[0])
}

// seedTranscript is an opening and a player's turn, the opening constrained.
var seedTranscript = []adapters.Exchange{
	{Block: "outline", Input: "open", Directive: true, Constraints: "be kind", Reply: "one"},
	{Block: "outline", Input: "first", Reply: "two"},
}

// assertSeeded checks a fresh thread begins with the System Message again,
// then replays the transcript as it was sent, then makes the turn.
func assertSeeded(t *testing.T, sent map[string]any) {
	t.Helper()
	if _, continued := sent["previous_response_id"]; continued {
		t.Errorf("request = %v, want a fresh thread", sent)
	}
	got, _ := json.Marshal(sent["input"])
	want := `[{"content":"rules","role":"developer"},` +
		`{"content":"be kind","role":"developer"},{"content":"open","role":"developer"},{"content":"one","role":"assistant"},` +
		`{"content":"first","role":"user"},{"content":"two","role":"assistant"},` +
		`{"content":"next","role":"user"}]`
	if string(got) != want {
		t.Errorf("input = %s, want the transcript then the turn: %s", got, want)
	}
}

// A stream that stops before its completed response may pass, so the block
// makes the turn again from the same continuation.
func TestThreadBrokenStreamMayPass(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"The bri\"}\n\n")
	}}
	th := testThread(fake.serve(t))

	_, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "x"})
	if !errors.Is(err, adapters.ErrTransient) {
		t.Fatalf("got %v, want a failure that may pass", err)
	}
}

func TestThreadRejectedKey(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) { w.WriteHeader(http.StatusUnauthorized) }}
	th := testThread(fake.serve(t))

	_, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "x"})
	if !errors.Is(err, adapters.ErrKeyRejected) {
		t.Fatalf("got %v, want ErrKeyRejected", err)
	}
}

var schema = &adapters.Schema{Name: "outline", Body: map[string]any{
	"type":                 "object",
	"properties":           map[string]any{"outline": map[string]any{"type": "string"}},
	"required":             []any{"outline"},
	"additionalProperties": false,
}}

// A structured turn asks for its schema in strict mode, on a continued
// response as on a fresh one; a plain turn asks for text, so no format is
// left to chance.
func TestThreadSendsTheTurnsFormat(t *testing.T) {
	fake := &fakeResponses{answer: func(n int, w http.ResponseWriter) {
		streamed(w, fmt.Sprintf("resp_%d", n), `{"outline":"x"}`)
	}}
	th := testThread(fake.serve(t))

	if _, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "a", Schema: schema}); err != nil {
		t.Fatal(err)
	}
	if _, err := th.Continue(context.Background(), adapters.ThreadTurn{
		Input:        "b",
		Continuation: adapters.Continuation{ID: "resp_1", Model: "gpt-5-mini", Made: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	structured, _ := json.Marshal(fake.requests[0]["text"])
	want := `{"format":{"name":"outline","schema":{"additionalProperties":false,` +
		`"properties":{"outline":{"type":"string"}},"required":["outline"],"type":"object"},` +
		`"strict":true,"type":"json_schema"}}`
	if string(structured) != want {
		t.Errorf("structured text = %s, want %s", structured, want)
	}
	plain, _ := json.Marshal(fake.requests[1]["text"])
	if string(plain) != `{"format":{"type":"text"}}` {
		t.Errorf("plain text = %s, want the text format", plain)
	}
}

// A refusal is a bad reply: the provider answered, and the turn may be made again.
func TestThreadRefusalIsABadReply(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"type":"response.refusal.delta","delta":"I can't help with that."}`+"\n\n")
		fmt.Fprint(w, `data: {"type":"response.completed","response":{"id":"resp_r",`+
			`"usage":{"input_tokens":50,"output_tokens":6}}}`+"\n\n")
	}}
	th := testThread(fake.serve(t))

	reply, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Schema: schema})
	if !errors.Is(err, adapters.ErrBadReply) {
		t.Fatalf("got %v, want a bad reply", err)
	}
	if reply.Usage.OutputTokens != 6 {
		t.Errorf("usage = %+v, want the refusal billed", reply.Usage)
	}
}

// A structured reply cut short cannot parse, so it is a bad reply too.
func TestThreadIncompleteStructuredReplyIsBad(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"type":"response.output_text.delta","delta":"{\"outl"}`+"\n\n")
		fmt.Fprint(w, `data: {"type":"response.incomplete","response":{"id":"resp_i",`+
			`"incomplete_details":{"reason":"max_output_tokens"},`+
			`"usage":{"input_tokens":50,"output_tokens":900}}}`+"\n\n")
	}}
	th := testThread(fake.serve(t))

	reply, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Schema: schema})
	if !errors.Is(err, adapters.ErrBadReply) || !strings.Contains(err.Error(), "max_output_tokens") {
		t.Fatalf("got %v, want a bad reply naming why it stopped", err)
	}
	if reply.Usage.OutputTokens != 900 {
		t.Errorf("usage = %+v, want the cut reply billed", reply.Usage)
	}
}
