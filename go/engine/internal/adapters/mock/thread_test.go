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
