package engine

import (
	"strings"
	"testing"
)

// A block on the story thread reports how full the thread is, and compacts
// it on demand: the thread summarises at once, and the game plays on from
// the summary. A block on no thread cannot.
func TestSessionCompactsAThreadOnDemand(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	say(t, s, "I cross the bridge")
	streams(t, s, "text")

	detail, _ := s.Inspect("expand")
	if detail.Thread == nil || detail.Thread.Tokens <= 0 || detail.Thread.Window <= 0 {
		t.Fatalf("expand's detail %+v, want how full its thread is", detail.Thread)
	}
	if other, _ := s.Inspect("image"); other.Thread != nil {
		t.Errorf("the image block reports a thread: %+v", other.Thread)
	}

	if err := s.Compact("expand"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the thread summarised", func() bool {
		return strings.Contains(s.wiring.Graph.ExportState()["story-thread"], `"summary":`)
	})
	if err := s.Compact("image"); err == nil {
		t.Error("compacted the image block, want it refused")
	}

	drain(s)
	say(t, s, "I walk on")
	if seen := streams(t, s, "text"); !seen["text"] {
		t.Error("no prose after compaction")
	}
}
