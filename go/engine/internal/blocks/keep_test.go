package blocks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"engine/internal/blocks"
	"engine/internal/ports"
)

// A keep sends its request, reports working until the answer arrives, hands
// it on under its key, and exports it; a resumed keep hands on what it kept
// without asking.
func TestKeepHoldsItsValueAcrossAResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keep := blocks.NewKeep("adapted-style", "style", "Image Style: woodcut")
	asked, out, states := keep.TextOutPort(), keep.PropsOutPort(), keep.StateOutPort()
	keep.Start(ctx)
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking)
	if got := receive(t, asked, "the request"); got != "Image Style: woodcut" {
		t.Errorf("asked %q", got)
	}
	keep.TextInPort() <- "woodcut, gentle"
	if got := receive(t, out, "the value"); got["style"] != "woodcut, gentle" {
		t.Errorf("handed on %v", got)
	}
	expectPhases(t, states, ports.PhaseReady)

	resumed := blocks.NewKeep("adapted-style", "style", "Image Style: woodcut")
	resumed.RestoreState(keep.ExportState())
	asked, out, states = resumed.TextOutPort(), resumed.PropsOutPort(), resumed.StateOutPort()
	resumed.Start(ctx)
	if got := receive(t, out, "the kept value"); got["style"] != "woodcut, gentle" {
		t.Errorf("a resumed keep handed on %v", got)
	}
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	select {
	case got := <-asked:
		t.Errorf("a keep holding its value asked again: %q", got)
	case <-time.After(30 * time.Millisecond):
	}
}

// A session stored after the request went out and before the answer came
// asks again when it is resumed, so it never waits for an answer that is gone.
func TestKeepAsksAgainWhenTheAnswerNeverCame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := blocks.NewKeep("condensed-scenario", "scenario", "Scenario: long")
	asked := first.TextOutPort()
	first.Start(ctx)
	receive(t, asked, "the first request")
	stored := first.ExportState()
	cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resumed := blocks.NewKeep("condensed-scenario", "scenario", "Scenario: long")
	if stored != "" {
		resumed.RestoreState(stored)
	}
	asked = resumed.TextOutPort()
	resumed.Start(ctx2)
	if got := receive(t, asked, "the request again"); got != "Scenario: long" {
		t.Errorf("asked %q", got)
	}
}

// A keep with an empty request asks nothing, is ready at once and hands on
// nothing.
func TestKeepWithNothingToAsk(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keep := blocks.NewKeep("condensed-scenario", "scenario", "")
	asked, out, states := keep.TextOutPort(), keep.PropsOutPort(), keep.StateOutPort()
	keep.Start(ctx)
	expectPhases(t, states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
	select {
	case got := <-out:
		t.Errorf("handed on %v", got)
	case got := <-asked:
		t.Errorf("asked %q", got)
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
	sink.FailureInPort() <- ports.Failure{Node: "rephrase", Input: `I "cross"`, Reason: ports.ReasonFailed}
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

// A failure edge may lead into any block that takes text: it receives the
// input the failed call was given.
func TestFailureIntoABlockThatTakesText(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := ports.NewGraph("failure-to-text")
	player := blocks.NewPlayerInputText("player")
	rephrase := blocks.NewToolCall("rephrase", &scriptedTool{errs: []error{errors.New("refused")}}, "rephrase it")
	said := blocks.NewPlayerOutputText("out-text", "text")
	fallback := blocks.NewPlayerOutputText("out-fallback", "fallback")
	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, said)
	g.ConnectFailureText(rephrase, fallback)
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	g.Start(ctx)

	player.Say("I cross")
	if got := receive(t, fallback.Arrivals(), "the failed line"); got != "I cross" {
		t.Errorf("the text block received %q, want the failed call's input", got)
	}
}
