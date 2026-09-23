package genre

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

// Every genre's graph must be complete: types settle whether an edge is legal,
// this settles whether anything was left unwired.
func TestGraphsAreValid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		graph *ports.Graph
	}{
		{"adventure", NewAdventure(nil).Graph},
		{"npc-live", mustNPCLive(t, mockNPCConfig("stay in character")).Graph},
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
	outText := blocks.NewPlayerOutputText("out-text", "text")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")

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
	expect(t, a.Sinks()["text"], prose)
	expect(t, a.Sinks()["image"], "<png of dim stone bridge, torchlight>")
	// Seeded values reach the player before the first turn, then the turn's
	// update follows. Order matters: a status bar should not start empty.
	expect(t, a.Sinks()["props"], "map[Health:Good]")
	expect(t, a.Sinks()["props"], "map[Health:Good Turn:1]")
	expect(t, a.Sinks()["audio"], "<narration of "+prose+">")
}

// The observer's steering has to reach the block whose output triggered it.
func TestNPCLiveObserverClosesTheLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := mustNPCLive(t, mockNPCConfig("refuse passage"))
	if err := n.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	n.Graph.Start(ctx)
	connect(t, ctx, n)

	var seen []string
	deadline := time.After(5 * time.Second)
	for {
		select {
		case v := <-n.Sinks()["text"]:
			seen = append(seen, v)
			if strings.HasPrefix(v, "The bridge stays closed. (steered:") {
				return
			}
		case <-deadline:
			t.Fatalf("steering never reached the live block\nsaw:\n%s", strings.Join(seen, "\n"))
		}
	}
}

// A muted observer must still read its input, or the block feeding it blocks on
// a full channel and takes the conversation down with it.
func TestMutedObserverJudgesNothingAndBlocksNobody(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	cfg.MuteObserver = true
	n := mustNPCLive(t, cfg)
	n.Graph.Start(ctx)
	connect(t, ctx, n)

	// The character drifts all the way to capitulation, which an active
	// observer would have flagged several lines earlier.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case v := <-n.Sinks()["text"]:
			if !strings.Contains(v, "won't stop you") {
				continue
			}
			return
		case <-deadline:
			t.Fatal("the conversation stalled, which is what a full observer channel looks like")
		}
	}
}

// An override replaces a genre's own prompt; a misspelling is refused rather
// than silently leaving the default in place.
func TestPromptOverride(t *testing.T) {
	cfg := mockNPCConfig("refuse passage")
	cfg.PromptOverride = map[string]string{PromptPortrait: "A woodcut of: "}

	w := mustNPCLive(t, cfg)
	if got := w.Prompts[PromptPortrait]; got != "A woodcut of: " {
		t.Errorf("override ignored, prompt is %q", got)
	}
	if w.Prompts[PromptObserver] == "" {
		t.Error("overriding one prompt dropped the others")
	}

	cfg.PromptOverride = map[string]string{"portraitt": "typo"}
	if _, err := NewNPCLive(cfg); err == nil {
		t.Fatal("a misspelled prompt name was accepted")
	}
}

func mockNPCConfig(guardrail string) NPCLiveConfig {
	return NPCLiveConfig{
		Live:       mock.Live{},
		Tool:       mock.Tool{},
		Image:      mock.Image{},
		Guardrail:  guardrail,
		Scenario:   "You are the bridge keeper. Do not concede passage.",
		InitPrompt: "Begin. Greet whoever has arrived.",
	}
}

func mustNPCLive(t *testing.T, cfg NPCLiveConfig) *Wiring {
	t.Helper()
	w, err := NewNPCLive(cfg)
	if err != nil {
		t.Fatalf("wiring npc-live: %v", err)
	}
	return w
}

// connect stands in for the browser. A live conversation is created by
// answering an offer, so nothing on this genre happens until one arrives.
func connect(t *testing.T, ctx context.Context, w *Wiring) {
	t.Helper()
	if _, err := w.Live.Connect(ctx, "v=0 mock offer"); err != nil {
		t.Fatalf("connect: %v", err)
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
	outText := blocks.NewPlayerOutputText("out-text", "text")

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
			case v := <-a.Sinks()["props"]:
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

// The portrait is made once because the prompt is emitted once. Once-ness is a
// property of the wiring rather than of the image block, so play can run for as
// many turns as it likes without a second picture being bought.
func TestPortraitIsMadeOnceHoweverLongPlayRuns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	counter := &countingImage{}
	cfg.Image = counter

	n := mustNPCLive(t, cfg)
	if err := n.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	n.Graph.Start(ctx)
	connect(t, ctx, n)

	select {
	case <-n.Sinks()["image"]:
	case <-time.After(3 * time.Second):
		t.Fatal("no portrait, and nobody spoke to trigger one")
	}

	// Several turns of play, so a block generating per input would have been
	// caught by now.
	for turns := 0; turns < 3; turns++ {
		select {
		case <-n.Sinks()["text"]:
		case <-time.After(3 * time.Second):
			t.Fatalf("the conversation stopped after %d turns", turns)
		}
	}

	if got := counter.calls(); got != 1 {
		t.Errorf("the image adapter was called %d times, want 1", got)
	}
	select {
	case v := <-n.Sinks()["image"]:
		t.Errorf("a second portrait arrived: %q", v)
	default:
	}
}

type countingImage struct {
	mu sync.Mutex
	n  int
}

func (c *countingImage) Generate(ctx context.Context, ask adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return mock.Image{}.Generate(ctx, ask)
}

func (c *countingImage) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// The observer's findings are wired to nothing, and an output with no edge
// reaches nothing. It steers the character, and that correction is the only
// thing a player ever sees of it.
//
// This is a regression test with a history. The findings used to leave the
// block through a public channel that no edge described, which put them on the
// player's stream while the drawing showed them going only to the character.
func TestObserverFindingsGoNowhere(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := mustNPCLive(t, mockNPCConfig("refuse passage"))
	if err := n.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, wired := n.Sinks()["flag"]; wired {
		t.Error("the genre still advertises a stream for what the observer found")
	}
	n.Graph.Start(ctx)
	connect(t, ctx, n)

	// Play until the observer has certainly judged and steered, then check that
	// nothing it found reached the one place a player reads.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case v := <-n.Sinks()["text"]:
			if strings.Contains(v, "VIOLATION") {
				t.Fatalf("a finding reached the conversation: %q", v)
			}
			if strings.HasPrefix(v, "The bridge stays closed. (steered:") {
				return
			}
		case <-deadline:
			t.Fatal("the observer never steered, so nothing was proven")
		}
	}
}

// Every stream a genre advertises must be produced by a node on its graph.
// Listing them by hand beside the wiring is what let the two disagree.
func TestEventSchemaComesFromTheGraph(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func() *Wiring
	}{
		{"adventure", func() *Wiring { return NewAdventure(nil) }},
		{"npc-live", func() *Wiring { return mustNPCLive(t, mockNPCConfig("refuse passage")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.build()
			if len(w.Sinks()) == 0 {
				t.Fatal("a genre with no streams cannot show anybody anything")
			}
			for stream, ch := range w.Sinks() {
				if ch == nil {
					t.Errorf("stream %q is advertised with nothing behind it", stream)
				}
			}
			// Derived, so the two cannot drift: the same call is the only
			// source of both.
			for stream := range w.Graph.Sinks() {
				if _, found := w.Sinks()[stream]; !found {
					t.Errorf("the graph produces %q but the genre does not advertise it", stream)
				}
			}
		})
	}
}

// A conversation nobody is having is closed, and picked up again when somebody
// comes back. Both halves matter: closing stops a genre billed by the minute
// running up a charge for an empty room, and resuming has to leave the
// character remembering what was said, or the saving costs the game.
func TestIdleConversationPausesAndResumes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opener := &recordingLive{opened: make(chan adapters.LiveConfig, 4), inner: mock.Live{}}
	cfg := mockNPCConfig("refuse passage")
	cfg.Live = opener
	cfg.IdleAfter = 150 * time.Millisecond

	n := mustNPCLive(t, cfg)
	if err := n.Graph.Validate(); err != nil {
		t.Fatal(err)
	}

	pauses := n.Live.Pauses()
	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	connect(t, ctx, n)

	first := opened(t, opener)
	if first.ResumeFrom != "" {
		t.Errorf("a first conversation claimed to continue %q", first.ResumeFrom)
	}

	// The mock says its piece and goes quiet, which is what a player wandering
	// off looks like from here.
	waitFor(t, pauses, "the conversation to be let go")

	waitFor(t, invites, "a second invitation after the pause")
	connect(t, ctx, n)

	second := opened(t, opener)
	if second.ResumeFrom == "" {
		t.Error("the conversation was reopened without naming the one it continues")
	}
	if len(second.History) == 0 {
		t.Fatal("the character was restored with no memory of what was said")
	}
	// Both halves, or a character resumes answering nothing.
	said := strings.Join(texts(second.History), " ")
	if !strings.Contains(said, "You shall not pass") {
		t.Errorf("the character's own words were not carried over: %q", said)
	}
}

// A player whose browser lost its audio comes back to a conversation still
// running on the provider's side, and their new offer continues it rather than
// waiting for the silence to prove they left. A page reload is the ordinary way
// this happens: it takes the peer connection with it and leaves the character
// talking to nobody.
func TestAReturningPlayerTakesOverTheConversation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opener := &recordingLive{opened: make(chan adapters.LiveConfig, 4), inner: mock.Live{}}
	cfg := mockNPCConfig("refuse passage")
	cfg.Live = opener
	// Long enough that nothing here is a pause: what ends the first
	// conversation must be the second offer and nothing else.
	cfg.IdleAfter = time.Hour

	n := mustNPCLive(t, cfg)
	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	connect(t, ctx, n)
	opened(t, opener)

	// The character has to have said something, or a continuation carries
	// nothing and proves nothing.
	waitForStanding(t, n, blocks.StandingOpen)
	waitForSpeech(t, n)

	// The reloaded page, offering again.
	connect(t, ctx, n)

	second := opened(t, opener)
	if second.ResumeFrom == "" {
		t.Error("the conversation was taken over without naming the one it continues")
	}
	if len(second.History) == 0 {
		t.Fatal("the character was restored with no memory of what was said")
	}
	if said := strings.Join(texts(second.History), " "); !strings.Contains(said, "You shall not pass") {
		t.Errorf("the character's own words were not carried over: %q", said)
	}
}

// Where the conversation stands is what a reloading page asks for, so it has to
// be right at each step: a live connection cannot be replayed, and a page that
// is told nothing waits for a moment that has already passed.
func TestStandingFollowsTheConversation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opener := &recordingLive{opened: make(chan adapters.LiveConfig, 4), inner: mock.Live{}}
	cfg := mockNPCConfig("refuse passage")
	cfg.Live = opener
	cfg.IdleAfter = 150 * time.Millisecond

	n := mustNPCLive(t, cfg)
	if got := n.Live.Standing(); got != blocks.StandingIdle {
		t.Errorf("before the game began the conversation stood at %q", got)
	}

	pauses := n.Live.Pauses()
	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	waitForStanding(t, n, blocks.StandingAwaiting)

	connect(t, ctx, n)
	waitForStanding(t, n, blocks.StandingOpen)

	waitFor(t, pauses, "the conversation to be let go")
	waitForStanding(t, n, blocks.StandingPaused)
}

// How a game's pictures look is the designer's to say, and the genre's job is
// to put it where an image model reads it as the treatment of everything before
// it: last, after the character being painted.
func TestTheImageStyleReachesThePortraitPrompt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	cfg.ImageStyle = "woodcut, heavy black lines"
	n := mustNPCLive(t, cfg)
	n.Graph.Start(ctx)

	prompt := portraitPrompt(t, n)
	if !strings.Contains(prompt, cfg.Scenario) {
		t.Errorf("the portrait was asked for without the character: %q", prompt)
	}
	if !strings.HasSuffix(prompt, "Style: "+cfg.ImageStyle) {
		t.Errorf("the game's style is not the last word in the prompt: %q", prompt)
	}
}

// A game naming no style still gets one. With nothing asked for the model
// chooses, and chooses differently every time — so a session's pictures would
// not look like each other, let alone like the game.
func TestAPortraitWithNoStyleStillAsksForOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := mustNPCLive(t, mockNPCConfig("refuse passage"))
	n.Graph.Start(ctx)

	if prompt := portraitPrompt(t, n); !strings.HasSuffix(prompt, DefaultImageStyle) {
		t.Errorf("a game that named no style was given none: %q", prompt)
	}
}

// portraitPrompt is what the stem actually asked the image model for, read back
// off the graph rather than rebuilt here — where a test rebuilding it would
// agree with itself whatever the genre did.
func portraitPrompt(t *testing.T, w *Wiring) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		detail, found := w.Graph.Inspect("portrait-prompt")
		if found && len(detail.Outputs) > 0 {
			return detail.Outputs[0].Value
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the portrait prompt was never emitted")
	return ""
}

// A character with nobody to answer has to speak first, or a player arrives at
// a portrait and a silence. The gate carries the cue that starts them off, and
// this is the whole of what makes a live game begin.
func TestTheCharacterOpensTheConversation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	cfg.IdleAfter = time.Hour
	n := mustNPCLive(t, cfg)

	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	connect(t, ctx, n)

	select {
	case line := <-n.Sinks()["text"]:
		if line == "" {
			t.Error("the character opened the conversation with nothing")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the character never spoke; nobody asked it to begin")
	}
}

// The cue waits for the session to say it is running. A provider that has
// created a conversation has not necessarily finished starting it, and an
// instruction written before then is one the session never had — which looks
// exactly like a character who was told to greet somebody and says nothing.
func TestTheOpeningCueWaitsForTheSessionToStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	cfg.Live = liveWithoutStart{inner: mock.Live{}}
	cfg.IdleAfter = time.Hour
	n := mustNPCLive(t, cfg)

	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	connect(t, ctx, n)

	select {
	case line := <-n.Sinks()["text"]:
		t.Fatalf("the character was cued before the session was running: %q", line)
	case <-time.After(300 * time.Millisecond):
	}
}

// liveWithoutStart is a provider that creates a conversation and then says
// nothing about it: everything arrives except the event that says the session
// is running.
type liveWithoutStart struct{ inner adapters.Live }

func (l liveWithoutStart) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	conn, err := l.inner.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	c := &connWithoutStart{LiveConn: conn, events: make(chan adapters.LiveEvent, 64)}
	go c.withhold()
	return c, nil
}

type connWithoutStart struct {
	adapters.LiveConn
	events chan adapters.LiveEvent
}

func (c *connWithoutStart) Events() <-chan adapters.LiveEvent { return c.events }

func (c *connWithoutStart) withhold() {
	defer close(c.events)
	for e := range c.LiveConn.Events() {
		if e.Kind == adapters.EventStarted {
			continue
		}
		c.events <- e
	}
}

// The graph is drawn from what the engine reports about itself, so pausing has
// to show there: the one block that costs money by the second goes dark when
// its conversation ends. Lighting it per utterance instead would have it look
// idle between sentences while it holds an open connection, and a node that
// stayed lit afterwards would be reporting a conversation that is gone.
func TestTheLiveBlockIsLitForAsLongAsItsConversation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := mockNPCConfig("refuse passage")
	cfg.IdleAfter = 150 * time.Millisecond
	n := mustNPCLive(t, cfg)

	// Subscribed before the graph starts, or the opening phases are missed.
	states := n.Graph.ObserveStates(ctx)
	invites := n.Live.Invitations()
	n.Graph.Start(ctx)

	waitFor(t, invites, "an invitation to connect")
	// Waiting for a player to arrive is not work: nothing is open and nothing
	// is being billed.
	waitForPhase(t, states, "live-session", ports.PhaseReady)

	connect(t, ctx, n)
	waitForPhase(t, states, "live-session", ports.PhaseWorking)

	// The mock says its piece and goes quiet, which is what a player wandering
	// off looks like from here.
	waitForPhase(t, states, "live-session", ports.PhaseReady)
}

func waitForPhase(t *testing.T, states <-chan ports.State, node string, want ports.Phase) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case state := <-states:
			if state.Node == node && state.Phase == want {
				return
			}
		case <-deadline:
			t.Fatalf("%s never reported %q", node, want)
		}
	}
}

func waitForStanding(t *testing.T, w *Wiring, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if got := w.Live.Standing(); got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("the conversation stood at %q rather than %q", w.Live.Standing(), want)
}

// Waits until the character has said something worth continuing from. What
// reaches the text sink has already been remembered, so this is the memory
// filling up, watched from the one place a test can see it.
func waitForSpeech(t *testing.T, w *Wiring) {
	t.Helper()
	select {
	case <-w.Sinks()["text"]:
	case <-time.After(5 * time.Second):
		t.Fatal("the character never said anything to continue from")
	}
}

func texts(said []adapters.Utterance) []string {
	out := make([]string, 0, len(said))
	for _, one := range said {
		out = append(out, one.Text)
	}
	return out
}

func opened(t *testing.T, r *recordingLive) adapters.LiveConfig {
	t.Helper()
	select {
	case cfg := <-r.opened:
		return cfg
	case <-time.After(5 * time.Second):
		t.Fatal("the conversation was never opened")
		return adapters.LiveConfig{}
	}
}

func waitFor(t *testing.T, signal <-chan ports.State, what string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("waited in vain for %s", what)
	}
}

// A genre says what a player may do with it, and the saying has to match the
// doing. A wiring that advertises typed input it cannot deliver would put a
// text box on the page that swallows what is typed.
func TestDeclaredInputsMatchTheWiring(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func() *Wiring
		want  []ports.InputMode
	}{
		{"adventure", func() *Wiring { return NewAdventure(nil) }, []ports.InputMode{ports.InputText}},
		{"npc-live", func() *Wiring { return mustNPCLive(t, mockNPCConfig("refuse passage")) }, []ports.InputMode{ports.InputAudioFullDuplex}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.build()
			if len(w.Inputs) != len(tc.want) {
				t.Fatalf("declares %v, want %v", w.Inputs, tc.want)
			}
			for i, mode := range tc.want {
				if w.Inputs[i] != mode {
					t.Errorf("declares %v, want %v", w.Inputs, tc.want)
				}
			}

			// Whatever it declares, it must actually accept.
			declared := map[ports.InputMode]bool{}
			for _, mode := range w.Inputs {
				declared[mode] = true
			}

			if declared[ports.InputText] && w.Say == nil {
				t.Error("declares typed input but has no way to take it")
			}
			spoken := declared[ports.InputAudioFullDuplex] || declared[ports.InputAudioPushToTalk]
			if spoken && w.Speak == nil {
				t.Error("declares spoken input but has no way to take it")
			}

			// And what it does not declare, it must not quietly offer.
			if w.Say != nil && !declared[ports.InputText] {
				t.Error("takes typed input it never declared")
			}
			if w.Speak != nil && !spoken {
				t.Error("takes spoken input it never declared")
			}
			if declared[ports.InputAudioFullDuplex] && declared[ports.InputAudioPushToTalk] {
				t.Error("declares both ways of speaking, which want different controls")
			}
		})
	}
}
