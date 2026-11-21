package blocks_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// outlineSchema is a reply with two texts and a status object, as Adventure's
// Outline gives.
func outlineSchema(status ...string) adapters.Schema {
	fields := map[string]any{}
	for _, name := range status {
		fields[name] = map[string]any{"type": "string"}
	}
	return adapters.Schema{Name: "outline", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"outline":      map[string]any{"type": "string"},
			"image_prompt": map[string]any{"type": "string"},
			"status": map[string]any{
				"type":                 "object",
				"properties":           fields,
				"required":             toAny(status),
				"additionalProperties": false,
			},
		},
		"required":             []any{"outline", "image_prompt", "status"},
		"additionalProperties": false,
	}}
}

func toAny(names []string) []any {
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = n
	}
	return out
}

var outlineResult = blocks.Result{Text: "outline", SecondaryText: "image_prompt", Props: "status"}

// structuredBlock is one started block and what it puts out.
type structuredBlock struct {
	*blocks.Structured
	text, secondary <-chan string
	props           <-chan ports.PropMap
	states          <-chan ports.State
}

func newStructured(thread *blocks.Thread, status ...string) structuredBlock {
	b := blocks.NewStructured("outline", thread, "outline the turn", outlineSchema(status...), outlineResult)
	b.Backoff = []time.Duration{0, 0}
	return structuredBlock{
		Structured: b,
		text:       b.TextOutPort(),
		secondary:  b.SecondaryTextOutPort(),
		props:      b.PropsOutPort(),
		states:     b.StateOutPort(),
	}
}

// Outline answers structured on the story thread, ahead of Expand: its text
// goes on, its image prompt apart, its status as props, and Expand's plain
// call continues the same thread. Each call carries the current status, so
// the thread's memory is not trusted for it.
func TestStructuredTurnOnTheSharedThread(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thread := blocks.NewThread("story-thread", mock.Thread{})
	outline := newStructured(thread, "Health")
	expand := startThreaded(t, thread, "expand")
	outline.Start(ctx)
	expand.Start(ctx)

	outline.PropsInPort() <- ports.PropMap{"Health": "badly wounded"}
	outline.TextInPort() <- "The player crosses."

	if got := receive(t, outline.text, "the outline"); got != "[outline 1] The player crosses." {
		t.Errorf("outline %q", got)
	}
	if got := receive(t, outline.secondary, "the image prompt"); got != "[image_prompt 1] The player crosses." {
		t.Errorf("image prompt %q", got)
	}
	if got := receive(t, outline.props, "the status"); got["Health"] != "badly wounded [turn 1]" || len(got) != 1 {
		t.Errorf("status %v, want the current value carried into the call", got)
	}

	expand.TextInPort() <- "[outline 1] The player crosses."
	if got := receive(t, expand.whole, "the prose"); !strings.HasPrefix(got, "[story turn 2]") {
		t.Errorf("expand said %q, want turn 2 of the same thread", got)
	}
	// The thread keeps Outline's whole JSON reply.
	if state := thread.ExportState(); !strings.Contains(state, `\"outline\":\"[outline 1] The player crosses.\"`) {
		t.Errorf("exported state %s lacks Outline's reply", state)
	}
}

// A reply that does not parse against the schema is never kept: the call is
// made again from the same continuation, and only the good reply is stored.
func TestStructuredRetriesABadReply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedStructured{replies: []string{
		`{"outline":"cut sh`,
		`{"outline":"x","image_prompt":"y","status":{}}`,
		`{"outline":"The bridge holds.","image_prompt":"a bridge","status":{"Health":"fine"}}`,
	}}
	thread := blocks.NewThread("story-thread", adapter)
	thread.RestoreState(`{"continuation":{"id":"resp_7"}}`)
	outline := newStructured(thread, "Health")
	outline.Start(ctx)

	outline.PropsInPort() <- ports.PropMap{"Health": "good"}
	outline.TextInPort() <- "go"
	if got := receive(t, outline.text, "the outline"); got != "The bridge holds." {
		t.Errorf("outline %q", got)
	}
	if from := adapter.continuations(); strings.Join(from, ",") != "resp_7,resp_7,resp_7" {
		t.Errorf("continued from %v, want resp_7 each time", from)
	}
	if state := thread.ExportState(); strings.Count(state, `"reply"`) != 1 || strings.Contains(state, "cut sh") {
		t.Errorf("exported state %s, want only the good reply kept", state)
	}
	expectPhases(t, outline.states, ports.PhaseReady, ports.PhaseWorking, ports.PhaseReady)
}

// A provider that continues in place, as Mistral appends to its stored
// conversation, has kept the bad reply on its side, so the retry goes to a
// fresh thread seeded with the transcript, which holds good replies only.
func TestStructuredRetryLeavesAThreadThatKeptTheBadReply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedStructured{inPlace: true, replies: []string{
		"not json",
		`{"outline":"The bridge holds.","image_prompt":"a bridge","status":{"Health":"fine"}}`,
	}}
	thread := blocks.NewThread("story-thread", adapter)
	thread.RestoreState(`{"continuation":{"id":"conv_1"},"transcript":[{"input":"before","reply":"{}"}]}`)
	outline := newStructured(thread, "Health")
	outline.Start(ctx)

	outline.PropsInPort() <- ports.PropMap{"Health": "good"}
	outline.TextInPort() <- "go"
	receive(t, outline.text, "the outline")
	if from := adapter.continuations(); strings.Join(from, ",") != "conv_1," {
		t.Errorf("continued from %q, want conv_1, then a fresh thread", from)
	}
	if seeded := adapter.turns[1].Transcript; len(seeded) != 1 || seeded[0].Input != "before" {
		t.Errorf("the fresh thread was seeded with %+v, want the transcript of good replies", seeded)
	}
	// The good reply's continuation is kept again.
	if state := thread.ExportState(); !strings.Contains(state, `"id":"conv_1"`) {
		t.Errorf("exported state %s, want the continuation the good reply made", state)
	}
}

// An empty structured reply does not parse, so it is a bad reply too, and the
// game goes on.
func TestStructuredEmptyReplyIsBad(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedStructured{replies: []string{
		"",
		`{"outline":"On.","image_prompt":"p","status":{"Health":"fine"}}`,
	}}
	outline := newStructured(blocks.NewThread("story-thread", adapter), "Health")
	outline.Start(ctx)

	outline.PropsInPort() <- ports.PropMap{"Health": "good"}
	outline.TextInPort() <- "go"
	if got := receive(t, outline.text, "the outline"); got != "On." {
		t.Errorf("outline %q", got)
	}
}

// After three bad replies the turn is dropped: nothing goes on, the block is
// ready again and says the turn failed, and the thread hands the turn past
// Expand, so the next turn's Outline does not wait for it.
func TestStructuredDropsTheTurnAfterThreeBadReplies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &scriptedStructured{
		replies: []string{
			"not json",
			`{"outline":"x"}`,
			"",
			`{"outline":"Again.","image_prompt":"p","status":{"Health":"fine"}}`,
		},
		refuse: 3,
	}
	thread := blocks.NewThread("story-thread", adapter)
	outline := newStructured(thread, "Health")
	expand := startThreaded(t, thread, "expand")
	outline.Start(ctx)
	expand.Start(ctx)

	outline.PropsInPort() <- ports.PropMap{"Health": "good"}
	outline.TextInPort() <- "first"
	expectPhases(t, outline.states, ports.PhaseReady, ports.PhaseWorking)
	dropped := receive(t, outline.states, "the dropped turn")
	if dropped.Phase != ports.PhaseReady || dropped.Reason != ports.ReasonTurnFailed {
		t.Fatalf("state %+v, want ready with %q", dropped, ports.ReasonTurnFailed)
	}

	outline.TextInPort() <- "second"
	if got := receive(t, outline.text, "the next outline"); got != "Again." {
		t.Errorf("outline %q", got)
	}
	select {
	case got := <-outline.secondary:
		if got != "p" {
			t.Errorf("image prompt %q, want only the second turn's", got)
		}
	case <-time.After(time.Second):
		t.Error("no image prompt for the second turn")
	}
	if n := adapter.calls(); n != 4 {
		t.Errorf("%d calls, want three for the dropped turn and one for the next", n)
	}
}

// A game without status fields has a schema without status, and puts out no
// props.
func TestStructuredWithoutStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	schema := adapters.Schema{Name: "outline", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"outline":      map[string]any{"type": "string"},
			"image_prompt": map[string]any{"type": "string"},
		},
		"required":             []any{"outline", "image_prompt"},
		"additionalProperties": false,
	}}
	b := blocks.NewStructured("outline", blocks.NewThread("story-thread", mock.Thread{}), "", schema, outlineResult)
	text, props := b.TextOutPort(), b.PropsOutPort()
	b.Start(ctx)

	b.PropsInPort() <- ports.PropMap{}
	b.TextInPort() <- "go"
	receive(t, text, "the outline")
	select {
	case got := <-props:
		t.Errorf("props %v, want none", got)
	case <-time.After(50 * time.Millisecond):
	}
}

// scriptedStructured answers its replies in turn; call number refuse is
// refused by the provider instead. inPlace continues on the continuation it
// was given, as a stored conversation does, and starts "conv_1" without one.
type scriptedStructured struct {
	mu      sync.Mutex
	replies []string
	refuse  int
	inPlace bool
	turns   []adapters.ThreadTurn
}

func (s *scriptedStructured) Model() string { return "fake-story" }

func (s *scriptedStructured) Continue(_ context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turns = append(s.turns, turn)
	n := len(s.turns)
	if n == s.refuse {
		return adapters.ThreadReply{}, adapters.BadReply("fake", "refused")
	}
	reply := s.replies[min(n, len(s.replies))-1]
	next := adapters.Continuation{ID: "resp_next", Model: "fake-story"}
	if s.inPlace {
		next.ID = turn.Continuation.ID
		if next.ID == "" {
			next.ID = "conv_1"
		}
	}
	return adapters.ThreadReply{
		Text:         reply,
		Usage:        adapters.Usage{Model: "fake-story", InputTokens: 10, OutputTokens: 5},
		Continuation: next,
	}, nil
}

func (s *scriptedStructured) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.turns)
}

func (s *scriptedStructured) continuations() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, c := range s.turns {
		out = append(out, c.Continuation.ID)
	}
	return out
}
