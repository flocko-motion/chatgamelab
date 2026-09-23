package blocks_test

import (
	"context"
	"testing"
	"time"

	"engine/internal/blocks"
	"engine/internal/ports"
)

// The source fires at startup and then closes, which is the whole of the
// once-only mechanism: nothing downstream needs a flag of its own.
func TestOnceTextFiresExactlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("stem")
	source := blocks.NewOnceText("scenario-prompt", "a bridge keeper")
	sink := blocks.NewPlayerOutputText("out-text")
	g.ConnectTextOut(source, sink)
	g.Start(ctx)

	select {
	case got := <-sink.Seen:
		if got != "a bridge keeper" {
			t.Fatalf("got %q, want %q", got, "a bridge keeper")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the source never fired")
	}

	select {
	case got := <-sink.Seen:
		t.Fatalf("the source fired twice, second time %q", got)
	case <-time.After(150 * time.Millisecond):
	}
}

// A resumed session must not run its init stem again: whatever the line
// triggered the first time was paid for and persisted, and doing it again would
// buy a different result and overwrite the one the player was shown. The stem
// still reports its cycle, or a genre that gates on it would never start.
func TestRestoredOnceTextStaysSilentAndStillReportsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := ports.NewGraph("stem")
	fired := blocks.NewOnceText("scenario-prompt", "a bridge keeper")
	firstSink := blocks.NewPlayerOutputText("out-text")
	first.ConnectTextOut(fired, firstSink)
	first.Start(ctx)

	select {
	case <-firstSink.Seen:
	case <-time.After(2 * time.Second):
		t.Fatal("the source never fired")
	}

	state := first.ExportState()
	if state["scenario-prompt"] == "" {
		t.Fatal("a fired source must export a marker, or a resume repeats the work")
	}

	resumed := ports.NewGraph("stem")
	source := blocks.NewOnceText("scenario-prompt", "a bridge keeper")
	sink := blocks.NewPlayerOutputText("out-text")
	gate := blocks.NewGate("start-game")
	resumed.ConnectTextOut(source, sink)
	resumed.ConnectState(source, gate)
	if err := resumed.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	resumed.Start(ctx)

	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("a restored source must still report working-then-ready")
	}

	select {
	case got := <-sink.Seen:
		t.Fatalf("a restored source fired again, sending %q", got)
	case <-time.After(150 * time.Millisecond):
	}
}
