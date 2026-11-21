package blocks_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

var heardSchema = adapters.Schema{Name: "spoken", Body: map[string]any{
	"type": "object",
	"properties": map[string]any{
		"heard":  map[string]any{"type": "string"},
		"action": map[string]any{"type": "string"},
	},
	"required":             []string{"heard", "action"},
	"additionalProperties": false,
}}

type hearingBlock struct {
	*blocks.HearingCall
	action, heard <-chan string
	failure       <-chan ports.Failure
	states        <-chan ports.State
	usage         <-chan ports.Usage
}

func startHearing(t *testing.T, hearing adapters.Hearing) hearingBlock {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	b := blocks.NewHearingCall("rephrase-audio", hearing, "hear it", heardSchema,
		blocks.Result{Text: "action", SecondaryText: "heard"})
	b.Backoff = make([]time.Duration, len(b.Backoff))
	hb := hearingBlock{
		HearingCall: b,
		action:      b.TextOutPort(),
		heard:       b.SecondaryTextOutPort(),
		failure:     b.FailureOutPort(),
		states:      b.StateOutPort(),
		usage:       b.UsageOutPort(),
	}
	b.Start(ctx)
	return hb
}

// The block hands its prompt, the clip and the schema to the model, and sends
// what was heard and the action made of it on, billing the call.
func TestHearingSendsWhatWasHeardAndTheAction(t *testing.T) {
	hearing := &scriptedHearing{replies: []string{"```json\n" + `{"heard":"ich öffne die Tür","action":"The player tries to open the door."}` + "\n```"}}
	b := startHearing(t, hearing)

	b.AudioInPort() <- loud

	if got := receive(t, b.heard, "what was heard"); got != "ich öffne die Tür" {
		t.Errorf("heard %q", got)
	}
	if got := receive(t, b.action, "the action"); got != "The player tries to open the door." {
		t.Errorf("action %q", got)
	}
	if calls := hearing.seen(); len(calls) != 1 || calls[0].system != "hear it" || string(calls[0].clip) != string(loud) || calls[0].schema != "spoken" {
		t.Errorf("the model was asked %+v", calls)
	}
	if used := receive(t, b.usage, "the usage"); used.Model != "fake-hearing" || used.AudioInputTokens != 12 {
		t.Errorf("billed %+v", used)
	}
	expectPhases(t, b.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// A reply that does not fit the schema is asked again; three fail the call,
// with nothing sent on and an empty input on the failure output, as the clip
// is not kept.
func TestHearingFailsAfterThreeBadReplies(t *testing.T) {
	hearing := &scriptedHearing{replies: []string{`{"said":"x"}`, "not json", `{"heard":"x","action":""}`, `{"heard":"x","action":"y"}`}}
	b := startHearing(t, hearing)

	b.AudioInPort() <- loud

	failed := receive(t, b.failure, "the failure")
	if failed.Input != "" || failed.Reason != ports.ReasonBadReply {
		t.Errorf("failure %+v, want a bad reply with no input", failed)
	}
	if n := len(hearing.seen()); n != 3 {
		t.Errorf("the model was asked %d times, want 3", n)
	}
	expectNothing(t, b.heard, b.action)
}

// A clip in which nothing was heard starts no turn: the call fails at once.
func TestHearingNothingFailsTheCall(t *testing.T) {
	hearing := &scriptedHearing{replies: []string{`{"heard":"  ","action":"The player says nothing."}`}}
	b := startHearing(t, hearing)

	b.AudioInPort() <- loud

	if failed := receive(t, b.failure, "the failure"); failed.Reason != ports.ReasonNothingHeard {
		t.Errorf("failure %+v, want nothing heard", failed)
	}
	if n := len(hearing.seen()); n != 1 {
		t.Errorf("the model was asked %d times, want 1", n)
	}
	expectNothing(t, b.heard, b.action)
}

// The switch decides whether the player may speak; a clip that reached the
// block is heard even after it goes off.
func TestHearingFinishesAClipAfterTheSwitchGoesOff(t *testing.T) {
	hearing := &scriptedHearing{replies: []string{`{"heard":"hello","action":"The player greets."}`}}
	b := startHearing(t, hearing)

	b.AudioInPort() <- loud
	b.SetActive(false)

	if got := receive(t, b.heard, "what was heard"); got != "hello" {
		t.Errorf("heard %q", got)
	}
}

func expectNothing(t *testing.T, outs ...<-chan string) {
	t.Helper()
	for _, out := range outs {
		select {
		case v := <-out:
			t.Errorf("a failed call sent %q", v)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

type heardCall struct {
	system string
	clip   adapters.Clip
	schema string
}

// scriptedHearing answers its replies in turn, the last one again after.
type scriptedHearing struct {
	replies []string

	mu    sync.Mutex
	calls []heardCall
}

func (s *scriptedHearing) Hear(_ context.Context, system string, clip adapters.Clip, schema adapters.Schema) (string, adapters.Usage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, heardCall{system: system, clip: clip, schema: schema.Name})
	reply := s.replies[min(len(s.calls), len(s.replies))-1]
	return reply, adapters.Usage{Model: "fake-hearing", InputTokens: 7, AudioInputTokens: 12, OutputTokens: 5}, nil
}

func (s *scriptedHearing) seen() []heardCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]heardCall(nil), s.calls...)
}

// A clip a little loud, as speech is.
var loud = ports.AudioChunk{0x00, 0x10, 0x00, 0x10}

// A silent clip is heard as nothing without asking the model, which may make
// words up for silence.
func TestHearingSilenceAsksNoModel(t *testing.T) {
	hearing := &scriptedHearing{replies: []string{`{"heard":"I open the door","action":"The player opens the door."}`}}
	b := startHearing(t, hearing)

	b.AudioInPort() <- ports.AudioChunk(make([]byte, 4800))

	if failed := receive(t, b.failure, "the failure"); failed.Reason != ports.ReasonNothingHeard {
		t.Errorf("failure %+v, want nothing heard", failed)
	}
	if n := len(hearing.seen()); n != 0 {
		t.Errorf("the model was asked %d times, want none", n)
	}
}
