package mock

import (
	"context"
	"strings"
	"testing"

	"engine/internal/adapters"
)

func TestThreadCountsOnFromItsContinuation(t *testing.T) {
	ctx := context.Background()
	var pieces []string
	first, err := Thread{}.Continue(ctx, adapters.ThreadTurn{
		Input: "I look around",
		Piece: func(p string) { pieces = append(pieces, p) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "[story turn 1] I look around" || first.Continuation.ID != "mock-thread-1" {
		t.Fatalf("first = %+v", first)
	}
	if got := strings.Join(pieces, ""); got != first.Text || len(pieces) < 2 {
		t.Errorf("pieces %q should stream the reply word by word", pieces)
	}
	if first.Usage.Model != "mock-story" || first.Usage.OutputTokens == 0 {
		t.Errorf("usage = %+v, want billed mock-story tokens", first.Usage)
	}

	next, err := Thread{}.Continue(ctx, adapters.ThreadTurn{Input: "I go on", Continuation: first.Continuation})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(next.Text, "[story turn 2]") {
		t.Errorf("a continued thread said %q, want turn 2", next.Text)
	}
}

// A fresh thread seeded with a transcript goes on from it, so a test can tell
// a reseeded thread from one that lost the story.
func TestThreadSeededFromATranscriptCountsOn(t *testing.T) {
	reply, err := Thread{}.Continue(context.Background(), adapters.ThreadTurn{
		Input:      "I go on",
		Transcript: []adapters.Exchange{{Input: "a", Reply: "b"}, {Input: "c", Reply: "d"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reply.Text, "[story turn 3]") {
		t.Errorf("a seeded thread said %q, want turn 3", reply.Text)
	}
}

// A structured turn streams JSON of the schema's shape: each text field marks
// the turn and the input, and each status value is carried forward with the
// turn it was last seen, so a test sees it change.
func TestThreadStructuredTurnAnswersTheSchema(t *testing.T) {
	schema := &adapters.Schema{Name: "outline", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"outline": map[string]any{"type": "string"},
			"status": map[string]any{
				"type":       "object",
				"properties": map[string]any{"Health": map[string]any{"type": "string"}, "Gold": map[string]any{"type": "string"}},
			},
		},
	}}
	var pieces []string
	reply, err := Thread{}.Continue(context.Background(), adapters.ThreadTurn{
		Input:        `{"input":"I cross","status":{"Health":"badly wounded [turn 1]","Gold":"12"}}`,
		Schema:       schema,
		Continuation: adapters.Continuation{ID: "mock-thread-1"},
		Piece:        func(p string) { pieces = append(pieces, p) },
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"outline":"[outline 2] I cross","status":{"Gold":"12 [turn 2]","Health":"badly wounded [turn 2]"}}`
	if reply.Text != want {
		t.Errorf("reply %s\nwant  %s", reply.Text, want)
	}
	if strings.Join(pieces, "") != reply.Text || len(pieces) < 2 {
		t.Errorf("pieces %q should stream the reply word by word", pieces)
	}
}

// A spoken turn streams its transcript word by word and its audio as PCM16
// tones beside it, and bills audio output tokens on the thread's model.
func TestThreadSpokenTurnStreamsTranscriptAndAudio(t *testing.T) {
	var pieces []string
	var chunks [][]byte
	reply, err := Thread{Name: "mock-realtime"}.Continue(context.Background(), adapters.ThreadTurn{
		Input: "NARRATE",
		Speak: true,
		Piece: func(p string) { pieces = append(pieces, p) },
		Audio: func(c []byte) { chunks = append(chunks, c) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "[story turn 1] NARRATE" || strings.Join(pieces, "") != reply.Text {
		t.Errorf("reply %q, pieces %q: the transcript is the reply", reply.Text, pieces)
	}
	if len(chunks) < len(pieces) {
		t.Fatalf("%d audio chunks for %d words, want each word spoken", len(chunks), len(pieces))
	}
	for _, c := range chunks {
		if len(c) == 0 || len(c)%2 != 0 {
			t.Fatalf("chunk of %d bytes, want PCM16 samples", len(c))
		}
	}
	if reply.Usage.Model != "mock-realtime" || reply.Usage.AudioOutputTokens == 0 {
		t.Errorf("usage = %+v, want audio output billed on mock-realtime", reply.Usage)
	}
	if reply.Continuation.Model != "mock-realtime" {
		t.Errorf("continuation = %+v, want it made on the thread's model", reply.Continuation)
	}
}

// A plain turn says nothing aloud, even with an audio listener.
func TestThreadPlainTurnIsSilent(t *testing.T) {
	spoke := false
	reply, err := Thread{}.Continue(context.Background(), adapters.ThreadTurn{
		Input: "x",
		Audio: func([]byte) { spoke = true },
	})
	if err != nil {
		t.Fatal(err)
	}
	if spoke || reply.Usage.AudioOutputTokens != 0 {
		t.Errorf("a plain turn spoke: usage %+v", reply.Usage)
	}
}

// The input grows with the thread, so a long game nears the window: a
// continued thread bills more input each turn, and the spoken replies it
// holds bill audio input; a fresh thread seeded from text holds no audio.
func TestThreadInputGrowsWithTheThread(t *testing.T) {
	ctx := context.Background()
	speech := Thread{Name: "mock-realtime"}
	first, err := speech.Continue(ctx, adapters.ThreadTurn{Input: "a", Speak: true})
	if err != nil {
		t.Fatal(err)
	}
	if first.Usage.HeardAudio() {
		t.Errorf("the first call's input carried audio: %+v", first.Usage)
	}
	next, err := speech.Continue(ctx, adapters.ThreadTurn{Input: "a", Continuation: first.Continuation})
	if err != nil {
		t.Fatal(err)
	}
	if next.Usage.Context() <= first.Usage.Context() || !next.Usage.HeardAudio() {
		t.Errorf("input %+v after %+v, want it grown, with the spoken reply as audio", next.Usage, first.Usage)
	}
	reseeded, err := speech.Continue(ctx, adapters.ThreadTurn{Input: "a",
		Transcript: []adapters.Exchange{{Input: "a", Reply: first.Text}}})
	if err != nil {
		t.Fatal(err)
	}
	if reseeded.Usage.HeardAudio() {
		t.Errorf("a thread seeded from text carried audio: %+v", reseeded.Usage)
	}
}

// A summary seeds a fresh thread as one more turn it holds.
func TestThreadSeededWithASummaryCountsIt(t *testing.T) {
	reply, err := Thread{}.Continue(context.Background(), adapters.ThreadTurn{
		Input:      "I go on",
		Summary:    "the story so far",
		Transcript: []adapters.Exchange{{Input: "a", Reply: "b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reply.Text, "[story turn 3]") {
		t.Errorf("a thread seeded with a summary said %q, want turn 3", reply.Text)
	}
}
