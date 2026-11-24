package blocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// LiveSession holds the control connection to one full-duplex conversation; its
// audio ports draw the browser-to-provider path even where no bytes cross here.
type LiveSession struct {
	name string
	live adapters.Live
	cfg  adapters.LiveConfig

	mic   ports.AudioInput
	steer ports.TextInput
	// injected is what an observer puts into the conversation from the side.
	injected ports.InjectionInput
	// opening takes words for the character to say once the first
	// conversation starts.
	opening ports.TextInput
	// voiceIn is the voice made for the game, by id; voice the latest, which
	// the next conversation speaks in instead of the configured one.
	voiceIn ports.TextInput
	voice   string
	// voiced closes once a voice has come; a conversation wired to be given
	// one waits for it a while (VoiceWait) rather than open in another.
	voiced     chan struct{}
	voicedOnce sync.Once
	voiceWired bool
	// VoiceWait is how long a conversation wired to be given a voice waits
	// for it before it opens in the configured one. Set before Start.
	VoiceWait time.Duration
	// styleIn is how the voice is meant to sound, in the author's words;
	// style the latest, which joins the next conversation's instructions
	// after VoiceLead.
	styleIn ports.TextInput
	style   string
	// VoiceLead introduces the voice style in the instructions; the genre's
	// words. Set before Start.
	VoiceLead string
	// Seat is this conversation's place where several share one observer:
	// an injection addressed to another seat passes it by. Empty takes
	// every injection. Set before Start.
	Seat string
	// Hear turns a transcript fragment into the one the transcript output
	// carries, such as a speaker named by seat, or drops it (false). Nil
	// passes every fragment as it is. Set before Start.
	Hear func(ports.Fragment) (ports.Fragment, bool)

	audio ports.AudioBroadcast
	text  ports.TextBroadcast
	// heard is the transcript as it comes, both speakers, with timing.
	heard ports.FragmentBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	// offers carries browser SDP offers; answering one creates the conversation.
	offers chan offer
	// invite and paused are lifecycle signals the client reads directly.
	invite ports.StateBroadcast
	paused ports.StateBroadcast
	// stopped says the game must end, for the reason it carries.
	stopped ports.StateBroadcast

	// idleAfter closes a silent conversation; zero never does. Keep it above the
	// 15 s each new session bills up front, or pausing costs more than it saves.
	idleAfter time.Duration
	// said is the transcript, both speakers, to restore a conversation that was let go.
	said []adapters.Utterance
	// resumeFrom names the closed conversation a resume continues.
	resumeFrom string
	// restored is a conversation taken back from a stored session: it waits
	// paused for the player, and the character does not greet again.
	restored bool
	// cue is the gate's opening instruction, held until the conversation
	// starts; guarded by mu.
	cue string
	// line is the opening's words to say, held the same way; guarded by mu.
	line     string
	active   chan struct{}
	pauseNow chan struct{}

	mu      sync.Mutex
	talking bool
	// conversing is whether a conversation exists, which sets the block's phase.
	conversing bool
	// stand is where the conversation is, for a page that has to be told.
	stand string
	// pending is the utterance so far; boundaries are judged on whole sentences.
	pending string
	closer  *time.Timer

	// journal is where the conversation's life is noted, for whoever debugs it.
	journal *ports.Journal

	// drops counts conversations dropped in a row with nothing said; guarded by mu.
	drops int
	// spoken notes whose first words of a conversation are journalled; guarded by mu.
	spoken map[adapters.Speaker]bool
	// notedPlayer is how far through said the player's lines are journalled; guarded by mu.
	notedPlayer int
}

// Standings tell a reloaded page what to do, as a live connection cannot be replayed.
const (
	// StandingIdle is before the game has begun. Nothing may connect yet.
	StandingIdle = "idle"
	// StandingAwaiting is ready for an offer, with nothing said so far.
	StandingAwaiting = "awaiting"
	// StandingOpen is a conversation running; another page may take it over.
	StandingOpen = "open"
	// StandingPaused is a conversation let go, waiting to be picked up.
	StandingPaused = "paused"
)

// ending says why one conversation stopped, and therefore what follows it.
type ending int

const (
	// endingOver: nothing follows, and the block is finished.
	endingOver ending = iota
	// endingIdle: let go for silence or on request; it waits to be picked up.
	endingIdle
	// endingReplaced: a second offer arrived, from a player whose browser lost the first.
	endingReplaced
	// endingDropped: the provider ended the conversation without being asked,
	// such as a lost connection; a new one carries on from the transcript.
	endingDropped
)

// maxDrops is how many conversations in a row may drop before anything is
// said in them; after that the conversation waits for the player's press,
// so a provider that keeps hanging up does not bill a new start each time.
const maxDrops = 3

// InvitePress is the reason on an invitation that waits for the player's
// press: a conversation let go, or one that could not go on by itself. An
// invitation without it is answered by connecting at once.
const InvitePress = "press"

func (e ending) String() string {
	switch e {
	case endingIdle:
		return "let go"
	case endingReplaced:
		return "replaced by a new offer"
	case endingDropped:
		return "dropped by the provider"
	}
	return "over, the block is finished"
}

// offer is one browser asking to connect, with channels for its answer.
type offer struct {
	sdp    string
	answer chan<- string
	fail   chan<- error
}

// An utterance needs both sentence-final punctuation and a silence; either alone
// cuts mid-clause or at every full stop. Gaps are generous: slow speakers pause
// between sentences, and fewer, longer cards are the better failure.
const (
	pauseGap = 2500 * time.Millisecond
	// After groupedRuns sentences a shorter silence splits a long monologue.
	groupedGap  = 1500 * time.Millisecond
	groupedRuns = 5
	// danglingGap ends speech that never reaches a full stop, such as a trailing-off line.
	danglingGap = 8 * time.Second
)

// NewLiveSession builds the conversation; it opens nothing until a player connects.
func NewLiveSession(name string, live adapters.Live, cfg adapters.LiveConfig, idleAfter time.Duration) *LiveSession {
	return &LiveSession{
		name:      name,
		live:      live,
		cfg:       cfg,
		idleAfter: idleAfter,
		mic:       make(ports.AudioInput, 32),
		steer:     make(ports.TextInput, 16),
		injected:  make(ports.InjectionInput, 16),
		opening:   make(ports.TextInput, 1),
		voiceIn:   make(ports.TextInput, 1),
		voiced:    make(chan struct{}),
		VoiceWait: 3 * time.Second,
		styleIn:   make(ports.TextInput, 1),
		offers:    make(chan offer, 4),
		active:    make(chan struct{}, 1),
		pauseNow:  make(chan struct{}, 1),
	}
}

func (b *LiveSession) NodeName() string { return b.name }

// SetGuardrail changes the youth-protection rule the next conversation is
// given on its own channel; an open one keeps the one it began with.
func (b *LiveSession) SetGuardrail(guardrail string) {
	b.mu.Lock()
	b.cfg.Guardrail = guardrail
	b.mu.Unlock()
}

// SetScenario changes the character's instructions from the next
// conversation on; an open one keeps those it began with.
func (b *LiveSession) SetScenario(scenario string) {
	b.mu.Lock()
	b.cfg.Scenario = scenario
	b.mu.Unlock()
}

func (b *LiveSession) UseJournal(j *ports.Journal)          { b.journal = j }
func (b *LiveSession) AudioInPort() chan<- ports.AudioChunk { return b.mic }

// TextInPort takes the gate's cue, and later instructions.
func (b *LiveSession) TextInPort() chan<- string { return b.steer }

// OpeningInPort takes words for the character to say when the first
// conversation starts, in its own words, after the opening instruction. A
// later line replaces one not yet said; a conversation continued after a
// pause or a restart says none.
func (b *LiveSession) OpeningInPort() chan<- string { return b.opening }

// VoiceInPort takes the voice to speak in, by name or id. Wiring it makes
// the first conversation wait for one.
func (b *LiveSession) VoiceInPort() chan<- string {
	b.mu.Lock()
	b.voiceWired = true
	b.mu.Unlock()
	return b.voiceIn
}

// SecondaryTextInPort takes how the voice is meant to sound.
func (b *LiveSession) SecondaryTextInPort() chan<- string { return b.styleIn }

// InjectionInPort takes what an observer puts into the conversation.
func (b *LiveSession) InjectionInPort() chan<- ports.Injection { return b.injected }

func (b *LiveSession) AudioOutPort() <-chan ports.AudioChunk { return b.audio.Subscribe() }

// TranscriptOutPort is the conversation as it is heard, both speakers, each
// fragment with its timing: for a block that follows the whole exchange.
func (b *LiveSession) TranscriptOutPort() <-chan ports.Fragment { return b.heard.Subscribe() }
func (b *LiveSession) TextOutPort() <-chan string               { return b.text.Subscribe() }

// RequiredInputs names audio so the wiring shows where the player's voice goes.
func (b *LiveSession) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindAudio}
}

// RequiredOutputs stops a genre wiring a character nobody can hear.
func (b *LiveSession) RequiredOutputs() []ports.Kind {
	return []ports.Kind{ports.KindAudio, ports.KindText}
}

func (b *LiveSession) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort reports the block's phase, set by converse.
func (b *LiveSession) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// Pause lets the conversation go now, without waiting out the idle timeout.
func (b *LiveSession) Pause() {
	b.journal.Note(b.name, "pause-requested", "")
	select {
	case b.pauseNow <- struct{}{}:
	default:
	}
}

// Pauses reports the conversation being let go, so a client can show it.
func (b *LiveSession) Pauses() <-chan ports.State { return b.paused.Subscribe() }

// Stops reports that the game must end, with the reason: the provider's
// safety filter ended the conversation (StopContentPolicy).
func (b *LiveSession) Stops() <-chan ports.State { return b.stopped.Subscribe() }

// StopContentPolicy is the reason on a stop when the provider's safety filter
// ended the conversation.
const StopContentPolicy = "content_policy"

// Standing answers a page that loaded after the invitation was sent.
func (b *LiveSession) Standing() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stand == "" {
		return StandingIdle
	}
	return b.stand
}

func (b *LiveSession) stands(standing string) {
	b.mu.Lock()
	from := b.stand
	b.stand = standing
	b.mu.Unlock()
	if from != standing {
		b.journal.Note(b.name, "standing", from+" → "+standing)
	}
}

// awaiting preserves a paused standing, which a page treats differently from idle.
func (b *LiveSession) awaiting() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stand == "" || b.stand == StandingIdle {
		b.stand = StandingAwaiting
	}
}

// Invitations tells a client when to connect, once the gate has opened.
func (b *LiveSession) Invitations() <-chan ports.State { return b.invite.Subscribe() }

// Connect brokers one browser's SDP offer and blocks until it is answered.
func (b *LiveSession) Connect(ctx context.Context, sdp string) (string, error) {
	answer := make(chan string, 1)
	fail := make(chan error, 1)

	select {
	case b.offers <- offer{sdp: sdp, answer: answer, fail: fail}:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	select {
	case sdp := <-answer:
		return sdp, nil
	case err := <-fail:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (b *LiveSession) Start(ctx context.Context) {
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	go b.holdOpening(ctx)
	go b.holdVoice(ctx)
	go b.holdStyle(ctx)
	go b.run(ctx)
}

// holdVoice keeps the voice made for the game, for every conversation to come.
func (b *LiveSession) holdVoice(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case voice, open := <-b.voiceIn:
			if !open {
				return
			}
			b.mu.Lock()
			b.voice = voice
			b.mu.Unlock()
			b.voicedOnce.Do(func() { close(b.voiced) })
		}
	}
}

// holdStyle keeps how the voice is meant to sound, for every conversation to come.
func (b *LiveSession) holdStyle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case style, open := <-b.styleIn:
			if !open {
				return
			}
			b.mu.Lock()
			b.style = style
			b.mu.Unlock()
		}
	}
}

// holdOpening keeps the latest opening line until a conversation says it.
func (b *LiveSession) holdOpening(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case line, open := <-b.opening:
			if !open {
				return
			}
			b.mu.Lock()
			if !b.restored {
				b.line = line
			}
			b.mu.Unlock()
		}
	}
}

// run waits for the gate's cue before inviting a player, and creates a session
// only once one connects, since a session bills from its first second.
func (b *LiveSession) run(ctx context.Context) {
	cue, ok := b.awaitCue(ctx)
	if !ok {
		return
	}
	b.mu.Lock()
	restored := b.restored
	if !restored {
		b.cue = cue
	}
	b.mu.Unlock()
	b.journal.Note(b.name, "cued", fmt.Sprintf("the gate opened (restored: %t)", restored))

	// One conversation per iteration; waiting holds a reconnecting player's offer.
	var waiting *offer
	for {
		if waiting == nil {
			b.awaiting()
			invitation := ports.State{Node: b.name, Phase: ports.PhaseReady}
			if b.Standing() == StandingPaused {
				invitation.Reason = InvitePress
			}
			b.invite.Send(invitation)
			b.journal.Note(b.name, "invited", "waiting for a browser's offer "+invitation.Reason)

			select {
			case <-ctx.Done():
				return
			case o := <-b.offers:
				waiting = &o
			}
		}
		b.journal.Note(b.name, "offer", fmt.Sprintf("an offer of %d bytes", len(waiting.sdp)))
		// A pause asked for while no conversation was open belongs to none;
		// left waiting, it would let the new one go at once.
		select {
		case <-b.pauseNow:
		default:
		}

		conn, err := b.open(ctx, *waiting)
		waiting = nil
		if err != nil {
			// Not the end of the game: the player tries again with a press,
			// once whatever stopped it has passed.
			b.stands(StandingPaused)
			b.paused.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			continue
		}
		b.stands(StandingOpen)
		b.mu.Lock()
		b.spoken = map[adapters.Speaker]bool{}
		b.mu.Unlock()
		b.converse(true)

		// What was queued while there was no conversation belongs to none: it
		// would reach the new one all at once.
		b.dropQueued()
		// The forwarding ends with its conversation, or a later one would
		// lose instructions and injections to a conversation already closed.
		conversation, endConversation := context.WithCancel(ctx)
		go b.instruct(conversation, conn)
		end, next := b.consume(ctx, conn)
		endConversation()
		b.journal.Note(b.name, "ended", fmt.Sprintf("conversation %s: %s", conn.ID(), end))
		if end == endingOver {
			b.stands(StandingIdle)
			return
		}
		b.mu.Lock()
		b.resumeFrom = conn.ID()
		b.mu.Unlock()

		if end == endingReplaced {
			// Continued at once, the same way a resume is.
			waiting = &next
			continue
		}

		if end == endingDropped {
			b.mu.Lock()
			b.drops++
			drops := b.drops
			b.mu.Unlock()
			if drops <= maxDrops {
				// The page connects again by itself, and the new
				// conversation carries on from the transcript.
				b.stands(StandingAwaiting)
				continue
			}
			b.journal.Warn(b.name, "dropping", fmt.Sprintf("%d conversations dropped in a row with nothing said; waiting for the player", drops))
		}

		b.stands(StandingPaused)
		b.paused.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}
}

// open passes both the prior ID and the transcript, whichever the provider honours.
func (b *LiveSession) open(ctx context.Context, waiting offer) (adapters.LiveConn, error) {
	b.awaitVoice(ctx)
	b.mu.Lock()
	cfg := b.cfg
	cfg.Offer = waiting.sdp
	cfg.ResumeFrom = b.resumeFrom
	if b.voice != "" {
		cfg.Voice = b.voice
	}
	if strings.TrimSpace(b.style) != "" {
		cfg.Scenario += "\n\n" + b.VoiceLead + b.style
	}
	b.mu.Unlock()
	cfg.History = b.transcript()

	b.journal.Note(b.name, "opening", fmt.Sprintf("resuming from %q, transcript of %d utterances", cfg.ResumeFrom, len(cfg.History)))
	conn, err := b.live.Open(ctx, cfg)
	if err != nil {
		b.journal.Fail(b.name, "open-failed", err.Error())
		waiting.fail <- err
		if !told(err) {
			b.notice("[engine] the conversation could not be opened: " + err.Error())
		}
		return nil, err
	}
	waiting.answer <- conn.Answer()
	b.journal.Note(b.name, "opened", "conversation "+conn.ID())
	return conn, nil
}

// awaitVoice waits, for a conversation wired to be given a voice and still
// without one, until the voice comes or VoiceWait has passed: the voice
// comes from preparation, which the browser may outrun.
func (b *LiveSession) awaitVoice(ctx context.Context) {
	b.mu.Lock()
	waits := b.voiceWired && b.voice == ""
	b.mu.Unlock()
	if !waits {
		return
	}
	select {
	case <-b.voiced:
	case <-time.After(b.VoiceWait):
	case <-ctx.Done():
	}
}

// liveState is what a stored session keeps of the conversation.
type liveState struct {
	From string               `json:"from,omitempty"`
	Said []adapters.Utterance `json:"said,omitempty"`
	// Voice is the voice it was given, so a restored conversation speaks
	// in it from its first word, before the voice is given again.
	Voice string `json:"voice,omitempty"`
}

// ExportState keeps the transcript, the last conversation's ID and the voice
// it was given, so a session restored after a restart resumes the way an
// idle one does, in the same voice. Live audio is never kept.
func (b *LiveSession) ExportState() string {
	b.mu.Lock()
	state := liveState{From: b.resumeFrom, Said: append([]adapters.Utterance(nil), b.said...), Voice: b.voice}
	b.mu.Unlock()
	if state.From == "" && len(state.Said) == 0 && state.Voice == "" {
		return ""
	}
	blob, err := json.Marshal(state)
	if err != nil {
		return ""
	}
	return string(blob)
}

// RestoreState takes the conversation back paused: one open before the
// restart is not re-attached, and the player resumes it with one press.
func (b *LiveSession) RestoreState(s string) {
	var state liveState
	if json.Unmarshal([]byte(s), &state) != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resumeFrom = state.From
	b.said = state.Said
	b.voice = state.Voice
	// What was said before is in the stored journal already.
	b.notedPlayer = len(state.Said)
	b.restored = true
	b.stand = StandingPaused
}

// Conversed says a conversation had been opened or something said, as
// restored from a stored session.
func (b *LiveSession) Conversed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.resumeFrom != "" || len(b.said) > 0
}

func (b *LiveSession) transcript() []adapters.Utterance {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]adapters.Utterance(nil), b.said...)
}

// remember keeps what was said, for a resume to restore.
func (b *LiveSession) remember(who adapters.Speaker, text string) {
	if text == "" {
		return
	}
	b.mu.Lock()
	// Fragments join up so a restored conversation reads as speech.
	if n := len(b.said); n > 0 && b.said[n-1].Speaker == who {
		b.said[n-1].Text += text
		b.mu.Unlock()
		return
	}
	b.said = append(b.said, adapters.Utterance{Speaker: who, Text: text})
	b.mu.Unlock()
	b.notePlayer(false)
}

// notePlayer writes the player's finished lines to the journal, as their words
// go nowhere else: each once the other side speaks, and, at the end of a
// conversation, the last one too, which a character who fell silent never
// answered.
func (b *LiveSession) notePlayer(ending bool) {
	b.mu.Lock()
	upTo := len(b.said) - 1
	if ending {
		upTo = len(b.said)
	}
	var lines []string
	for ; b.notedPlayer < upTo; b.notedPlayer++ {
		if u := b.said[b.notedPlayer]; u.Speaker == adapters.SpeakerPlayer {
			lines = append(lines, u.Text)
		}
	}
	b.mu.Unlock()
	for _, line := range lines {
		b.journal.Note(b.name, "player-said", line)
	}
}

// TranscriptLine is one turn of the conversation, as a reader sees it.
type TranscriptLine struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// The speakers a TranscriptLine names.
const (
	SpeakerCharacter = "character"
	SpeakerPlayer    = "player"
)

// Transcript is the conversation so far, both speakers, oldest first: what a
// resume continues from, for a reader to follow.
func (b *LiveSession) Transcript() []TranscriptLine {
	said := b.transcript()
	lines := make([]TranscriptLine, len(said))
	for i, u := range said {
		lines[i] = TranscriptLine{Speaker: SpeakerCharacter, Text: u.Text}
		if u.Speaker == adapters.SpeakerPlayer {
			lines[i].Speaker = SpeakerPlayer
		}
	}
	return lines
}

// greet sends the opening cue on session.started; the session ignores anything
// sent earlier. Only the first conversation gets it: later ones continue it, and
// a character should not greet again after a reconnect.
func (b *LiveSession) greet(ctx context.Context, conn adapters.LiveConn) {
	b.mu.Lock()
	line, cue := b.line, b.cue
	b.line, b.cue = "", ""
	b.mu.Unlock()
	if cue != "" {
		b.journal.Note(b.name, "cue", cue)
		if err := conn.Instruct(ctx, cue); err != nil {
			b.journal.Warn(b.name, "cue-rejected", err.Error())
		}
	}
	// Said rather than instructed: an instruction waits for the model's turn,
	// and the character is to speak first.
	if line != "" {
		b.journal.Note(b.name, "opening-line", line)
		if err := conn.SayAloud(ctx, line); err != nil {
			b.journal.Warn(b.name, "opening-line-rejected", err.Error())
		}
	}
}

// awaitCue blocks until the gate opens; an empty cue is a valid answer.
func (b *LiveSession) awaitCue(ctx context.Context) (string, bool) {
	select {
	case <-ctx.Done():
		return "", false
	case cue, open := <-b.steer:
		return cue, open
	}
}

// instruct forwards instructions and injections to the conversation.
func (b *LiveSession) instruct(ctx context.Context, conn adapters.LiveConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case text, open := <-b.steer:
			if !open {
				return
			}
			// Refused, it is the journal's: the transcript stays the conversation.
			if err := conn.Instruct(ctx, text); err != nil {
				b.journal.Warn(b.name, "instruction-rejected", err.Error())
			}
		case in, open := <-b.injected:
			if !open {
				return
			}
			if in.To != "" && b.Seat != "" && in.To != b.Seat {
				continue
			}
			if err := inject(ctx, conn, in); err != nil {
				b.journal.Warn(b.name, "injection-rejected", string(in.Kind)+": "+err.Error())
			}
		}
	}
}

// inject puts an injection into the conversation the way its kind asks.
func inject(ctx context.Context, conn adapters.LiveConn, in ports.Injection) error {
	switch in.Kind {
	case ports.InjectBackground:
		return conn.Inform(ctx, in.Text)
	case ports.InjectSpeech:
		return conn.SayAloud(ctx, in.Text)
	}
	return conn.Instruct(ctx, in.Text)
}

// consume runs one conversation and returns why it ended, which decides what follows.
func (b *LiveSession) consume(ctx context.Context, conn adapters.LiveConn) (end ending, next offer) {
	defer func() {
		// Said first, so later phases rest at ready.
		b.converse(false)
		b.stopCloser()
		b.flush()
		// Closed gracefully here: that finalises the recording the next conversation resumes from.
		conn.Close()
		if end == endingOver {
			b.closeAll()
		}
	}()

	go b.drainMic(ctx)

	// Idle silence runs from the last thing either speaker said, and never expires
	// while the character is still talking.
	var timeout <-chan time.Time
	var timer *time.Timer
	if b.idleAfter > 0 {
		timer = time.NewTimer(b.idleAfter)
		defer timer.Stop()
		timeout = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return endingOver, offer{}
		case <-b.pauseNow:
			// Taken at once, even mid-sentence: whoever pressed pause has stopped listening.
			return endingIdle, offer{}
		case o := <-b.offers:
			// A second offer means a reload dropped the first browser; the player is back.
			return endingReplaced, o
		case <-timeout:
			if b.speaking() {
				// Mid-sentence: wait again for the silence after it.
				timer.Reset(b.idleAfter)
				continue
			}
			return endingIdle, offer{}
		case <-b.active:
			if timer != nil {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(b.idleAfter)
			}
		case e, open := <-conn.Events():
			if !open {
				return endingDropped, offer{}
			}
			if e.Kind == adapters.EventStarted {
				b.journal.Note(b.name, "started", "the provider started conversation "+conn.ID())
				// Beside the loop, which must go on reading: a conversation may
				// answer the opening with events before the call returns.
				go b.greet(ctx, conn)
			}
			b.handle(e)
			if e.Kind == adapters.EventClosed {
				return closing(e.CloseReason), offer{}
			}
		}
	}
}

// stir resets the silence clock; dropping one nudge among many costs nothing.
func (b *LiveSession) stir() {
	select {
	case b.active <- struct{}{}:
	default:
	}
}

func (b *LiveSession) speaking() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.talking
}

// drainMic keeps the audio edge flowing; the real samples go browser-to-provider.
func (b *LiveSession) drainMic(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, open := <-b.mic:
			if !open {
				return
			}
		}
	}
}

func (b *LiveSession) handle(e adapters.LiveEvent) {
	switch e.Kind {
	case adapters.EventText:
		// Both speakers keep it alive and are remembered, so a resume has both sides.
		b.stir()
		b.firstWords(e.Speaker)
		b.remember(e.Speaker, e.Text)
		speaker := SpeakerCharacter
		if e.Speaker == adapters.SpeakerPlayer {
			speaker = SpeakerPlayer
		}
		fragment := ports.Fragment{Speaker: speaker, Text: e.Text, StartMs: e.StartMs, EndMs: e.EndMs, At: time.Now()}
		if b.Hear == nil {
			b.heard.Send(fragment)
		} else if heard, keep := b.Hear(fragment); keep {
			b.heard.Send(heard)
		}
		if e.Speaker == adapters.SpeakerPlayer {
			// The player's own words are remembered but shown to nobody.
			return
		}
		b.mu.Lock()
		b.drops = 0
		b.mu.Unlock()
		b.speak(true)
		b.text.Send(e.Text)
		b.reschedule(e.Text)

	case adapters.EventAudio:
		b.stir()
		if e.Speaker == adapters.SpeakerCharacter {
			b.audio.Send(ports.AudioChunk(e.Audio))
		}

	case adapters.EventUsage:
		// The provider's running total, so a dropped event costs only an instant's accuracy.
		b.usage.Send(ports.Usage{
			Node:         b.name,
			Model:        e.Usage.Model,
			AudioSeconds: e.Usage.AudioSeconds,
		})

	case adapters.EventClosed:
		b.stopCloser()
		b.flush()
		// GPT-Live says why it closed (session.closed's reason); only a lost
		// connection is a problem. It never closes for silence: letting a quiet
		// conversation go is the idle timeout's, here.
		switch e.CloseReason {
		case "", "close_requested":
			b.journal.Note(b.name, "closed", "the provider closed it as asked ("+e.CloseReason+")")
		case "remote_hangup":
			b.journal.Note(b.name, "closed", "the player's connection ended (remote_hangup): a reload or a closed page; a new page carries on")
		case "expired":
			b.journal.Note(b.name, "closed", "the conversation reached the provider's duration limit (expired); a new one carries on")
		case "content":
			b.journal.Warn(b.name, "closed", "the provider's safety filter ended the conversation (content)")
		default:
			b.journal.Warn(b.name, "closed", "the connection to the provider was lost ("+e.CloseReason+")")
		}
		if e.CloseReason == "content" {
			// Youth protection fired inside the model: the provider's decision,
			// which ends the game. The session tells the player why.
			b.stopped.Send(ports.State{Node: b.name, Phase: ports.PhaseReady, Reason: StopContentPolicy})
			return
		}
		// Any other close is carried on from with a new conversation, which the
		// journal records; the transcript stays the conversation.

	case adapters.EventError:
		// The journal keeps it, and Insights shows it among the problems; the
		// transcript stays the conversation. What the player must know, a
		// key refused or no credit left, the session tells in its own words,
		// and a lost connection is carried on from by itself.
		b.journal.Warn(b.name, "live-error", e.Err.Error())
	}
}

// reschedule pushes the boundary timer back with each new fragment.
func (b *LiveSession) reschedule(fragment string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pending += fragment
	if b.closer != nil {
		b.closer.Stop()
	}
	b.closer = time.AfterFunc(silenceAfter(b.pending), b.endUtterance)
}

// silenceAfter is how long the quiet after this much speech must last to end it.
func silenceAfter(pending string) time.Duration {
	switch {
	case !endsSentence(pending):
		// Mid-sentence: the speaker has not finished.
		return danglingGap
	case sentences(pending) >= groupedRuns:
		return groupedGap
	}
	return pauseGap
}

// endUtterance marks the boundary in-band as an empty delta, so it stays ordered with the text.
func (b *LiveSession) endUtterance() {
	b.mu.Lock()
	if b.pending == "" {
		b.mu.Unlock()
		return
	}
	b.pending = ""
	b.mu.Unlock()

	b.text.Send("")
	b.speak(false)
}

// notice is an engine message, an utterance of its own, so the history keeps it.
func (b *LiveSession) notice(text string) {
	b.endUtterance()
	b.text.Send(text)
	b.text.Send("")
}

// flush closes an utterance left open when the conversation stops.
func (b *LiveSession) flush() {
	b.notePlayer(true)
	b.endUtterance()
	b.speak(false)
}

func (b *LiveSession) stopCloser() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closer != nil {
		b.closer.Stop()
		b.closer = nil
	}
}

// sentences counts finished sentences; decimals and abbreviations only cause an early cut.
func sentences(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case '.', '!', '?', '…':
			n++
		}
	}
	return n
}

// endsSentence looks past trailing quotes and brackets so dialogue counts.
func endsSentence(s string) bool {
	trimmed := strings.TrimRight(s, " \t\n\"'»)]")
	if trimmed == "" {
		return false
	}
	runes := []rune(trimmed)
	switch runes[len(runes)-1] {
	case '.', '!', '?', '…':
		return true
	}
	return false
}

// speak tracks mid-utterance for the idle check; the phase belongs to converse.
func (b *LiveSession) speak(active bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.talking = active
}

// converse reports working for the whole open conversation, which bills by the
// second, and ready once it is let go.
func (b *LiveSession) converse(on bool) {
	b.mu.Lock()
	if b.conversing == on {
		b.mu.Unlock()
		return
	}
	b.conversing = on
	b.mu.Unlock()

	phase := ports.PhaseReady
	if on {
		phase = ports.PhaseWorking
	}
	b.state.Send(ports.State{Node: b.name, Phase: phase})
}

func (b *LiveSession) closeAll() {
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	b.audio.Close()
	b.text.Close()
	b.heard.Close()
	b.state.Close()
	b.usage.Close()
	b.invite.Close()
	b.paused.Close()
	b.stopped.Close()
}

// told says an error is one the session tells the player in its own words, a
// key refused or no credit left, so the transcript does not repeat the
// provider's.
func told(err error) bool {
	return errors.Is(err, adapters.ErrNoCredit) || errors.Is(err, adapters.ErrKeyRejected)
}

// closing says what follows a conversation the provider closed, by GPT-Live's
// reason: nothing after one closed as asked (close_requested) or by the
// model's safety filter (content); a new one after the others. A lost
// connection (connection_lost) is a drop; a reload ends the old page's
// connection gracefully (remote_hangup), and the new page carries on; a
// conversation at the duration limit (expired) was active, since a quiet one
// is let go by the idle timeout long before.
func closing(reason string) ending {
	switch reason {
	case "", "close_requested", "content":
		return endingOver
	}
	return endingDropped
}

// firstWords notes when each speaker first speaks in a conversation, so the
// journal shows how long a greeting took to come.
func (b *LiveSession) firstWords(who adapters.Speaker) {
	b.mu.Lock()
	first := b.spoken != nil && !b.spoken[who]
	if first {
		b.spoken[who] = true
	}
	b.mu.Unlock()
	if first {
		speaker := "the character"
		if who == adapters.SpeakerPlayer {
			speaker = "the player"
		}
		b.journal.Note(b.name, "first-words", speaker)
	}
}

// dropQueued lets go of instructions and injections that waited while no
// conversation was open, and notes how many.
func (b *LiveSession) dropQueued() {
	dropped := 0
	for {
		select {
		case <-b.steer:
			dropped++
		case <-b.injected:
			dropped++
		default:
			if dropped > 0 {
				b.journal.Note(b.name, "dropped-queued", fmt.Sprintf("%d instructions and injections sent while no conversation was open", dropped))
			}
			return
		}
	}
}
