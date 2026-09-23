package blocks_test

import (
	"context"
	"testing"
	"time"

	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// The gate holds until every block wired to it reports done, which is what the
// init stem means: play begins when preparation has finished, not when it has
// started.
func TestGateWaitsForEveryReportingBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("init")
	gate := blocks.NewGate("start-game")
	first := blocks.NewImageOnce("first", slowImage{delay: 60 * time.Millisecond}, "a")
	second := blocks.NewImageOnce("second", slowImage{delay: 180 * time.Millisecond}, "b")
	sink := blocks.NewPlayerOutputImage("out-image")

	g.ConnectImageOut(first, sink)
	g.ConnectImageOut(second, sink)
	g.ConnectSignal(first, gate)
	g.ConnectSignal(second, gate)

	g.Start(ctx)

	select {
	case <-gate.Ready():
		t.Fatal("the gate opened before the slower block reported done")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened")
	}
}

// A genre with nothing to prepare is ready at once, so a caller never has to
// ask which kind of genre it got.
func TestGateWithNothingToWaitForOpensImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("no-init")
	gate := blocks.NewGate("start-game")
	g.Track(gate)
	g.Start(ctx)

	select {
	case <-gate.Ready():
	case <-time.After(time.Second):
		t.Fatal("a gate with no inputs must open immediately")
	}
}

// A block that fails still reports done. Otherwise one broken optional step
// would hold the gate shut forever.
func TestFailedBlockStillReportsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("failing-init")
	gate := blocks.NewGate("start-game")
	broken := blocks.NewImageOnce("broken", failingImage{}, "a")
	sink := blocks.NewPlayerOutputImage("out-image")

	g.ConnectImageOut(broken, sink)
	g.ConnectSignal(broken, gate)
	g.Start(ctx)

	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("a failed block must still report done, or the gate never opens")
	}
}

type slowImage struct{ delay time.Duration }

func (s slowImage) Generate(ctx context.Context, prompt string) ([]byte, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return mock.Image{}.Generate(ctx, prompt)
}

type failingImage struct{}

func (failingImage) Generate(context.Context, string) ([]byte, error) {
	return nil, errNoImage
}

var errNoImage = &imageError{}

type imageError struct{}

func (*imageError) Error() string { return "no image today" }
