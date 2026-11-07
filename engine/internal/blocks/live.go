package blocks

import (
	"context"
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

	audio ports.AudioBroadcast
	text  ports.TextBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	// offers carries browser SDP offers; answering one creates the conversation.
	offers chan offer
	// invite and paused are lifecycle signals the client reads directly.
	invite ports.StateBroadcast
	paused ports.StateBroadcast

	// idleAfter closes a silent conversation; zero never does. Keep it above the
	// 15 s each new session bills up front, or pausing costs more than it saves.
	idleAfter time.Duration
	// said is held in memory only, to restore a conversation that was let go.
	said []adapters.Utterance
	// resumeFrom names the closed conversation a resume continues.
	resumeFrom string
	// cue is the gate's opening instruction, held until the conversation starts.
	cue      string
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
)

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
		offers:    make(chan offer, 4),
		active:    make(chan struct{}, 1),
		pauseNow:  make(chan struct{}, 1),
	}
}

func (b *LiveSession) NodeName() string                     { return b.name }
func (b *LiveSession) AudioInPort() chan<- ports.AudioChunk { return b.mic }

// TextInPort fans in the gate's cue and the observer's corrections.
func (b *LiveSession) TextInPort() chan<- string { return b.steer }

func (b *LiveSession) AudioOutPort() <-chan ports.AudioChunk { return b.audio.Subscribe() }
func (b *LiveSession) TextOutPort() <-chan string            { return b.text.Subscribe() }

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
	select {
	case b.pauseNow <- struct{}{}:
	default:
	}
}

// Pauses reports the conversation being let go, so a client can show it.
func (b *LiveSession) Pauses() <-chan ports.State { return b.paused.Subscribe() }

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
	defer b.mu.Unlock()
	b.stand = standing
}

// awaiting preserves a paused standing, which a page treats differently from idle.
func (b *LiveSession) awaiting() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stand == "" || b.stand == StandingIdle {
		b.stand = StandingAwaiting
	}
}

// Invitations tells a client when to connect, after the portrait is on screen.
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
	go b.run(ctx)
}

// run waits for the gate's cue before inviting a player, and creates a session
// only once one connects, since a session bills from its first second.
func (b *LiveSession) run(ctx context.Context) {
	cue, ok := b.awaitCue(ctx)
	if !ok {
		return
	}
	b.cue = cue

	// One conversation per iteration; waiting holds a reconnecting player's offer.
	var waiting *offer
	for {
		if waiting == nil {
			b.awaiting()
			b.invite.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

			select {
			case <-ctx.Done():
				return
			case o := <-b.offers:
				waiting = &o
			}
		}

		conn, err := b.open(ctx, *waiting)
		waiting = nil
		if err != nil {
			return
		}
		b.stands(StandingOpen)
		b.converse(true)

		go b.instruct(ctx, conn)
		end, next := b.consume(ctx, conn)
		if end == endingOver {
			b.stands(StandingIdle)
			return
		}
		b.resumeFrom = conn.ID()

		if end == endingReplaced {
			// Continued at once, the same way a resume is.
			waiting = &next
			continue
		}

		b.stands(StandingPaused)
		b.paused.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}
}

// open passes both the prior ID and the transcript, whichever the provider honours.
func (b *LiveSession) open(ctx context.Context, waiting offer) (adapters.LiveConn, error) {
	cfg := b.cfg
	cfg.Offer = waiting.sdp
	cfg.ResumeFrom = b.resumeFrom
	cfg.History = b.transcript()

	conn, err := b.live.Open(ctx, cfg)
	if err != nil {
		waiting.fail <- err
		b.text.Send("[engine] the conversation could not be opened: " + err.Error())
		b.closeAll()
		return nil, err
	}
	waiting.answer <- conn.Answer()
	return conn, nil
}

func (b *LiveSession) transcript() []adapters.Utterance {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]adapters.Utterance(nil), b.said...)
}

// remember keeps what was said, in memory only, for a resume to restore.
func (b *LiveSession) remember(who adapters.Speaker, text string) {
	if text == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	// Fragments join up so a restored conversation reads as speech.
	if n := len(b.said); n > 0 && b.said[n-1].Speaker == who {
		b.said[n-1].Text += text
		return
	}
	b.said = append(b.said, adapters.Utterance{Speaker: who, Text: text})
}

// greet sends the opening cue on session.started; the session ignores anything
// sent earlier. Only the first conversation gets it: later ones continue it, and
// a character should not greet again after a reconnect.
func (b *LiveSession) greet(ctx context.Context, conn adapters.LiveConn) {
	if b.cue == "" {
		return
	}
	cue := b.cue
	b.cue = ""
	if err := conn.Instruct(ctx, cue); err != nil {
		b.text.Send("[engine] the opening cue was rejected: " + err.Error())
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

// instruct forwards the observer's corrections to the conversation.
func (b *LiveSession) instruct(ctx context.Context, conn adapters.LiveConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case text, open := <-b.steer:
			if !open {
				return
			}
			if err := conn.Instruct(ctx, strings.TrimPrefix(text, steerPrefix)); err != nil {
				b.text.Send("[engine] instruction rejected: " + err.Error())
			}
		}
	}
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
				return endingOver, offer{}
			}
			if e.Kind == adapters.EventStarted {
				b.greet(ctx, conn)
			}
			b.handle(e)
			if e.Kind == adapters.EventClosed {
				return endingOver, offer{}
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
		b.remember(e.Speaker, e.Text)
		if e.Speaker == adapters.SpeakerPlayer {
			// The player's own words are remembered but shown to nobody.
			return
		}
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
		if e.CloseReason == "content" {
			// Youth protection fired inside the model; say so plainly.
			b.text.Send("[engine] the conversation was ended by the model's safety filter")
			return
		}
		if e.CloseReason != "" && e.CloseReason != "close_requested" {
			b.text.Send("[engine] the conversation ended: " + e.CloseReason)
		}

	case adapters.EventError:
		b.text.Send("[engine] live error: " + e.Err.Error())
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

// flush closes an utterance left open when the conversation stops.
func (b *LiveSession) flush() {
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
	b.state.Close()
	b.usage.Close()
	b.invite.Close()
	b.paused.Close()
}
