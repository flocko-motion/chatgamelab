package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

// fakeRealtime is a Realtime API server: it opens a session per connection,
// records every client event, and answers each response.create with answer.
type fakeRealtime struct {
	mu       sync.Mutex
	sessions int
	// events are the client events, per session, in order.
	events [][]map[string]any
	// answer writes the reply to a response.create on a session; nil speaks
	// "The bridge sways." with two audio chunks.
	answer func(session int, request map[string]any, send func(map[string]any))
	// reject answers the handshake with this status instead.
	reject int
}

func (f *fakeRealtime) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.reject != 0 {
			w.WriteHeader(f.reject)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("authorization %q", got)
		}
		if got := r.URL.Query().Get("model"); got != "gpt-realtime-2.1-mini" {
			t.Errorf("model %q", got)
		}
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer ws.CloseNow()
		ws.SetReadLimit(-1)

		f.mu.Lock()
		f.sessions++
		session := f.sessions
		f.events = append(f.events, nil)
		f.mu.Unlock()

		ctx := r.Context()
		send := func(event map[string]any) {
			blob, _ := json.Marshal(event)
			_ = ws.Write(ctx, websocket.MessageText, blob)
		}
		send(map[string]any{"type": "session.created", "session": map[string]any{"id": fmt.Sprintf("sess_%d", session)}})
		for {
			_, data, err := ws.Read(ctx)
			if err != nil {
				return
			}
			var event map[string]any
			_ = json.Unmarshal(data, &event)
			f.mu.Lock()
			f.events[session-1] = append(f.events[session-1], event)
			f.mu.Unlock()
			switch event["type"] {
			case "conversation.item.delete":
				send(map[string]any{"type": "conversation.item.deleted", "item_id": event["item_id"]})
			case "response.create":
				request, _ := event["response"].(map[string]any)
				answer := f.answer
				if answer == nil {
					answer = spoken("The bridge ", "sways.")
				}
				answer(session, request, send)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// sent is the client events of one session, by type.
func (f *fakeRealtime) sent(session int, kind string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	for _, e := range f.events[session-1] {
		if e["type"] == kind {
			out = append(out, e)
		}
	}
	return out
}

func (f *fakeRealtime) opened() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sessions
}

// realtimeUsage is response.done's usage: 100 text and 50 audio tokens in, of
// which 40 text and 10 audio cached, and 8 text and 60 audio tokens out.
var realtimeUsage = map[string]any{
	"input_tokens": 150, "output_tokens": 68,
	"input_token_details": map[string]any{
		"text_tokens": 100, "audio_tokens": 50, "cached_tokens": 50,
		"cached_tokens_details": map[string]any{"text_tokens": 40, "audio_tokens": 10},
	},
	"output_token_details": map[string]any{"text_tokens": 8, "audio_tokens": 60},
}

var itemCount int

// spoken answers with an audio message: transcript deltas, each followed by
// an audio chunk, then the done response.
func spoken(pieces ...string) func(int, map[string]any, func(map[string]any)) {
	return func(_ int, _ map[string]any, send func(map[string]any)) {
		itemCount++
		item := fmt.Sprintf("item_%d", itemCount)
		send(map[string]any{"type": "response.created", "response": map[string]any{"id": "resp_" + item}})
		for i, p := range pieces {
			send(map[string]any{"type": "response.output_audio_transcript.delta", "item_id": item, "delta": p})
			send(map[string]any{"type": "response.output_audio.delta", "item_id": item,
				"delta": base64.StdEncoding.EncodeToString([]byte{byte(i + 1), 0})})
		}
		send(map[string]any{"type": "response.done", "response": map[string]any{
			"id": "resp_" + item, "status": "completed",
			"output": []any{map[string]any{"id": item, "type": "message", "role": "assistant",
				"content": []any{map[string]any{"type": "output_audio", "transcript": strings.Join(pieces, "")}}}},
			"usage": realtimeUsage,
		}})
	}
}

// called answers with a function call whose arguments are args.
func called(args string) func(int, map[string]any, func(map[string]any)) {
	return func(_ int, _ map[string]any, send func(map[string]any)) {
		itemCount++
		item := fmt.Sprintf("item_%d", itemCount)
		send(map[string]any{"type": "response.function_call_arguments.delta", "item_id": item, "delta": args})
		send(map[string]any{"type": "response.done", "response": map[string]any{
			"id": "resp_" + item, "status": "completed",
			"output": []any{map[string]any{"id": item, "type": "function_call", "name": "outline",
				"call_id": "call_" + item, "arguments": args}},
			"usage": realtimeUsage,
		}})
	}
}

func testRealtime(srv *httptest.Server) *Realtime {
	r := NewRealtime(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-realtime-2.1-mini", "cedar")
	r.url = "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/realtime"
	return r
}

// A spoken turn streams its transcript as pieces and its audio beside it; the
// reply is the transcript, and audio tokens are billed apart from text.
func TestRealtimeSpokenTurn(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	var pieces []string
	var audio [][]byte
	reply, err := r.Continue(context.Background(), adapters.ThreadTurn{
		System:      "You are a game master.",
		Input:       "NARRATE",
		Directive:   true,
		Constraints: "NARRATION RULES must be respected: keep it gentle",
		Speak:       true,
		Piece:       func(p string) { pieces = append(pieces, p) },
		Audio:       func(c []byte) { audio = append(audio, c) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "The bridge sways." || strings.Join(pieces, "|") != "The bridge |sways." {
		t.Errorf("reply %q, pieces %q", reply.Text, pieces)
	}
	if len(audio) != 2 || audio[0][0] != 1 || audio[1][0] != 2 {
		t.Errorf("audio %v, want both chunks decoded in order", audio)
	}
	want := adapters.Usage{
		Model: "gpt-realtime-2.1-mini", InputTokens: 60, CachedInputTokens: 40, OutputTokens: 8,
		AudioInputTokens: 40, CachedAudioInputTokens: 10, AudioOutputTokens: 60,
	}
	if reply.Usage != want {
		t.Errorf("usage = %+v\nwant    %+v", reply.Usage, want)
	}
	if reply.Continuation.ID != "sess_1/1" || reply.Continuation.Model != "gpt-realtime-2.1-mini" || reply.Continuation.Made.IsZero() {
		t.Errorf("continuation = %+v", reply.Continuation)
	}

	update := fake.sent(1, "session.update")
	if len(update) != 1 {
		t.Fatalf("%d session updates, want one", len(update))
	}
	session, _ := json.Marshal(update[0]["session"])
	// The System Message is the session's instructions, set once as it opens.
	for _, want := range []string{`"instructions":"You are a game master."`, `"voice":"cedar"`, `"turn_detection":null`, `"type":"audio/pcm"`, `"rate":24000`} {
		if !strings.Contains(string(session), want) {
			t.Errorf("session %s lacks %s", session, want)
		}
	}
	created := fake.sent(1, "response.create")
	response, _ := json.Marshal(created[0]["response"])
	// A response sets no instructions, which would replace the session's.
	if strings.Contains(string(response), `"instructions"`) || !strings.Contains(string(response), `"output_modalities":["audio"]`) {
		t.Errorf("response.create = %s, want audio out and no instructions", response)
	}
	// The constraints go in a system message of their own, then the game's
	// own message as a system message.
	if got := items(fake.sent(1, "conversation.item.create")); got !=
		"system/input_text:NARRATION RULES must be respected: keep it gentle system/input_text:NARRATE" {
		t.Errorf("items %s, want the constraints apart, then the game's message", got)
	}
}

// A JSON turn is a forced function call named after the schema; its arguments
// are the reply, and the call is answered so the conversation stays whole.
func TestRealtimeJSONTurnIsAForcedFunctionCall(t *testing.T) {
	fake := &fakeRealtime{answer: called(`{"outline":"x"}`)}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	reply, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "act", Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != `{"outline":"x"}` {
		t.Errorf("reply %q, want the arguments", reply.Text)
	}

	created := fake.sent(1, "response.create")
	response, _ := json.Marshal(created[0]["response"])
	for _, want := range []string{
		`"output_modalities":["text"]`,
		`"tool_choice":{"name":"outline","type":"function"}`,
		`"tools":[{"name":"outline","parameters":{"additionalProperties":false,`,
	} {
		if !strings.Contains(string(response), want) {
			t.Errorf("response.create = %s\nlacks %s", response, want)
		}
	}
	eventually(t, func() bool { return len(fake.sent(1, "conversation.item.create")) == 2 })
	output, _ := json.Marshal(fake.sent(1, "conversation.item.create")[1]["item"])
	if !strings.Contains(string(output), `"type":"function_call_output"`) || !strings.Contains(string(output), `"call_id":"call_item_`) {
		t.Errorf("item = %s, want the call answered", output)
	}
}

// The continuation is the session: the next call goes on in it, sending only
// its own input.
func TestRealtimeContinuesInTheSession(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	first, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "one", Speak: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Continue(context.Background(), adapters.ThreadTurn{
		Input: "two", Speak: true, Continuation: first.Continuation,
		Transcript: []adapters.Exchange{{Input: "one", Reply: first.Text}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.opened() != 1 || second.Continuation.ID != "sess_1/2" {
		t.Errorf("%d sessions, continuation %+v: want one session gone on", fake.opened(), second.Continuation)
	}
	if n := len(fake.sent(1, "conversation.item.create")); n != 2 {
		t.Errorf("%d items created, want only each call's input", n)
	}
}

// A reply the caller did not keep is taken out of the session: the next call
// from the continuation before it deletes the reply's items and goes on.
func TestRealtimeForgetsAReplyNotKept(t *testing.T) {
	fake := &fakeRealtime{answer: called(`{"outline":"x"}`)}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	kept, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "one", Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "two", Schema: schema, Continuation: kept.Continuation}); err != nil {
		t.Fatal(err)
	}
	again, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "two", Schema: schema, Continuation: kept.Continuation})
	if err != nil {
		t.Fatal(err)
	}
	if fake.opened() != 1 {
		t.Errorf("%d sessions, want the one session", fake.opened())
	}
	// The input, the function call and its output of the call not kept.
	if n := len(fake.sent(1, "conversation.item.delete")); n != 3 {
		t.Errorf("%d items deleted, want the three the reply added", n)
	}
	if again.Continuation.ID != "sess_1/2" {
		t.Errorf("continuation = %+v, want turn 2 again", again.Continuation)
	}
}

// A continuation the adapter does not hold, as after an engine restart, opens
// a fresh session seeded with the transcript as text items.
func TestRealtimeSeedsAFreshSession(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	_, err := r.Continue(context.Background(), adapters.ThreadTurn{
		System:       "rules",
		Input:        "next",
		Speak:        true,
		Continuation: adapters.Continuation{ID: "sess_gone/4", Model: "gpt-realtime-2.1-mini"},
		Transcript: []adapters.Exchange{
			{Input: "open", Directive: true, Constraints: "be kind", Reply: "zero"},
			{Input: "first", Reply: "one"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The fresh session gets the System Message again, and each exchange as
	// it was sent.
	update, _ := json.Marshal(fake.sent(1, "session.update")[0]["session"])
	if !strings.Contains(string(update), `"instructions":"rules"`) {
		t.Errorf("session %s, want the System Message", update)
	}
	want := "system/input_text:be kind system/input_text:open assistant/output_text:zero " +
		"user/input_text:first assistant/output_text:one user/input_text:next"
	if got := items(fake.sent(1, "conversation.item.create")); got != want {
		t.Errorf("items %s\nwant  %s", got, want)
	}
}

// items reads created conversation items as role/part:text.
func items(created []map[string]any) string {
	var out []string
	for _, e := range created {
		item, _ := e["item"].(map[string]any)
		content, _ := item["content"].([]any)
		if len(content) == 0 {
			continue
		}
		part, _ := content[0].(map[string]any)
		out = append(out, fmt.Sprintf("%s/%s:%s", item["role"], part["type"], part["text"]))
	}
	return strings.Join(out, " ")
}

// A session near OpenAI's limit is replaced before the call, seeded again.
func TestRealtimeReseedsBeforeTheSessionLimit(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()
	clock := time.Now()
	r.now = func() time.Time { return clock }

	first, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "one", Speak: true})
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(58 * time.Minute)
	if _, err := r.Continue(context.Background(), adapters.ThreadTurn{
		Input: "two", Speak: true, Continuation: first.Continuation,
		Transcript: []adapters.Exchange{{Input: "one", Reply: first.Text}},
	}); err != nil {
		t.Fatal(err)
	}
	if fake.opened() != 2 {
		t.Errorf("%d sessions, want a fresh one near the limit", fake.opened())
	}
}

// A reply cut short, or stopped by the content filter, is a bad reply: billed,
// taken out of the session, never kept.
func TestRealtimeBadReplies(t *testing.T) {
	for reason, want := range map[string]error{"max_output_tokens": adapters.ErrTruncated, "content_filter": adapters.ErrRefused} {
		t.Run(reason, func(t *testing.T) {
			fake := &fakeRealtime{answer: func(_ int, _ map[string]any, send func(map[string]any)) {
				send(map[string]any{"type": "response.function_call_arguments.delta", "delta": `{"outl`})
				send(map[string]any{"type": "response.done", "response": map[string]any{
					"status":         "incomplete",
					"status_details": map[string]any{"type": "incomplete", "reason": reason},
					"output": []any{map[string]any{"id": "item_cut", "type": "function_call", "name": "outline",
						"call_id": "call_cut", "arguments": `{"outl`}},
					"usage": realtimeUsage,
				}})
			}}
			r := testRealtime(fake.serve(t))
			defer r.Close()

			reply, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Schema: schema})
			if !errors.Is(err, want) || !errors.Is(err, adapters.ErrBadReply) {
				t.Fatalf("got %v, want %v", err, want)
			}
			if reply.Usage.AudioOutputTokens != 60 {
				t.Errorf("usage = %+v, want the bad reply billed", reply.Usage)
			}
			eventually(t, func() bool { return len(fake.sent(1, "conversation.item.delete")) == 2 })
		})
	}
}

// A connection that drops mid-reply may pass; the next call opens a fresh
// session seeded with the transcript.
func TestRealtimeDroppedConnectionMayPass(t *testing.T) {
	fake := &fakeRealtime{}
	fake.answer = func(session int, request map[string]any, send func(map[string]any)) {
		if session == 1 {
			send(map[string]any{"type": "response.output_audio_transcript.delta", "delta": "The bri"})
			panic(http.ErrAbortHandler)
		}
		spoken("Again.")(session, request, send)
	}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	_, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Speak: true})
	if !errors.Is(err, adapters.ErrTransient) {
		t.Fatalf("got %v, want a failure that may pass", err)
	}
	reply, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Speak: true})
	if err != nil || reply.Text != "Again." || fake.opened() != 2 {
		t.Errorf("reply %q, err %v, %d sessions: want a fresh session", reply.Text, err, fake.opened())
	}
}

// A provider error event fails the call; a server error may pass.
func TestRealtimeErrorEvent(t *testing.T) {
	fake := &fakeRealtime{answer: func(_ int, _ map[string]any, send func(map[string]any)) {
		send(map[string]any{"type": "error", "error": map[string]any{"type": "server_error", "code": "server_error", "message": "boom"}})
	}}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	_, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Speak: true})
	if !errors.Is(err, adapters.ErrTransient) || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v, want a server error that may pass", err)
	}
}

func TestRealtimeRejectedKey(t *testing.T) {
	fake := &fakeRealtime{reject: http.StatusUnauthorized}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	_, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Speak: true})
	if !errors.Is(err, adapters.ErrKeyRejected) {
		t.Fatalf("got %v, want ErrKeyRejected", err)
	}
}

func TestRealtimeSpokenTurnTakesNoSchema(t *testing.T) {
	r := NewRealtime(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-realtime-2.1-mini", "")
	if _, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "x", Speak: true, Schema: schema}); err == nil {
		t.Fatal("a spoken JSON turn was accepted")
	}
}

func eventually(t *testing.T, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); !ok(); {
		if time.Now().After(deadline) {
			t.Fatal("it never happened")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A compacted thread's fresh session is seeded with its summary, as the
// game's system message, before the exchanges after it.
func TestRealtimeSeedsTheSummary(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	_, err := r.Continue(context.Background(), adapters.ThreadTurn{
		System:     "rules",
		Summary:    "the story so far",
		Input:      "next",
		Speak:      true,
		Transcript: []adapters.Exchange{{Input: "first", Reply: "one"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "system/input_text:the story so far user/input_text:first assistant/output_text:one user/input_text:next"
	if got := items(fake.sent(1, "conversation.item.create")); got != want {
		t.Errorf("items %s\nwant  %s", got, want)
	}
}

// A turn's voice is the session's: a session opens in it, and a turn in
// another voice opens a fresh session in that one, seeded from the
// transcript, as the provider fixes a session's voice once it has spoken. A
// turn without one goes on in the session as it is.
func TestRealtimeSpeaksInTheTurnsVoice(t *testing.T) {
	fake := &fakeRealtime{}
	r := testRealtime(fake.serve(t))
	defer r.Close()

	voiceOf := func(session int) string {
		update, _ := json.Marshal(fake.sent(session, "session.update")[0]["session"])
		for _, v := range []string{"cedar", "marin", "ash"} {
			if strings.Contains(string(update), `"voice":"`+v+`"`) {
				return v
			}
		}
		return string(update)
	}

	first, err := r.Continue(context.Background(), adapters.ThreadTurn{Input: "one", Speak: true, Voice: "marin"})
	if err != nil {
		t.Fatal(err)
	}
	if got := voiceOf(1); got != "marin" {
		t.Errorf("the session opened in %s, want the turn's voice marin", got)
	}
	second, err := r.Continue(context.Background(), adapters.ThreadTurn{
		Input: "two", Continuation: first.Continuation, Transcript: []adapters.Exchange{{Input: "one", Reply: first.Text}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.opened() != 1 {
		t.Fatalf("%d sessions, want a turn without a voice to go on in the session", fake.opened())
	}
	if _, err := r.Continue(context.Background(), adapters.ThreadTurn{
		Input: "three", Speak: true, Voice: "ash", Continuation: second.Continuation,
		Transcript: []adapters.Exchange{{Input: "one", Reply: first.Text}, {Input: "two", Reply: second.Text}},
	}); err != nil {
		t.Fatal(err)
	}
	if fake.opened() != 2 || voiceOf(2) != "ash" {
		t.Errorf("%d sessions, the last in %s: want a fresh session in ash", fake.opened(), voiceOf(fake.opened()))
	}
}
