package blocks_test

import (
	"context"
	"testing"
	"time"

	"engine/internal/blocks"
	"engine/internal/ports"
)

// A keep reports working until its value arrives, hands it on under its key,
// and exports it; a resumed keep hands on what it kept without being asked.
func TestKeepHoldsItsValueAcrossAResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keep := blocks.NewKeep("adapted-style", "style", true)
	out, states := keep.PropsOutPort(), keep.StateOutPort()
	keep.Start(ctx)
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking)
	keep.TextInPort() <- "woodcut, gentle"
	if got := receive(t, out, "the value"); got["style"] != "woodcut, gentle" {
		t.Errorf("handed on %v", got)
	}
	expectPhases(t, states, ports.PhaseReady)

	resumed := blocks.NewKeep("adapted-style", "style", true)
	resumed.RestoreState(keep.ExportState())
	out, states = resumed.PropsOutPort(), resumed.StateOutPort()
	resumed.Start(ctx)
	if got := receive(t, out, "the kept value"); got["style"] != "woodcut, gentle" {
		t.Errorf("a resumed keep handed on %v", got)
	}
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// A keep that expects nothing is ready at once and hands on nothing.
func TestKeepExpectingNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keep := blocks.NewKeep("condensed-scenario", "scenario", false)
	out, states := keep.PropsOutPort(), keep.StateOutPort()
	keep.Start(ctx)
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	select {
	case got := <-out:
		t.Errorf("handed on %v", got)
	case <-time.After(30 * time.Millisecond):
	}
	if state := keep.ExportState(); state != "" {
		t.Errorf("exported %q", state)
	}
}

// Compose pairs each text on its first input with the next on its second, in
// order, whichever arrives first, with the latest props.
func TestComposePairsInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	compose := blocks.NewCompose("image-prompt", func(p blocks.Parts) string {
		return p.Text + "|" + p.Secondary + "|" + p.Props["style"]
	})
	out := compose.TextOutPort()
	compose.Start(ctx)

	compose.PropsInPort() <- ports.PropMap{"style": "woodcut"}
	compose.SecondaryTextInPort() <- "outline 1"
	compose.SecondaryTextInPort() <- "outline 2"
	compose.TextInPort() <- "prompt 1"
	if got := receive(t, out, "the first prompt"); got != "prompt 1|outline 1|woodcut" {
		t.Errorf("built %q", got)
	}
	compose.TextInPort() <- "prompt 2"
	if got := receive(t, out, "the second prompt"); got != "prompt 2|outline 2|woodcut" {
		t.Errorf("built %q", got)
	}
}

// An error sink puts each text on the `error` stream with its code.
func TestErrorSinkReportsItsCode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sink := blocks.NewErrorSink("turn-failed", "turn_failed")
	sink.Start(ctx)
	sink.TextInPort() <- `I "cross"`
	if sink.Stream() != "error" {
		t.Errorf("stream %q", sink.Stream())
	}
	if got := receive(t, sink.Arrivals(), "the report"); got != `{"code":"turn_failed","text":"I \"cross\""}` {
		t.Errorf("reported %s", got)
	}
}

// The player's input is held until every release source has worked and is
// ready again, or one reports a failure. What was typed meanwhile is sent on
// release, and reports after it never hold a source up.
func TestPlayerInputReleases(t *testing.T) {
	for name, reports := range map[string][]ports.State{
		"all done": {
			{Node: "outline", Phase: ports.PhaseReady},
			{Node: "expand", Phase: ports.PhaseReady},
			{Node: "outline", Phase: ports.PhaseWorking},
			{Node: "outline", Phase: ports.PhaseReady},
			{Node: "expand", Phase: ports.PhaseWorking},
			{Node: "expand", Phase: ports.PhaseReady},
		},
		"one failed": {
			{Node: "outline", Phase: ports.PhaseWorking},
			{Node: "outline", Phase: ports.PhaseReady, Reason: ports.ReasonBadReply},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			player := blocks.NewPlayerInputText("player-input-text")
			out := player.TextOutPort()
			player.ExpectReleases(2)
			player.Start(ctx)
			player.Say("I look around")

			for i, report := range reports {
				player.StateInPort() <- report
				if i == len(reports)-2 {
					select {
					case got := <-out:
						t.Fatalf("released early with %q", got)
					case <-time.After(20 * time.Millisecond):
					}
				}
			}
			if got := receive(t, out, "the held line"); got != "I look around" {
				t.Errorf("released %q", got)
			}
			// Later reports are drained: more than the port's buffer never blocks.
			for range 20 {
				select {
				case player.StateInPort() <- ports.State{Node: "expand", Phase: ports.PhaseReady}:
				case <-time.After(time.Second):
					t.Fatal("a report after the release was not taken")
				}
			}
		})
	}
}
