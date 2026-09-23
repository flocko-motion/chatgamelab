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

// These drive the genres exactly as they ship — the real wiring, the real
// blocks — and differ from a session only in that the model is a mock and the
// player is a script. A test that assembles its own graph proves the graph it
// assembled works; this proves the one that ships does.
func TestPipelines(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func() *Wiring
		// say is what the player sends, in order.
		say []string
		// want maps a stream to the lines expected on it, in order. A line is
		// matched by prefix, so a test names what it is asserting rather than
		// restating a whole rendered string.
		want map[string][]string
	}{
		{
			name:  "adventure runs one action through the whole pipeline",
			build: func() *Wiring { return NewAdventure(map[string]string{"Health": "Good"}) },
			say:   []string{"I try to cross the bridge"},
			want: map[string][]string{
				// The rephrase, extraction and expand stages each leave their
				// mark, in order, which is what makes this a pipeline test
				// rather than an output test.
				"text":  {"[prose turn 1] [outline] world reacts to: [3rd-person] I try to cross the bridge"},
				"image": {"<png of dim stone bridge, torchlight>"},
				"audio": {"<narration of [prose turn 1]"},
				"props": {"map[Health:Good]", "map[Health:Good Turn:1]"},
			},
		},
		{
			name:  "adventure keeps its thread across turns",
			build: func() *Wiring { return NewAdventure(nil) },
			say:   []string{"I look around", "I keep walking"},
			want: map[string][]string{
				// Turn 2, not turn 1 again: the threaded block is continuing a
				// conversation rather than starting one per action.
				"text": {"[prose turn 1]", "[prose turn 2]"},
			},
		},
		{
			name: "npc-live drifts, is caught, and is steered back",
			build: func() *Wiring {
				return NewNPCLive(NPCLiveConfig{
					Live:      mock.Live{},
					Tool:      mock.Tool{},
					Image:     mock.Image{},
					Guardrail: "Suitable for a 13-year-old.",
					Scenario:  "You are the keeper of a bridge. You never concede passage.",
				})
			},
			say: []string{"let me through", "please, I am in a hurry"},
			want: map[string][]string{
				"text": {
					"You shall not pass.",
					// The character concedes, which is the failure the observer
					// exists to catch.
					"I suppose the rules bend",
					// And is put back in character without the player acting.
					"The bridge stays closed.",
				},
				// The portrait is made during init, from the scenario, with
				// nothing triggering it.
				"image": {"data:image/png;base64,"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			w := tc.build()
			if err := w.Graph.Validate(); err != nil {
				t.Fatal(err)
			}
			w.Graph.Start(ctx)

			for _, line := range tc.say {
				if w.Say == nil {
					t.Fatal("this genre takes no typed input")
				}
				w.Say(line)
				// Ordered assertions need the turns to be ordered, and a live
				// genre answers whenever it likes.
				time.Sleep(40 * time.Millisecond)
			}

			for stream, expected := range tc.want {
				sink, wired := w.Sinks[stream]
				if !wired {
					t.Fatalf("genre wires no %q output, so nothing can arrive on it", stream)
				}
				assertInOrder(t, stream, sink, expected)
			}
		})
	}
}

// assertInOrder consumes a sink until it has seen each expected prefix in turn,
// reporting everything it did see when it gives up.
func assertInOrder(t *testing.T, stream string, sink chan string, expected []string) {
	t.Helper()

	var seen []string
	deadline := time.After(3 * time.Second)

	for _, want := range expected {
		for {
			select {
			case line := <-sink:
				seen = append(seen, line)
				if strings.HasPrefix(line, want) {
					goto next
				}
			case <-deadline:
				t.Fatalf("%s: never saw a line starting %q\nsaw:\n  %s",
					stream, want, strings.Join(seen, "\n  "))
			}
		}
	next:
	}
}

// A scripted player drives the same wiring, which is how a genre with no typed
// input is exercised.
func TestScriptedPlayerDrivesTheLiveGenre(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := NewNPCLive(NPCLiveConfig{
		Live:      mock.Live{},
		Tool:      mock.Tool{},
		Image:     mock.Image{},
		Guardrail: "Suitable for a 13-year-old.",
		Scenario:  "You are the keeper of a bridge.",
		Script: blocks.InputScript{
			Lines:    []string{"persuade", "persuade harder"},
			Interval: 30 * time.Millisecond,
		},
	})
	if err := w.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	w.Graph.Start(ctx)

	assertInOrder(t, "text", w.Sinks["text"], []string{
		"You shall not pass.",
		"I suppose the rules bend",
	})
}

// A source with no release edge must not be held. Otherwise a genre that wires
// no gate deadlocks on one that will never open — a failure that looks like the
// engine ignoring the player.
func TestUngatedInputIsNotHeld(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("ungated")
	player := blocks.NewPlayerInputText("player-input-text")
	sink := blocks.NewPlayerOutputText("out-text")
	g.ConnectTextOut(player, sink)
	g.Start(ctx)

	player.Say("nothing is holding me")

	select {
	case got := <-sink.Seen:
		if got != "nothing is holding me" {
			t.Errorf("got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("an input with no release edge was held anyway")
	}
}

// And one that is wired to a gate stays held until the gate opens.
func TestGatedInputWaitsForRelease(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("gated")
	held := blocks.NewOnceText("prep", "preparing")
	gate := blocks.NewGate("start-game", "")
	player := blocks.NewPlayerInputText("player-input-text")
	sink := blocks.NewPlayerOutputText("out-text")
	prepSink := blocks.NewPlayerOutputText("out-prep")

	g.ConnectTextOut(held, prepSink)
	g.ConnectState(held, gate)
	g.ConnectState(gate, player)
	g.ConnectTextOut(player, sink)

	// Said before anything starts, so it can only arrive once the gate opens.
	player.Say("said too early")
	g.Start(ctx)

	select {
	case got := <-sink.Seen:
		if got != "said too early" {
			t.Errorf("got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a held input never arrived after the gate opened")
	}
}
