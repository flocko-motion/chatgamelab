package engine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"engine"
	"engine/internal/adapters/mock"
)

// The public surface has to be enough on its own: launch, drive, read events.
// If a test needs internal/, the boundary is in the wrong place.
func TestAdventureThroughPublicAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &engine.MemoryPersist{}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreAdventureRounds,
		ID:       "test-adventure",
		Persist:  store,
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Say("I try to cross the bridge"); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"text": false, "audio": false, "image": false, "props": false}
	deadline := time.After(10 * time.Second)
	for remaining(want) > 0 {
		select {
		case e := <-s.Events():
			want[e.Stream] = true
		case <-deadline:
			t.Fatalf("never saw every stream, still missing %d: %v", remaining(want), want)
		}
	}

	// What a reload redraws is stored; audio is not. Which phase changes come
	// with it depends on timing, so the check is for the streams, not a count.
	stored := map[string]bool{"text": false, "image": false, "props": false}
	for remaining(stored) > 0 {
		for _, e := range store.Events("test-adventure") {
			if _, wanted := stored[e.Stream]; wanted {
				stored[e.Stream] = true
			}
			if e.Stream == "audio" {
				t.Fatal("audio was stored")
			}
		}
		if remaining(stored) == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("never stored every stream, still missing %d: %v", remaining(stored), stored)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// Adventure plays without pictures, as it does on a platform that offers none,
// such as Mistral: the turn still tells its story and updates its status, and
// no picture arrives.
func TestAdventureWithoutImages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	offer := engine.MockOffer()
	offer.Caveats = engine.Caveats{
		Allow:   map[engine.Tier]engine.Toggles{engine.TierLow: {}},
		Default: engine.Choice{Tier: engine.TierLow},
	}
	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreAdventureRounds,
		ID:       "adventure-without-images",
	}, []engine.Offer{offer})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Say("I try to cross the bridge"); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"text": false, "props": false}
	deadline := time.After(10 * time.Second)
	for remaining(want) > 0 {
		select {
		case e := <-s.Events():
			if e.Stream == "image" {
				t.Fatal("a picture arrived with images off")
			}
			want[e.Stream] = true
		case <-deadline:
			t.Fatalf("the turn never completed, still missing %d: %v", remaining(want), want)
		}
	}

	// The mock draws at once, so a picture would have come with the turn.
	settle := time.After(200 * time.Millisecond)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "image" {
				t.Fatal("a picture arrived with images off")
			}
		case <-settle:
			return
		}
	}
}

// A live genre is spoken, and says so. GPT-Live has no event that delivers user
// text into a conversation, so a text box on this genre would be a control that
// cannot work — and silently swallowing what was typed would look to a player
// like a character ignoring them.
func TestLiveGenreIsSpokenOnly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre:  engine.SubgenreNPCLive,
		Guardrail: "refuse passage",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Speak([]byte("persuasion")); err != nil {
		t.Errorf("Speak should be accepted by npc-live: %v", err)
	}
	if err := s.Say("let me through"); err == nil {
		t.Error("npc-live accepted typed input it has no way to deliver")
	}
}

// A player is invited to open their own connection once the game has begun, and
// not before: the portrait should be on screen before anybody is asked to
// speak. Until then the conversation does not exist, and is not being billed.
func TestConnectionIsInvitedOnlyOnceTheGameBegins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		Scenario: "You are the keeper of a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream != "connect" {
				continue
			}
			// The invitation only follows the gate, so the game has started by
			// the time one arrives.
			select {
			case <-s.Ready():
			default:
				t.Error("a player was invited to connect before the gate opened")
			}
			answer, err := s.Connect(ctx, "v=0 mock offer", "")
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			if answer == "" {
				t.Error("the browser was given no answer to apply")
			}
			return
		case <-deadline:
			t.Fatal("nobody was ever invited to connect")
		}
	}
}

// A genre holding no conversation has nothing to broker, and says so rather
// than leaving a caller waiting for an answer that cannot come.
func TestConnectIsRefusedByATurnBasedGenre(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{Subgenre: engine.SubgenreAdventureRounds}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.Connect(ctx, "v=0 mock offer", ""); err == nil {
		t.Error("adventure brokered a connection to a conversation it does not hold")
	}
}

// A genre that genuinely takes no input of a kind must say so rather than
// swallow it: input vanishing into a genre would surface as a mute character
// rather than as an error. Adventure takes both: typed lines, and speech, one
// clip per utterance, while voice input is on.
func TestUnsupportedInputKindIsRejected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{Subgenre: engine.SubgenreAdventureRounds}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Speak(spokenClip()); err != nil {
		t.Errorf("Speak should be accepted by adventure: %v", err)
	}
	if err := s.Say("I try to cross the bridge"); err != nil {
		t.Errorf("Say should be accepted by adventure: %v", err)
	}
	if err := s.SetToggle(engine.ToggleAudioIn, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Speak(spokenClip()); err == nil {
		t.Error("Speak was accepted with voice input off")
	}
}

func TestUnknownGenreRefused(t *testing.T) {
	_, err := engine.Launch(context.Background(), engine.SessionSpec{Subgenre: "quiz"}, mockOffers())
	if err == nil {
		t.Fatal("expected an error for an unknown genre")
	}
}

func remaining(m map[string]bool) int {
	n := 0
	for _, seen := range m {
		if !seen {
			n++
		}
	}
	return n
}

// A real platform without an injected key resolver must refuse to launch,
// rather than discovering the problem on the first model call.
func TestRealPlatformNeedsKeyFunc(t *testing.T) {
	_, err := engine.Launch(context.Background(), engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
	}, []engine.Offer{openaiOffer(nil)})
	if err == nil {
		t.Fatal("expected a launch error when no Keys function is injected")
	}
}

// A serialised spec is what gets stored, so it holds the player's preference
// and never the offers: those are live, and carry the keys.
func TestSpecMarshalsWithoutOffers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	offer := openaiOffer(func(context.Context) (string, error) { return "sk-should-not-appear", nil })
	s, err := engine.Launch(ctx, engine.SessionSpec{Subgenre: engine.SubgenreNPCLive, ID: "stored"}, []engine.Offer{offer})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	spec := engine.SessionSpec{Subgenre: engine.SubgenreNPCLive, ID: "stored", Preference: s.Preference()}
	blob, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{`"offers"`, "caveats", "sk-should-not-appear", "Keys"} {
		if strings.Contains(string(blob), banned) {
			t.Errorf("serialised spec holds %s: %s", banned, blob)
		}
	}
	if !strings.Contains(string(blob), `"preference"`) || !strings.Contains(string(blob), `"offer":"openai-low"`) {
		t.Errorf("serialised spec lost the preference: %s", blob)
	}
}

// The portrait is init work that nothing waits on, so a failure there costs the
// picture and not the conversation. What gates is a wiring decision, and this is
// the behaviour that decision buys.
func TestPortraitFailureDoesNotStopTheSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		ID:       "no-portrait",
		Scenario: "a bridge keeper",
	}, []engine.Offer{
		// A key resolver that fails stands in for any init work that cannot
		// complete: the image call never gets a key.
		openaiOffer(func(context.Context) (string, error) {
			return "", errors.New("no key for you")
		}),
	})
	if err != nil {
		t.Fatalf("a failed portrait must not stop the session: %v", err)
	}
	defer s.Close()

	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the gate never opened, though nothing was wired to hold it")
	}
}

// The portrait arrives without anything triggering it: the game's picture
// from the gallery, shown before play begins.
func TestPortraitArrivesWithoutBeingTriggered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre:     engine.SubgenreNPCLive,
		ID:           "portrait",
		Scenario:     "You are the keeper of a bridge.",
		GalleryImage: mock.GalleryImage("bridge"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "image" {
				// A picture, not a placeholder: the client decodes this through
				// the same path a generated image will use.
				if !strings.HasPrefix(e.Value, "data:image/png;base64,") {
					t.Errorf("portrait was not a displayable image: %.40s", e.Value)
				}
				return
			}
		case <-deadline:
			t.Fatal("no portrait, and nobody spoke to trigger one")
		}
	}
}

// Every block reports what it is doing on the session stream. This is the
// debugging surface: which block is busy, and for how long, is how a turn is
// read — and the same data a graph view draws.
func TestBlocksReportTheirPhases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		ID:       "phases",
		Scenario: "You are the keeper of a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := awaitConnection(ctx, s); err != nil {
		t.Fatal(err)
	}

	// The observer works when it passes an injection on.
	if err := s.Inject("observer", "direction", "Keep the bridge closed."); err != nil {
		t.Fatal(err)
	}

	// Both, and in no particular order: these are separate blocks whose reports
	// are merged by introspection through a goroutine each, so nothing orders
	// one against the other. Waiting for one and then asserting the other had
	// already arrived is a race, not a fact.
	working := map[string]bool{}
	deadline := time.After(10 * time.Second)
	for !working["observer"] || !working["live-session"] {
		select {
		case e := <-s.Events():
			if e.Stream != "state" {
				continue
			}
			var report engine.BlockState
			if err := json.Unmarshal([]byte(e.Value), &report); err != nil {
				t.Fatalf("state event was not a block report: %q", e.Value)
			}
			if report.Phase == "working" {
				working[report.Node] = true
			}
		case <-deadline:
			t.Fatalf("not every block reported working; saw %v", working)
		}
	}
}

// The gate holds the player's inputs until preparation finishes, and reports
// that it has opened — which is what a client needs before offering a control
// that will actually work.
func TestGateOpeningIsVisibleOnTheStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		ID:       "gated",
		Scenario: "You are the keeper of a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream != "state" {
				continue
			}
			var report engine.BlockState
			if err := json.Unmarshal([]byte(e.Value), &report); err != nil {
				t.Fatal(err)
			}
			if report.Node == "start-game" && report.Phase == "ready" {
				return
			}
		case <-deadline:
			t.Fatal("the gate never reported opening, so a client cannot know when to let anyone play")
		}
	}
}

// The cue outlives the wait for a player. It is issued when the gate opens, but
// there is no character to give it to until somebody connects — and a cue
// dropped in that window would leave the player facing a portrait that never
// speaks.
func TestOpeningCueSurvivesTheWaitForAPlayer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		ID:       "early",
		Scenario: "You are the keeper of a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := awaitConnection(ctx, s); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "text" && strings.Contains(e.Value, "shall not pass") {
				return
			}
		case <-deadline:
			t.Fatal("the character never spoke, so the opening cue was lost")
		}
	}
}

// awaitConnection plays the browser's part: wait to be invited, then connect.
func awaitConnection(ctx context.Context, s *engine.Session) error {
	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream != "connect" {
				continue
			}
			_, err := s.Connect(ctx, "v=0 mock offer", "")
			return err
		case <-deadline:
			return errors.New("nobody was ever invited to connect")
		}
	}
}

// Mistral has no live voice model, so a live genre must refuse it at launch
// rather than fail once a player connects.
func TestNPCLiveRefusesMistral(t *testing.T) {
	_, err := engine.Launch(context.Background(), engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
	}, []engine.Offer{{
		Platform: engine.PlatformMistral,
		ID:       "mistral",
		Keys:     func(context.Context) (string, error) { return "k", nil },
		Caveats: engine.Caveats{
			Allow:   map[engine.Tier]engine.Toggles{engine.TierLow: {}},
			Default: engine.Choice{Tier: engine.TierLow},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "audio-live") {
		t.Fatalf("err = %v, want a refusal naming the missing audio-live toggle", err)
	}
}

// openaiOffer is a single OpenAI offer at low, with what NPC-Live needs.
func openaiOffer(keys engine.KeyFunc) engine.Offer {
	needs := engine.Toggles{Images: true, AudioLive: true}
	return engine.Offer{
		Platform: engine.PlatformOpenAI,
		ID:       "openai-low",
		Keys:     keys,
		Caveats: engine.Caveats{
			Allow:   map[engine.Tier]engine.Toggles{engine.TierLow: needs},
			Default: engine.Choice{Tier: engine.TierLow, Toggles: needs},
		},
	}
}

// spokenClip is a second of loud PCM16 at 24 kHz mono, as the playback-engine
// records it.
func spokenClip() []byte { return bytes.Repeat([]byte{0x00, 0x10}, 24_000) }

// mockOffers is what a test runs on: the mock, with nothing to choose.
func mockOffers() []engine.Offer { return []engine.Offer{engine.MockOffer()} }

// The journal records a live conversation's life as the engine saw it, kept
// for a page that reloads and streamed to the one that is open.
func TestJournalRecordsTheLiveConversation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive,
		Scenario: "You are the keeper of a bridge.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	streamed := 0
	deadline := time.After(10 * time.Second)
	connected := false
	for !connected || !journalHas(s.Journal(), "live-session", "started") {
		select {
		case e := <-s.Events():
			switch e.Stream {
			case "journal":
				streamed++
			case "connect":
				if _, err := s.Connect(ctx, "v=0 mock offer", ""); err != nil {
					t.Fatal(err)
				}
				connected = true
			}
		case <-deadline:
			t.Fatalf("the journal never showed the conversation starting: %+v", s.Journal())
		}
	}

	for _, want := range [][2]string{
		{engine.JournalSession, "launched"},
		{engine.JournalSession, "settled"},
		{"live-session", "invited"},
		{engine.JournalClient, "connect"},
		{"live-session", "opened"},
	} {
		if !journalHas(s.Journal(), want[0], want[1]) {
			t.Errorf("the journal lacks %s %s: %+v", want[0], want[1], s.Journal())
		}
	}
	if streamed == 0 {
		t.Error("no journal entry reached the event stream")
	}
}

func journalHas(entries []engine.JournalEntry, source, kind string) bool {
	for _, e := range entries {
		if e.Source == source && e.Kind == kind {
			return true
		}
	}
	return false
}

// A toggle is offered only where it switches a block of the wiring: Adventure
// makes pictures and speaks, NPC-Live shows the gallery's picture and has
// nothing to switch.
func TestTheTopologyNamesTheTogglesItSwitches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for genre, want := range map[engine.Subgenre][]string{
		engine.SubgenreAdventureRounds: {"audio-in", "audio-out", "images"},
		engine.SubgenreNPCLive:         {},
	} {
		s, err := engine.Launch(ctx, engine.SessionSpec{Subgenre: genre, Scenario: "A bridge."}, mockOffers())
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Topology().Switches; !slices.Equal(got, want) {
			t.Errorf("%s switches %v, want %v", genre, got, want)
		}
		s.Close()
	}
}

// A conversation's block shows it whole in its details, both speakers.
func TestTheLiveBlockShowsItsTranscript(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := engine.Launch(ctx, engine.SessionSpec{Subgenre: engine.SubgenreNPCLive, Scenario: "You keep a bridge."}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := awaitConnection(ctx, s); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); ; {
		detail, _ := s.Inspect("live-session")
		if len(detail.Transcript) > 0 && detail.Transcript[0].Speaker == "character" && detail.Transcript[0].Text != "" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the live block's details show no transcript: %+v", detail.Transcript)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
