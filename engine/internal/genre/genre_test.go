package genre

import (
	"context"
	"strings"
	"testing"
	"time"

	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// Every genre's graph must be complete: types settle whether an edge is legal,
// this settles whether anything was left unwired.
func TestGraphsAreValid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		graph *ports.Graph
	}{
		{"adventure", NewAdventure(nil).Graph},
		{"npc-live", NewNPCLive(mockNPCConfig("stay in character", blocks.InputScript{})).Graph},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.graph.Validate(); err != nil {
				t.Fatalf("%v", err)
			}
		})
	}
}

// A block left unwired has to be caught here, because nothing about it fails to
// compile.
func TestValidateCatchesUnwiredBlock(t *testing.T) {
	g := ports.NewGraph("broken")
	player := blocks.NewPlayerInputText("player")
	rephrase := blocks.NewDummyToolCall("rephrase", "3rd-person")
	orphan := blocks.NewDummyImage("orphan-image")
	outText := blocks.NewPlayerOutputText("out-text")
	outImage := blocks.NewPlayerOutputImage("out-image")

	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, outText)
	g.ConnectImageOut(orphan, outImage)

	err := g.Validate()
	if err == nil {
		t.Fatal("expected an error for a block with no incoming text edge")
	}
	if !strings.Contains(err.Error(), "orphan-image has no incoming text edge") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdventureFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := NewAdventure(map[string]string{"Health": "Good"})
	if err := a.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	a.Graph.Start(ctx)
	a.Say("I try to cross the bridge")

	const prose = "[prose turn 1] [outline] world reacts to: [3rd-person] I try to cross the bridge"
	expect(t, a.Sinks["text"], prose)
	expect(t, a.Sinks["image"], "<png of dim stone bridge, torchlight>")
	// Seeded values reach the player before the first turn, then the turn's
	// update follows. Order matters: a status bar should not start empty.
	expect(t, a.Sinks["props"], "map[Health:Good]")
	expect(t, a.Sinks["props"], "map[Health:Good Turn:1]")
	expect(t, a.Sinks["audio"], "<narration of "+prose+">")
}

// The observer's steering has to reach the block whose output triggered it.
func TestNPCLiveObserverClosesTheLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := NewNPCLive(mockNPCConfig("refuse passage", blocks.InputScript{
		Lines:    []string{"persuade", "persuade harder", "persuade hardest"},
		Interval: 40 * time.Millisecond,
	}))
	if err := n.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	n.Graph.Start(ctx)

	var seen []string
	deadline := time.After(3 * time.Second)
	for {
		select {
		case v := <-n.Sinks["text"]:
			seen = append(seen, v)
			if strings.HasPrefix(v, "The bridge stays closed. (steered:") {
				if len(n.Observer.Flags) == 0 {
					t.Error("steering fired without the observer flagging anything")
				}
				return
			}
		case <-deadline:
			t.Fatalf("steering never reached the live block\nsaw:\n%s", strings.Join(seen, "\n"))
		}
	}
}

func mockNPCConfig(guardrail string, script blocks.InputScript) NPCLiveConfig {
	return NPCLiveConfig{
		Live:      mock.Live{},
		Tool:      mock.Tool{},
		Image:     mock.Image{},
		Guardrail: guardrail,
		Scenario:  "You are the bridge keeper. Do not concede passage.",
		Script:    script,
	}
}

func expect(t *testing.T, ch chan string, want string) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Errorf("got  %q\nwant %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("never received %q", want)
	}
}

// A genre that forgets to wire the current values must fail validation, rather
// than quietly leaving a block to rely on the model remembering them.
func TestExtractionRequiresPropsEdge(t *testing.T) {
	g := ports.NewGraph("no-props")
	player := blocks.NewPlayerInputText("player")
	outline := blocks.NewDummyExtraction("outline")
	outText := blocks.NewPlayerOutputText("out-text")

	g.ConnectTextOut(player, outline)
	g.ConnectTextOut(outline, outText)

	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "outline has no incoming props edge") {
		t.Fatalf("expected a missing props edge to be caught, got: %v", err)
	}
}

// A block whose props input is required must not act on a player action before
// the current values have reached it. Without that guarantee it decides from
// defaults and then overwrites the real values on the way out — which shows up
// as a status bar that resets on the first turn, intermittently.
func TestSeededStatusSurvivesTheFirstTurn(t *testing.T) {
	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		a := NewAdventure(map[string]string{"Health": "Wounded", "Gold": "12"})
		a.Graph.Start(ctx)
		a.Say("I press on")

		deadline := time.After(2 * time.Second)
		var sawTurn bool
		for !sawTurn {
			select {
			case v := <-a.Sinks["props"]:
				if !strings.Contains(v, "Turn:1") {
					continue
				}
				sawTurn = true
				if !strings.Contains(v, "Health:Wounded") {
					cancel()
					t.Fatalf("run %d: seeded value lost on the first turn: %s", i, v)
				}
			case <-deadline:
				cancel()
				t.Fatalf("run %d: no turn-1 props", i)
			}
		}
		cancel()
	}
}
