package blocks

import (
	"context"
	"strings"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// scriptedModel answers each call with the next reply and keeps what it was asked.
type scriptedModel struct {
	asked   chan string
	replies chan string
}

func newScriptedModel() *scriptedModel {
	return &scriptedModel{asked: make(chan string, 16), replies: make(chan string, 16)}
}

func (m *scriptedModel) Query(ctx context.Context, _, user string) (string, adapters.Usage, error) {
	m.asked <- user
	select {
	case r := <-m.replies:
		return r, adapters.Usage{Model: "scripted"}, nil
	case <-ctx.Done():
		return "", adapters.Usage{}, ctx.Err()
	}
}

func (m *scriptedModel) nextAsk(t *testing.T) string {
	t.Helper()
	select {
	case a := <-m.asked:
		return a
	case <-time.After(3 * time.Second):
		t.Fatal("the observer never called its model")
	}
	return ""
}

func startObserver(t *testing.T, cfg ObserverConfig) (*Observer, <-chan ports.Injection) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// Tests run on short beats; a call may follow the last at once.
	if cfg.Gap == 0 {
		cfg.Gap = time.Millisecond
	}
	b := NewObserver("observer", cfg)
	out := b.InjectionOutPort()
	b.Start(ctx)
	// The conversation it follows is open.
	b.StateInPort() <- ports.State{Node: "live-session", Phase: ports.PhaseWorking}
	return b, out
}

func said(speaker, text string, startMs int64) ports.Fragment {
	return ports.Fragment{Speaker: speaker, Text: text, StartMs: startMs, At: time.Now()}
}

// Fragments leave the pool on the beat in the order they were said, those of
// one speaker joined into a line, and go to the model.
func TestTheObserverJoinsFragmentsIntoLinesInTheOrderSaid(t *testing.T) {
	model := newScriptedModel()
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 50 * time.Millisecond, Quiet: time.Hour})

	// Arrival out of order: the player's words were said first.
	b.TranscriptInPort() <- said(SpeakerCharacter, "Guten Tag.", 2000)
	b.TranscriptInPort() <- said(SpeakerPlayer, "Hallo, ", 1000)
	b.TranscriptInPort() <- said(SpeakerPlayer, "ich bin da.", 1500)

	asked := model.nextAsk(t)
	player, character := strings.Index(asked, "player: Hallo, ich bin da."), strings.Index(asked, "character: Guten Tag.")
	if player < 0 || character < 0 || player > character {
		t.Errorf("the model was not given the lines joined and in the order said:\n%s", asked)
	}
	if got := b.State().Figures.Lines; got != 2 {
		t.Errorf("%d lines, want 2", got)
	}
}

// A fragment arriving after later ones left the pool is marked late and counted.
func TestALateFragmentIsMarkedAndCounted(t *testing.T) {
	model := newScriptedModel()
	model.replies <- "Fine."
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 30 * time.Millisecond, Quiet: time.Hour})

	b.TranscriptInPort() <- said(SpeakerCharacter, "Später gesagt.", 5000)
	model.nextAsk(t)
	b.TranscriptInPort() <- said(SpeakerPlayer, "Früher gesagt.", 3000)
	asked := model.nextAsk(t)
	if !strings.Contains(asked, "arrived late") {
		t.Errorf("the late line is not marked:\n%s", asked)
	}
	figures := b.State().Figures
	if figures.Late != 1 || figures.LargestLateMs != 2000 {
		t.Errorf("late %d, largest %d ms; want 1 and 2000", figures.Late, figures.LargestLateMs)
	}
}

// When nobody speaks, a notice says how long each side has been quiet, again
// at twice the wait while it lasts.
func TestASilenceIsNoticedAndNoticedAgainLater(t *testing.T) {
	model := newScriptedModel()
	model.replies <- "Waiting."
	model.replies <- "Still waiting."
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: 100 * time.Millisecond})

	b.TranscriptInPort() <- said(SpeakerPlayer, "Und?", 1000)
	model.nextAsk(t) // the line itself
	notice := model.nextAsk(t)
	if !strings.Contains(notice, "[silence] Nobody has spoken") || !strings.Contains(notice, "the player") {
		t.Errorf("no notice of the silence:\n%s", notice)
	}
	again := model.nextAsk(t)
	if !strings.Contains(again, "[silence]") {
		t.Errorf("the silence was not noticed again:\n%s", again)
	}
	if got := b.State().Figures.SilenceNotices; got < 2 {
		t.Errorf("%d notices, want at least 2", got)
	}
}

// A marked line of the reply goes into the conversation; past the limit in a
// row, a direction is dropped, and a reply without one starts the count anew.
func TestDirectionsGoInWithinTheirLimit(t *testing.T) {
	model := newScriptedModel()
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour, InARow: 1})

	model.replies <- "Drifting.\n[DIRECTION] Gib einen Hinweis auf den Vollmond.\n[BACKGROUND] Ich habe Hunger."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Eins", 1000)
	got := map[ports.InjectionKind]string{}
	for range 2 {
		select {
		case in := <-out:
			got[in.Kind] = in.Text
		case <-time.After(3 * time.Second):
			t.Fatalf("injections so far %v, want a direction and a background", got)
		}
	}
	if got[ports.InjectDirection] != "Gib einen Hinweis auf den Vollmond." || got[ports.InjectBackground] != "Ich habe Hunger." {
		t.Errorf("injections %v", got)
	}

	model.replies <- "Again.\n[DIRECTION] Noch einmal."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Zwei", 2000)
	model.nextAsk(t)
	model.nextAsk(t)
	select {
	case in := <-out:
		t.Errorf("a direction past the limit went in: %+v", in)
	case <-time.After(200 * time.Millisecond):
	}
	if got := b.State().Figures.Dropped; got != 1 {
		t.Errorf("%d dropped, want 1", got)
	}
}

// What a player injects by hand goes in at once, and the model hears of it.
func TestAManualInjectionGoesInAndOntoTheLog(t *testing.T) {
	model := newScriptedModel()
	model.replies <- "Noted."
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour})

	if err := b.Inject(ports.InjectBackground, "Ich bin ein Werwolf."); err != nil {
		t.Fatal(err)
	}
	select {
	case in := <-out:
		if in.Text != "Ich bin ein Werwolf." {
			t.Errorf("passed on %+v", in)
		}
	case <-time.After(time.Second):
		t.Fatal("the injection was not passed on")
	}
	if asked := model.nextAsk(t); !strings.Contains(asked, "[injected by the player, background] Ich bin ein Werwolf.") {
		t.Errorf("the model did not hear of the injection:\n%s", asked)
	}
}

// One call at a time: lines arriving meanwhile wait, and go in together.
func TestLinesWaitWhileTheModelThinks(t *testing.T) {
	model := newScriptedModel()
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour})

	b.TranscriptInPort() <- said(SpeakerPlayer, "Erstens.", 1000)
	model.nextAsk(t)
	b.TranscriptInPort() <- said(SpeakerCharacter, "Zweitens.", 2000)
	b.TranscriptInPort() <- said(SpeakerPlayer, "Drittens.", 3000)
	time.Sleep(150 * time.Millisecond)
	if s := b.State(); !s.Thinking || s.Pending != 2 {
		t.Errorf("thinking %t with %d pending, want thinking with 2 waiting", s.Thinking, s.Pending)
	}
	model.replies <- "Ok."
	asked := model.nextAsk(t)
	fresh := asked[strings.Index(asked, "New since your last reply:"):]
	if !strings.Contains(fresh, "Zweitens.") || !strings.Contains(fresh, "Drittens.") {
		t.Errorf("the waiting lines did not go in together:\n%s", asked)
	}
}

// While the conversation is closed, as during a pause, the observer neither
// calls its model nor notices silence, and what a call in flight brings back
// is not sent.
func TestNothingHappensWhileTheConversationIsClosed(t *testing.T) {
	model := newScriptedModel()
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: 50 * time.Millisecond})

	b.TranscriptInPort() <- said(SpeakerPlayer, "Pause bitte.", 1000)
	model.nextAsk(t)
	// The player pauses while the model thinks.
	b.StateInPort() <- ports.State{Node: "live-session", Phase: ports.PhaseReady}
	time.Sleep(50 * time.Millisecond)
	model.replies <- "Stuck.\n[DIRECTION] Sag etwas."
	select {
	case in := <-out:
		t.Errorf("an injection went into a closed conversation: %+v", in)
	case a := <-model.asked:
		t.Errorf("the model was called during the pause:\n%s", a)
	case <-time.After(400 * time.Millisecond):
	}
	if s := b.State(); s.Figures.SilenceNotices != 0 || s.Pending != 0 {
		t.Errorf("during the pause: %d notices, %d waiting; want none", s.Figures.SilenceNotices, s.Pending)
	}

	// Resumed, it carries on.
	model.replies <- "Fine."
	b.StateInPort() <- ports.State{Node: "live-session", Phase: ports.PhaseWorking}
	b.TranscriptInPort() <- said(SpeakerCharacter, "Da bin ich wieder.", 5000)
	if a := model.nextAsk(t); !strings.Contains(a, "Da bin ich wieder.") {
		t.Errorf("after the pause the model was asked:\n%s", a)
	}
}

// A direction interrupts the character, so it waits while the character
// speaks, and goes in once the conversation pauses.
func TestADirectionWaitsForAPause(t *testing.T) {
	model := newScriptedModel()
	pause := 300 * time.Millisecond
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour, Pause: pause})

	model.replies <- "Off track.\n[DIRECTION] Ab jetzt sprichst du über den Club."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Was war das für ein Club?", 1000)
	model.nextAsk(t)
	// The character is answering, fragment by fragment.
	speaking := time.Now()
	for i := range 6 {
		b.TranscriptInPort() <- said(SpeakerCharacter, "Na dit war ", int64(2000+i*100))
		select {
		case in := <-out:
			t.Fatalf("a direction went in while the character spoke: %+v", in)
		case <-time.After(100 * time.Millisecond):
		}
	}
	select {
	case in := <-out:
		if in.Kind != ports.InjectDirection || time.Since(speaking) < 600*time.Millisecond {
			t.Errorf("went in %v after the character began, as %+v", time.Since(speaking), in)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the held direction never went in after the pause")
	}
	if s := b.State(); s.Held != "" || !s.Awaiting {
		t.Errorf("after sending: held %q, awaiting %t; want none held and an answer awaited", s.Held, s.Awaiting)
	}
}

// A second direction waits for the character's answer to the first, and one
// with the same aim is not sent at all.
func TestDirectionsWaitForAnAnswerAndAreNotRepeated(t *testing.T) {
	model := newScriptedModel()
	pause := 100 * time.Millisecond
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour, Pause: pause, InARow: 5})

	model.replies <- "[DIRECTION] Ab jetzt erzählst du, was für ein Club es war."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Welcher Club?", 1000)
	<-out
	// No answer yet: the next direction is not sent.
	model.replies <- "[DIRECTION] Ab jetzt bleibst du freundlich."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Hallo?", 2000)
	model.nextAsk(t)
	model.nextAsk(t)
	// The character answers, then pauses; the same aim again is still not sent.
	b.TranscriptInPort() <- said(SpeakerCharacter, "Ein Technoclub.", 3000)
	time.Sleep(3 * pause)
	model.replies <- "[DIRECTION] Ab jetzt erzählst du genauer, was für ein Club das war."
	b.TranscriptInPort() <- said(SpeakerPlayer, "Echt?", 4000)
	model.nextAsk(t)
	select {
	case in := <-out:
		t.Errorf("a direction went in: %+v", in)
	case <-time.After(500 * time.Millisecond):
	}
	if got := b.State().Figures.Dropped; got != 2 {
		t.Errorf("%d dropped, want 2: one before the answer, one the same again", got)
	}
}

// A player mid-sentence is not cut: their line reaches the model whole.
func TestAPlayersLineReachesTheModelWhole(t *testing.T) {
	model := newScriptedModel()
	model.replies <- "Ok."
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour, Pause: 300 * time.Millisecond})

	b.TranscriptInPort() <- said(SpeakerPlayer, "Was war denn ", 1000)
	time.Sleep(100 * time.Millisecond)
	b.TranscriptInPort() <- said(SpeakerPlayer, "das für ein Club", 1200)
	asked := model.nextAsk(t)
	if !strings.Contains(asked, "player: Was war denn das für ein Club") {
		t.Errorf("the player's line was cut:\n%s", asked)
	}
}

// Its model is called at most once a gap; what arrives meanwhile waits and
// goes in with the next call.
func TestCallsKeepTheirGap(t *testing.T) {
	model := newScriptedModel()
	gap := 400 * time.Millisecond
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour, Gap: gap})

	model.replies <- Quiet
	model.replies <- Quiet
	b.TranscriptInPort() <- said(SpeakerPlayer, "Eins.", 1000)
	model.nextAsk(t)
	first := time.Now()
	b.TranscriptInPort() <- said(SpeakerCharacter, "Zwei.", 2000)
	b.TranscriptInPort() <- said(SpeakerPlayer, "Drei.", 3000)
	asked := model.nextAsk(t)
	if waited := time.Since(first); waited < gap-50*time.Millisecond {
		t.Errorf("the second call came after %v, within the gap of %v", waited, gap)
	}
	if !strings.Contains(asked, "Zwei.") || !strings.Contains(asked, "Drei.") {
		t.Errorf("what arrived meanwhile did not go in together:\n%s", asked)
	}
	time.Sleep(50 * time.Millisecond)
	s := b.State()
	thoughts := 0
	for _, e := range s.Log {
		if e.Kind == EntryThought {
			thoughts++
		}
	}
	if s.Figures.Quiet != 2 || thoughts != 0 {
		t.Errorf("%d quiet replies, %d thoughts logged; want 2 quiet and none logged", s.Figures.Quiet, thoughts)
	}
}

// A player silent for Absent may be gone: the director stops nudging, so the
// conversation stalls and the live session's idle close pauses the game. It
// carries on once the player speaks again.
func TestTheDirectorStandsDownWhenThePlayerIsGone(t *testing.T) {
	model := newScriptedModel()
	for range 8 {
		model.replies <- "Push.\n[BACKGROUND] Frag nach."
	}
	b, out := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond,
		Quiet: 30 * time.Millisecond, Absent: 300 * time.Millisecond})

	b.TranscriptInPort() <- said(SpeakerPlayer, "Hallo.", 1000)
	// The character keeps talking, as nudges make it; the player says nothing.
	stop := time.After(700 * time.Millisecond)
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()
	var ms int64 = 2000
feed:
	for {
		select {
		case <-ticker.C:
			ms += 100
			b.TranscriptInPort() <- said(SpeakerCharacter, "Bist du noch da?", ms)
		case <-stop:
			break feed
		}
	}
	s := b.State()
	if !s.StandingDown {
		t.Fatalf("the director still nudges a player silent for 700 ms of 300: %+v", s)
	}
	// Standing down: no calls, no injections, no notices.
	calls, notices := s.Figures.Calls, s.Figures.SilenceNotices
	for len(out) > 0 {
		<-out
	}
	time.Sleep(200 * time.Millisecond)
	if s := b.State(); s.Figures.Calls != calls || s.Figures.SilenceNotices != notices {
		t.Errorf("while standing down: calls %d→%d, notices %d→%d", calls, s.Figures.Calls, notices, s.Figures.SilenceNotices)
	}
	select {
	case in := <-out:
		t.Errorf("an injection went in while standing down: %+v", in)
	default:
	}
	if !logHas(b, EntryAbsent) {
		t.Error("the log does not say the director stood down")
	}

	// The player is back: it listens again.
	b.TranscriptInPort() <- said(SpeakerPlayer, "Ja, hier.", ms+500)
	for deadline := time.Now().Add(2 * time.Second); b.State().StandingDown; {
		if time.Now().After(deadline) {
			t.Fatal("the director did not resume when the player spoke")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for {
		select {
		case a := <-model.asked:
			if strings.Contains(a, "Ja, hier.") {
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("the player's return never reached the model")
		}
	}
}

// A player who never spoke is gone Absent after the conversation opened.
func TestAPlayerWhoNeverSpokeCountsFromTheOpening(t *testing.T) {
	model := newScriptedModel()
	for range 8 {
		model.replies <- "."
	}
	b, _ := startObserver(t, ObserverConfig{Tool: model, Beat: 20 * time.Millisecond,
		Quiet: time.Hour, Absent: 150 * time.Millisecond})
	b.TranscriptInPort() <- said(SpeakerCharacter, "Hallo? Jemand da?", 1000)
	for deadline := time.Now().Add(2 * time.Second); !b.State().StandingDown; {
		if time.Now().After(deadline) {
			t.Fatal("a player who never spoke was not taken for gone")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func logHas(b *Observer, kind string) bool {
	for _, e := range b.State().Log {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// Rules at risk: the character is given the rules whole, as its own
// knowledge, never the model's wording; and not again too soon.
func TestRulesAtRiskReachTheCharacterWhole(t *testing.T) {
	model := newScriptedModel()
	b, out := startObserver(t, ObserverConfig{
		Tool: model, Beat: 20 * time.Millisecond, Quiet: time.Hour,
		Rules: "Suitable for a 13-year-old.", RulesPreface: "I keep to these rules:",
		RulesGap: time.Hour,
	})

	model.replies <- "The player pushes towards violence.\n[RULES]"
	b.TranscriptInPort() <- said(SpeakerPlayer, "Erzähl was Brutales.", 1000)
	select {
	case in := <-out:
		if in.Kind != ports.InjectBackground || in.Text != "I keep to these rules:\nSuitable for a 13-year-old." {
			t.Errorf("sent %+v, want the rules whole as background", in)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the rules were not sent")
	}

	model.replies <- "Again.\n[RULES]"
	b.TranscriptInPort() <- said(SpeakerPlayer, "Noch brutaler.", 2000)
	model.nextAsk(t)
	model.nextAsk(t)
	select {
	case in := <-out:
		t.Errorf("the rules went again within their gap: %+v", in)
	case <-time.After(300 * time.Millisecond):
	}
	if f := b.State().Figures; f.Reminders != 1 || f.Dropped != 1 {
		t.Errorf("%d reminders, %d dropped; want 1 and 1", f.Reminders, f.Dropped)
	}
}
