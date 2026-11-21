package engine

import (
	"reflect"
	"testing"
)

// Streamed pieces are kept as one text event per finished utterance, followed
// by its close, so the history keeps the story rather than its fragments; a
// discarded utterance is never kept.
func TestRecorderKeepsWholeUtterances(t *testing.T) {
	r := newRecorder()
	var stored []SessionEvent
	for _, e := range []SessionEvent{
		{Stream: "text", Value: "The bri"},
		{Stream: "text", Value: TextDiscard},
		{Stream: "text", Value: "The bridge "},
		{Stream: "image", Value: "<png>"},
		{Stream: "text", Value: "sways."},
		{Stream: "text", Value: ""},
		// A close with nothing open keeps nothing.
		{Stream: "text", Value: ""},
		{Stream: "audio", Value: "<pcm>"},
	} {
		stored = append(stored, r.record(e)...)
	}

	want := []SessionEvent{
		{Stream: "image", Value: "<png>"},
		{Stream: "text", Value: "The bridge sways."},
		{Stream: "text", Value: ""},
	}
	if !reflect.DeepEqual(r.replay(), want) {
		t.Errorf("history = %q, want %q", r.replay(), want)
	}
	if !reflect.DeepEqual(stored, want) {
		t.Errorf("stored = %q, want what the history keeps, in its order", stored)
	}
}
