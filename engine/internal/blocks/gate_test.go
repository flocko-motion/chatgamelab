package blocks_test

import (
	"context"
	"testing"
	"time"

	"engine/internal/adapters"
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
	firstPrompt := blocks.NewOnceText("first-prompt", "a")
	secondPrompt := blocks.NewOnceText("second-prompt", "b")
	first := blocks.NewImage("first", slowImage{delay: 60 * time.Millisecond})
	second := blocks.NewImage("second", slowImage{delay: 180 * time.Millisecond})
	sink := blocks.NewPlayerOutputImage("out-image")

	g.ConnectTextOut(firstPrompt, first)
	g.ConnectTextOut(secondPrompt, second)
	g.ConnectImageOut(first, sink)
	g.ConnectImageOut(second, sink)
	g.ConnectState(first, gate)
	g.ConnectState(second, gate)

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

// A block that fails still returns to ready. Otherwise one broken optional step
// would hold the gate shut forever.
func TestFailedBlockStillReturnsToReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("failing-init")
	gate := blocks.NewGate("start-game")
	prompt := blocks.NewOnceText("prompt", "a")
	broken := blocks.NewImage("broken", failingImage{})
	sink := blocks.NewPlayerOutputImage("out-image")

	g.ConnectTextOut(prompt, broken)
	g.ConnectImageOut(broken, sink)
	g.ConnectState(broken, gate)
	g.Start(ctx)

	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("a failed block must still return to ready, or the gate never opens")
	}
}

type slowImage struct{ delay time.Duration }

func (s slowImage) Generate(ctx context.Context, prompt string) ([]byte, adapters.Usage, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, adapters.Usage{}, ctx.Err()
	}
	return mock.Image{}.Generate(ctx, prompt)
}

type failingImage struct{}

func (failingImage) Generate(context.Context, string) ([]byte, adapters.Usage, error) {
	return nil, adapters.Usage{}, errNoImage
}

var errNoImage = &imageError{}

type imageError struct{}

func (*imageError) Error() string { return "no image today" }

// A block in the turn loop reports done on every pass. The gate counts
// reporters by name, so a chatty one cannot stand in for a silent one — which
// would open the game before the other block had finished.
func TestGateCountsReportersNotMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("repeating-init")
	gate := blocks.NewGate("start-game")
	chatty := &repeatingReporter{name: "chatty", out: make(chan ports.State, 8)}
	silent := &repeatingReporter{name: "silent", out: make(chan ports.State, 8)}

	g.ConnectState(chatty, gate)
	g.ConnectState(silent, gate)
	g.Start(ctx)

	// One block works three times over; the gate must still be waiting.
	for i := 0; i < 3; i++ {
		chatty.out <- ports.State{Node: chatty.name, Phase: ports.PhaseWorking}
		chatty.out <- ports.State{Node: chatty.name, Phase: ports.PhaseReady}
	}
	select {
	case <-gate.Ready():
		t.Fatal("one reporter's repeats opened a gate waiting on two blocks")
	case <-time.After(150 * time.Millisecond):
	}

	silent.out <- ports.State{Node: silent.name, Phase: ports.PhaseWorking}
	silent.out <- ports.State{Node: silent.name, Phase: ports.PhaseReady}
	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened after every reporter finished")
	}
}

// A block still working has not finished, so the gate stays shut.
func TestGateIgnoresWorking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("working-init")
	gate := blocks.NewGate("start-game")
	busy := &repeatingReporter{name: "busy", out: make(chan ports.State, 8)}

	g.ConnectState(busy, gate)
	g.Start(ctx)

	busy.out <- ports.State{Node: busy.name, Phase: ports.PhaseWorking}
	select {
	case <-gate.Ready():
		t.Fatal("a working report opened the gate")
	case <-time.After(150 * time.Millisecond):
	}

	busy.out <- ports.State{Node: busy.name, Phase: ports.PhaseReady}
	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened")
	}
}

type repeatingReporter struct {
	name string
	out  chan ports.State
}

func (r *repeatingReporter) NodeName() string                 { return r.name }
func (r *repeatingReporter) StateOutPort() <-chan ports.State { return r.out }

// Every block reports ready at startup, before it has done anything. That must
// not satisfy a gate, or the game starts before preparation has even begun.
func TestInitialReadyDoesNotOpenTheGate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("startup-ready")
	gate := blocks.NewGate("start-game")
	block := &repeatingReporter{name: "slow", out: make(chan ports.State, 8)}

	g.ConnectState(block, gate)
	g.Start(ctx)

	block.out <- ports.State{Node: block.name, Phase: ports.PhaseReady}
	select {
	case <-gate.Ready():
		t.Fatal("a startup ready opened the gate before any work happened")
	case <-time.After(150 * time.Millisecond):
	}

	block.out <- ports.State{Node: block.name, Phase: ports.PhaseWorking}
	block.out <- ports.State{Node: block.name, Phase: ports.PhaseReady}
	select {
	case <-gate.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened after a full working cycle")
	}
}
