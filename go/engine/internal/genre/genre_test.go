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
		{"adventure", mustAdventure(t, nil).Graph},
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
	rephrase := blocks.NewToolCall("rephrase", mock.Tool{}, "rephrase")
	orphan := blocks.NewImage("orphan-image", mock.Image{})
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

// A block whose failure output is left unwired fails validation: what a
// failure leads to is the genre's to say, never left to chance.
func TestValidateCatchesAnUnwiredFailure(t *testing.T) {
	g := ports.NewGraph("unwired-failure")
	player := blocks.NewPlayerInputText("player")
	rephrase := blocks.NewToolCall("rephrase", mock.Tool{}, "rephrase")
	outText := blocks.NewPlayerOutputText("out-text", "text")
	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, outText)

	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "nothing consumes failure on rephrase") {
		t.Fatalf("expected the unwired failure output to be caught, got: %v", err)
	}

	g.ConnectFailure(rephrase, blocks.NewErrorSink("turn-failed", "turn_failed"))
	if err := g.Validate(); err != nil {
		t.Fatalf("wired, the graph is still refused: %v", err)
	}
}

// The world opens the game: before the player acts, the opening runs through
// Outline and Expand into narration, a picture and the initial status. Then
// the player's turn runs through Rephrase, Outline and Expand.
func TestAdventureFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := mustAdventure(t, map[string]string{"Health": "Good"})
	if err := a.Graph.Validate(); err != nil {
		t.Fatal(err)
	}
	narrate(a, true)
	text, props := utterances(streamOf(a, "text")), streamOf(a, "props")
	a.Graph.Start(ctx)

	// Seeded values reach the player first, so a status bar never starts
	// empty; then the opening's initial values.
	expect(t, props, `{"Health":"Good"}`)
	// The opening's prose: Expand's second seat on the thread, on its
	// opening prompt.
	opening := receiveLine(t, text, "the opening's prose")
	if !strings.HasPrefix(opening, "[story turn 2] NARRATE the opening scene") {
		t.Errorf("opening prose %q", opening)
	}
	expect(t, props, `{"Health":"[turn 1]"}`)
	if got := receiveLine(t, streamOf(a, "image"), "the opening's picture"); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("opening picture %q", got)
	}
	// The prose streams as it is spoken; the transcript is the prose.
	if chunk := receiveLine(t, streamOf(a, "audio"), "the opening's narration"); len(chunk) == 0 || len(chunk)%2 != 0 {
		t.Errorf("audio of %d bytes, want spoken PCM16", len(chunk))
	}

	a.Say("I try to cross the bridge")
	prose := receiveLine(t, text, "the turn's prose")
	if !strings.HasPrefix(prose, "[story turn 4] NARRATE the outline into prose") {
		t.Errorf("turn prose %q", prose)
	}
	expect(t, props, `{"Health":"[turn 3]"}`)
	// The turn's picture prompt carries the rephrased action through Outline.
	waitForImagePrompt(t, a, "The visual should show: [image_prompt 3] [mock-tool] I try to cross the bridge")
}

// Narration off, Adventure's turns are silent: the prose arrives, no audio.
func TestAdventureSilentWithoutNarration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := mustAdventure(t, nil)
	narrate(a, false)
	text := utterances(streamOf(a, "text"))
	a.Graph.Start(ctx)
	if got := receiveLine(t, text, "the opening's prose"); !strings.HasPrefix(got, "[story turn 2] NARRATE the opening scene") {
		t.Errorf("opening prose %q", got)
	}
	select {
	case chunk := <-streamOf(a, "audio"):
		t.Errorf("narration off, the player heard %d bytes", len(chunk))
	case <-time.After(20 * time.Millisecond):
	}
}

// narrate sets the audio-out toggle's blocks, as a session does.
func narrate(w *Wiring, on bool) {
	for _, block := range w.Switches[adapters.ToggleAudioOut] {
		block.SetActive(on)
	}
}

// receiveLine waits for one value on a stream.
func receiveLine(t *testing.T, ch <-chan string, what string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(3 * time.Second):
		t.Fatalf("never received %s", what)
		return ""
	}
}

// waitForImagePrompt waits until the picture prompt block has put out a
// prompt containing want, read back off the graph.
func waitForImagePrompt(t *testing.T, w *Wiring, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var seen []string
	for time.Now().Before(deadline) {
		seen = seen[:0]
		if detail, found := w.Graph.Inspect("image-prompt"); found {
			for _, out := range detail.Outputs {
				seen = append(seen, out.Value)
				if strings.Contains(out.Value, want) {
					return
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no picture prompt contains %q\nsaw:\n%s", want, strings.Join(seen, "\n---\n"))
}

// streamOf is the arrivals of the wiring's sink for a stream, nil when the
// genre wires none. A stream several sinks share, as `error` is, gives the
// first.
func streamOf(w *Wiring, stream string) chan string {
	for _, sink := range w.Sinks() {
		if sink.Stream() == stream {
			return sink.Arrivals()
		}
	}
	return nil
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
		case v := <-streamOf(n, "text"):
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
		case v := <-streamOf(n, "text"):
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

func mustAdventure(t testing.TB, status map[string]string) *Wiring {
	t.Helper()
	return adventure(t, AdventureConfig{Status: status})
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
func TestStructuredRequiresPropsEdge(t *testing.T) {
	g := ports.NewGraph("no-props")
	player := blocks.NewPlayerInputText("player")
	outline := blocks.NewStructured("outline", blocks.NewThread("story-thread", mock.Thread{}, "", ""), OutlineSchema(nil), outlineResult)
	outText := blocks.NewPlayerOutputText("out-text", "text")

	g.ConnectTextOut(player, outline)
	g.ConnectTextOut(outline, outText)

	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "outline has no incoming props edge") {
		t.Fatalf("expected a missing props edge to be caught, got: %v", err)
	}
}

// A block whose props input is required must not act before the current
// values have reached it, and a player's first turn carries the values the
// opening set, not the defaults. Without that guarantee it decides from
// defaults and overwrites the real values on the way out — which shows up as
// a status bar that resets on the first turn, intermittently.
func TestOpeningStatusSurvivesTheFirstTurn(t *testing.T) {
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		a := mustAdventure(t, map[string]string{"Health": "Wounded", "Gold": "12"})
		props := streamOf(a, "props")
		a.Graph.Start(ctx)
		a.Say("I press on")

		deadline := time.After(3 * time.Second)
		var sawTurn bool
		for !sawTurn {
			select {
			case v := <-props:
				if !strings.Contains(v, "[turn 3]") {
					continue
				}
				sawTurn = true
				if !strings.Contains(v, `"Health":"[turn 3]"`) || strings.Contains(v, "Wounded") {
					cancel()
					t.Fatalf("run %d: the first turn did not go on from the opening's values: %s", i, v)
				}
			case <-deadline:
				cancel()
				t.Fatalf("run %d: no props for the first turn", i)
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
	case <-streamOf(n, "image"):
	case <-time.After(3 * time.Second):
		t.Fatal("no portrait, and nobody spoke to trigger one")
	}

	// Several turns of play, so a block generating per input would have been
	// caught by now.
	for turns := 0; turns < 3; turns++ {
		select {
		case <-streamOf(n, "text"):
		case <-time.After(3 * time.Second):
			t.Fatalf("the conversation stopped after %d turns", turns)
		}
	}

	if got := counter.calls(); got != 1 {
		t.Errorf("the image adapter was called %d times, want 1", got)
	}
	select {
	case v := <-streamOf(n, "image"):
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
	if streamOf(n, "flag") != nil {
		t.Error("the genre still advertises a stream for what the observer found")
	}
	n.Graph.Start(ctx)
	connect(t, ctx, n)

	// Play until the observer has certainly judged and steered, then check that
	// nothing it found reached the one place a player reads.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case v := <-streamOf(n, "text"):
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
		{"adventure", func() *Wiring { return mustAdventure(t, nil) }},
		{"npc-live", func() *Wiring { return mustNPCLive(t, mockNPCConfig("refuse passage")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.build()
			if len(w.Sinks()) == 0 {
				t.Fatal("a genre with no streams cannot show anybody anything")
			}
			known := map[string]bool{}
			for _, name := range []string{"text", "audio", "image", "props", "error", "problem"} {
				known[name] = true
			}
			for _, sink := range w.Sinks() {
				if sink.Arrivals() == nil {
					t.Errorf("stream %q is advertised with nothing behind it", sink.Stream())
				}
				if !known[sink.Stream()] {
					t.Errorf("sink %s puts out %q, a stream the session does not carry", sink.NodeName(), sink.Stream())
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
	case line := <-streamOf(n, "text"):
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
	case line := <-streamOf(n, "text"):
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
	case <-streamOf(w, "text"):
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
		{"adventure", func() *Wiring { return mustAdventure(t, nil) }, []ports.InputMode{ports.InputText}},
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
