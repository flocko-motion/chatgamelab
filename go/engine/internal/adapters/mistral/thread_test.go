package mistral

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

	"engine/internal/adapters"
)

// call is one request the fake conversations API saw.
type call struct {
	path string
	body map[string]any
}

// fakeConversations streams replies as Mistral does, recording each request.
type fakeConversations struct {
	mu    sync.Mutex
	calls []call
	// answer writes the response to the nth request.
	answer func(n int, path string, w http.ResponseWriter)
}

func (f *fakeConversations) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("unreadable request: %v", err)
		}
		f.mu.Lock()
		f.calls = append(f.calls, call{path: r.URL.Path, body: body})
		n := len(f.calls)
		f.mu.Unlock()
		f.answer(n, r.URL.Path, w)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// streamed writes the stream Mistral sent when probed on 21 November 2026.
func streamed(w http.ResponseWriter, conversation string, pieces ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "event: conversation.response.started\ndata: {\"type\":\"conversation.response.started\",\"conversation_id\":%q}\n\n", conversation)
	for _, p := range pieces {
		delta, _ := json.Marshal(map[string]any{"type": "message.output.delta", "role": "assistant", "content": p})
		fmt.Fprintf(w, "event: message.output.delta\ndata: %s\n\n", delta)
	}
	fmt.Fprint(w, "event: conversation.response.done\ndata: {\"type\":\"conversation.response.done\","+
		"\"usage\":{\"prompt_tokens\":37,\"completion_tokens\":4,\"total_tokens\":41}}\n\n")
}

func testThread(srv *httptest.Server) *Thread {
	th := NewThread(key, "mistral-small-2603")
	th.api.base = srv.URL
	return th
}

func TestThreadStartsThenAppends(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, _ string, w http.ResponseWriter) {
		streamed(w, "conv_1", "The bridge ", "sways.")
	}}
	th := testThread(fake.serve(t))

	var pieces []string
	first, err := th.Continue(context.Background(), adapters.ThreadTurn{
		Instructions: "Expand the outline.",
		Input:        "The player crosses.",
		Piece:        func(p string) { pieces = append(pieces, p) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "The bridge sways." || strings.Join(pieces, "|") != "The bridge |sways." {
		t.Errorf("reply %q, pieces %q", first.Text, pieces)
	}
	if first.Continuation.ID != "conv_1" || first.Continuation.Model != "mistral-small-2603" {
		t.Errorf("continuation = %+v", first.Continuation)
	}
	if first.Usage != (adapters.Usage{Model: "mistral-small-2603", InputTokens: 37, OutputTokens: 4}) {
		t.Errorf("usage = %+v", first.Usage)
	}
	start := fake.calls[0]
	if start.path != "/conversations" || start.body["model"] != "mistral-small-2603" || start.body["store"] != true || start.body["stream"] != true {
		t.Errorf("start = %+v", start)
	}
	// Instructions travel in the user entry, since an append cannot carry them.
	assertInputs(t, start.body, `[{"content":"Expand the outline.\n\nThe player crosses.","object":"entry","role":"user","type":"message.input"}]`)

	if _, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "On.", Continuation: first.Continuation}); err != nil {
		t.Fatal(err)
	}
	appended := fake.calls[1]
	if appended.path != "/conversations/conv_1" {
		t.Errorf("append went to %s", appended.path)
	}
	// Appending takes no model: the conversation stays on the one it started with.
	if _, has := appended.body["model"]; has {
		t.Errorf("append = %v, carries a model", appended.body)
	}
}

// A conversation Mistral no longer holds is taken over by a fresh one, seeded
// with the transcript, so the session goes on.
func TestThreadReseedsAnUnknownConversation(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, path string, w http.ResponseWriter) {
		if path != "/conversations" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"object":"Error","message":"Conversation with id conv_old was not found","type":"invalid_request_error","code":3000}`)
			return
		}
		streamed(w, "conv_new", "Again.")
	}}
	th := testThread(fake.serve(t))

	reply, err := th.Continue(context.Background(), adapters.ThreadTurn{
		Input:        "next",
		Continuation: adapters.Continuation{ID: "conv_old", Model: "mistral-small-2603"},
		Transcript:   []adapters.Exchange{{Instructions: "Outline.", Input: "first", Reply: "one"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Continuation.ID != "conv_new" || len(fake.calls) != 2 {
		t.Fatalf("continuation = %+v after %d calls", reply.Continuation, len(fake.calls))
	}
	assertInputs(t, fake.calls[1].body, `[`+
		// Each past call as it was sent, with its own instructions.
		`{"content":"Outline.\n\nfirst","object":"entry","role":"user","type":"message.input"},`+
		`{"content":"one","object":"entry","role":"assistant","type":"message.input"},`+
		`{"content":"next","object":"entry","role":"user","type":"message.input"}]`)
}

func assertInputs(t *testing.T, body map[string]any, want string) {
	t.Helper()
	got, _ := json.Marshal(body["inputs"])
	if string(got) != want {
		t.Errorf("inputs = %s\nwant      %s", got, want)
	}
}

// A stream that stops before it is done may pass, so the block makes the turn
// again from the same continuation.
func TestThreadBrokenStreamMayPass(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, _ string, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"message.output.delta\",\"content\":\"The bri\"}\n\n")
	}}
	th := testThread(fake.serve(t))

	_, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "x"})
	if !errors.Is(err, adapters.ErrTransient) {
		t.Fatalf("got %v, want a failure that may pass", err)
	}
}

// A structured turn asks for its schema in strict mode, on start and on
// append; a plain turn asks for text, so a format stored on the conversation
// never reaches the next turn.
func TestThreadSendsTheTurnsFormat(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, _ string, w http.ResponseWriter) {
		streamed(w, "conv_1", `{"outline":"x"}`)
	}}
	th := testThread(fake.serve(t))
	schema := &adapters.Schema{Name: "outline", Body: map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"outline": map[string]any{"type": "string"}},
		"required":             []any{"outline"},
		"additionalProperties": false,
	}}

	first, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "a", Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "b", Schema: schema, Continuation: first.Continuation}); err != nil {
		t.Fatal(err)
	}
	if _, err := th.Continue(context.Background(), adapters.ThreadTurn{Input: "c", Continuation: first.Continuation}); err != nil {
		t.Fatal(err)
	}

	want := `{"response_format":{"json_schema":{"name":"outline","schema":{"additionalProperties":false,` +
		`"properties":{"outline":{"type":"string"}},"required":["outline"],"type":"object"},"strict":true},` +
		`"type":"json_schema"}}`
	for i, c := range fake.calls[:2] {
		if got, _ := json.Marshal(c.body["completion_args"]); string(got) != want {
			t.Errorf("call %d (%s) completion_args = %s, want %s", i, c.path, got, want)
		}
	}
	if got, _ := json.Marshal(fake.calls[2].body["completion_args"]); string(got) != `{"response_format":{"type":"text"}}` {
		t.Errorf("plain completion_args = %s, want the text format", got)
	}
}
