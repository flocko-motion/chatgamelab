package mistral

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"engine/internal/adapters"
)

var spokenSchema = adapters.Schema{Name: "spoken", Body: map[string]any{"type": "object"}}

func testHearing(base string) *Hearing {
	h := NewHearing(func(context.Context) (string, error) { return "k", nil }, "voxtral-small-2507")
	h.api.base = base
	return h
}

// The clip goes as WAV with the system prompt, the schema in strict mode; the
// reply is the message's content, and the clip's length is billed beside the
// text tokens.
func TestHearingSendsTheClipWithTheSchema(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, _ string, w http.ResponseWriter) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"heard\":\"hallo\"}"}}],"usage":{"prompt_tokens":90,"completion_tokens":12}}`)
	}}
	srv := fake.serve(t)
	clip := make(adapters.Clip, adapters.ClipRate) // half a second

	reply, used, err := testHearing(srv.URL).Hear(context.Background(), "hear it", clip, spokenSchema)
	if err != nil {
		t.Fatal(err)
	}
	if reply != `{"heard":"hallo"}` {
		t.Errorf("reply %q", reply)
	}
	if used != (adapters.Usage{Model: "voxtral-small-2507", InputTokens: 90, OutputTokens: 12, AudioSeconds: 0.5}) {
		t.Errorf("usage %+v", used)
	}

	got := fake.calls[0]
	if got.path != "/chat/completions" || got.body["model"] != "voxtral-small-2507" {
		t.Errorf("request %s %v", got.path, got.body)
	}
	if format := fmt.Sprint(got.body["response_format"]); format != "map[json_schema:map[name:spoken schema:map[type:object] strict:true] type:json_schema]" {
		t.Errorf("format %s", format)
	}
	messages := got.body["messages"].([]any)
	if system := messages[0].(map[string]any); system["content"] != "hear it" {
		t.Errorf("system message %v", system)
	}
	audio := messages[1].(map[string]any)["content"].([]any)[0].(map[string]any)
	wav, _ := base64.StdEncoding.DecodeString(audio["input_audio"].(string))
	if audio["type"] != "input_audio" || !bytes.HasPrefix(wav, []byte("RIFF")) {
		t.Errorf("audio part %v", audio["type"])
	}
}

// A reply cut short is a bad reply, still billed.
func TestHearingCutShortIsABadReply(t *testing.T) {
	fake := &fakeConversations{answer: func(_ int, _ string, w http.ResponseWriter) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"length","message":{"content":"{\"heard"}}],"usage":{"prompt_tokens":90}}`)
	}}
	srv := fake.serve(t)
	_, used, err := testHearing(srv.URL).Hear(context.Background(), "", adapters.Clip{1, 0}, spokenSchema)
	if !errors.Is(err, adapters.ErrBadReply) || used.InputTokens != 90 {
		t.Errorf("err %v, usage %+v: want a bad reply, billed", err, used)
	}
}
