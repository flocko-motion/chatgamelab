package openai

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

func testHearing(srv string) *Hearing {
	h := NewHearing(func(context.Context) (string, error) { return "sk-test", nil }, "gpt-audio-1.5")
	h.url = srv
	return h
}

// The clip goes as WAV with the system prompt, the schema as a forced
// function call; its arguments are the reply, and the audio tokens are
// counted apart from the text.
func TestHearingSendsTheClipAndReadsTheCall(t *testing.T) {
	fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"tool_calls":[{"function":{"name":"spoken","arguments":"{\"heard\":\"hi\"}"}}]}}],`+
			`"usage":{"prompt_tokens":150,"completion_tokens":20,"prompt_tokens_details":{"audio_tokens":40,"cached_tokens":10}}}`)
	}}
	srv := fake.serve(t)

	reply, used, err := testHearing(srv.URL).Hear(context.Background(), "hear it", adapters.Clip{1, 0, 2, 0}, spokenSchema)
	if err != nil {
		t.Fatal(err)
	}
	if reply != `{"heard":"hi"}` {
		t.Errorf("reply %q", reply)
	}
	if used != (adapters.Usage{Model: "gpt-audio-1.5", InputTokens: 100, CachedInputTokens: 10, AudioInputTokens: 40, OutputTokens: 20}) {
		t.Errorf("usage %+v", used)
	}

	req := fake.requests[0]
	if req["model"] != "gpt-audio-1.5" || fmt.Sprint(req["tool_choice"]) != "map[function:map[name:spoken] type:function]" {
		t.Errorf("request %v", req)
	}
	messages := req["messages"].([]any)
	if system := messages[0].(map[string]any); system["role"] != "system" || system["content"] != "hear it" {
		t.Errorf("system message %v", system)
	}
	part := messages[1].(map[string]any)["content"].([]any)[0].(map[string]any)["input_audio"].(map[string]any)
	wav, _ := base64.StdEncoding.DecodeString(part["data"].(string))
	if part["format"] != "wav" || !bytes.HasPrefix(wav, []byte("RIFF")) || !bytes.HasSuffix(wav, []byte{1, 0, 2, 0}) {
		t.Errorf("audio part %v", part)
	}
}

// A refusal, a reply cut short and one without the call are bad replies,
// still billed.
func TestHearingBadReplies(t *testing.T) {
	for name, body := range map[string]string{
		"refused":   `{"choices":[{"finish_reason":"stop","message":{"refusal":"no"}}],"usage":{"prompt_tokens":5}}`,
		"truncated": `{"choices":[{"finish_reason":"length","message":{}}],"usage":{"prompt_tokens":5}}`,
		"no call":   `{"choices":[{"finish_reason":"stop","message":{}}],"usage":{"prompt_tokens":5}}`,
	} {
		t.Run(name, func(t *testing.T) {
			fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) { fmt.Fprint(w, body) }}
			srv := fake.serve(t)
			_, used, err := testHearing(srv.URL).Hear(context.Background(), "hear it", adapters.Clip{1, 0}, spokenSchema)
			if !errors.Is(err, adapters.ErrBadReply) {
				t.Errorf("err %v, want a bad reply", err)
			}
			if used.InputTokens != 5 {
				t.Errorf("usage %+v, want the call billed", used)
			}
		})
	}
}

// A provider that is down may come back; a rejected key will not.
func TestHearingStatusErrors(t *testing.T) {
	for status, want := range map[int]error{503: adapters.ErrTransient, 401: adapters.ErrKeyRejected} {
		fake := &fakeResponses{answer: func(_ int, w http.ResponseWriter) { w.WriteHeader(status) }}
		srv := fake.serve(t)
		if _, _, err := testHearing(srv.URL).Hear(context.Background(), "", adapters.Clip{1, 0}, spokenSchema); !errors.Is(err, want) {
			t.Errorf("%d: err %v, want %v", status, err, want)
		}
	}
}
