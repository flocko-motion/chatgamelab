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
