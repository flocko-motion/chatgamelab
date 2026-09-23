package blocks_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"engine/internal/blocks"
	"engine/internal/ports"
)

// One output must feed N receivers, each seeing every value. A shared channel
// would make the receivers steal from one another instead, which nothing about
// the types would catch.
func TestOneOutputFeedsManyReceivers(t *testing.T) {
	const receivers = 4
	lines := []string{"alpha", "beta", "gamma"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("fan-out")
	src := blocks.NewDummyInputText("src", blocks.InputScript{Lines: lines})

	sinks := make([]*blocks.PlayerOutputText, receivers)
	for i := range sinks {
		sinks[i] = blocks.NewPlayerOutputText(fmt.Sprintf("sink-%d", i), "text")
		g.ConnectTextOut(src, sinks[i])
	}

	g.Start(ctx)

	for i, sink := range sinks {
		for _, want := range lines {
			select {
			case got := <-sink.Arrivals():
				if got != want {
					t.Fatalf("sink %d: got %q, want %q", i, got, want)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("sink %d never received %q", i, want)
			}
		}
	}
}

// An interval-driven source keeps producing, and every receiver keeps up.
func TestIntervalSourceFansOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("fan-out-interval")
	src := blocks.NewDummyInputText("ticker", blocks.InputScript{
		Lines:    []string{"tick"},
		Interval: 10 * time.Millisecond,
		Repeat:   true,
	})
	a := blocks.NewPlayerOutputText("a", "text")
	b := blocks.NewPlayerOutputText("b", "text")
	g.ConnectTextOut(src, a)
	g.ConnectTextOut(src, b)
	g.Start(ctx)

	for _, sink := range []*blocks.PlayerOutputText{a, b} {
		for i := 0; i < 3; i++ {
			select {
			case <-sink.Arrivals():
			case <-time.After(2 * time.Second):
				t.Fatalf("%s stopped receiving at %d", sink.NodeName(), i)
			}
		}
	}
}
