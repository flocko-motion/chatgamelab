package blocks

import (
	"context"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// LiveSession is one full-duplex conversation with a character.
//
// The audio runs between the player's browser and the provider; this block
// holds the control connection. Its audio ports therefore describe the genre's
// dataflow rather than bytes crossing this process — the conversation really
// does take the player's speech and really does send speech back, and a graph
// omitting that would hide the one path this genre is about. Where the provider
// reflects audio to us those ports carry it, and where it does not they stand
// for it, with nothing in the wiring changing either way.
//
// The character and the constraint on it are configuration rather than ports:
// they are fixed before the session exists and never change. What flows in is
// appended instructions — the gate's opening cue, and the observer's
// corrections — because those arrive at moments nobody can predict.
//
// There is one of these, not a dummy and a real one: what varies is the adapter
// it holds.
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

	// offers carries a browser's SDP offer. The conversation cannot be created
	// before one arrives, because creating it is the act of answering it.
	offers chan offer
	// invite tells whoever is watching that a player may now open their own
	// connection, and paused that the conversation has been let go. Neither is
	// a graph edge, because their reader is the client rather than another
	// block — they are the session's lifecycle rather than its dataflow.
	invite ports.StateBroadcast
	paused ports.StateBroadcast

	// idleAfter is how long a conversation may sit with nobody speaking before
	// it is closed. Zero leaves it open for as long as it lasts.
	idleAfter time.Duration
	// said is the conversation so far, kept only to restore it. It is never
	// persisted and never leaves this process.
	said []adapters.Utterance
	// resumeFrom names the closed conversation a resume continues.
	resumeFrom string
	// cue is the gate's opening instruction, held until a conversation says it
	// is running. Cleared once delivered: it starts the game, and a resumed
	// conversation carries on rather than being greeted again.
	cue      string
	active   chan struct{}
	pauseNow chan struct{}

	mu      sync.Mutex
	talking bool
	// conversing is whether a conversation exists at all, which is what the
	// block rests at between utterances: open rather than ready.
	conversing bool
	// stand is where the conversation is, for a page that has to be told.
	stand string
	// pending is the character's utterance as it accumulates, held because a
	// boundary is judged against a whole sentence rather than a fragment.
	pending string
	closer  *time.Timer
}

// Where the conversation is, for a client that was not watching it happen. A
// live connection cannot be replayed into a reloaded page, so the page is told
// instead — and what it should do differs in each of these: open a connection,
// take one over, or wait to be asked.
const (
	// StandingIdle is before the game has begun. Nothing may connect yet.
	StandingIdle = "idle"
	// StandingAwaiting is ready for an offer, with nothing said so far.
	StandingAwaiting = "awaiting"
	// StandingOpen is a conversation running. A page arriving at one it is not
	// part of may offer to take it over, which continues it.
	StandingOpen = "open"
	// StandingPaused is a conversation let go, waiting to be picked up.
	StandingPaused = "paused"
)

// ending says why one conversation stopped, and therefore what follows it.
type ending int

const (
	// endingOver: nothing follows, and the block is finished.
	endingOver ending = iota
	// endingIdle: let go for want of anybody speaking, or because somebody
	// asked. It waits to be picked up.
	endingIdle
	// endingReplaced: a second offer arrived for a conversation that was
	// already running, which is a player coming back to one their browser lost.
	// Theirs is waiting to be answered.
	endingReplaced
)

// offer is one browser asking to connect, and the channels its answer comes
// back on.
type offer struct {
	sdp    string
	answer chan<- string
	fail   chan<- error
}

// What ends an utterance, given that GPT-Live marks no such thing.
//
// Two conditions, and both are needed. Silence alone cuts wherever the speaker
// drew breath: a character thinking mid-clause is not finished, and ending
// there puts half a sentence in one card and half in the next. Punctuation
// alone cuts at every full stop, which turns one reply into a column of boxes.
// So an utterance ends at a sentence, after a silence.
//
// The gaps are generous on purpose. A character speaking slowly — which a
// thoughtful one does, and which is most of the appeal — pauses longer between
// sentences than a transcript stream makes obvious, and every gap misjudged
// short is a reply broken into pieces that were meant to be read together.
// Fewer, longer cards are the better failure.
const (
	pauseGap = 2500 * time.Millisecond
	// Once groupedRuns sentences have gone by, a shorter silence is enough, so
	// a character talking at length is broken into readable pieces rather than
	// one wall of text.
	groupedGap  = 1500 * time.Millisecond
	groupedRuns = 5
	// The backstop, for speech that never reaches a full stop: a character who
	// trails off, or a transcript that loses its last fragment. Long enough
	// that it is never what ends an ordinary sentence, short enough that the
	// line does not stay open until the next reply runs into it.
	danglingGap = 8 * time.Second
)

// NewLiveSession builds the conversation. idleAfter is how long it may go
// unspoken before being closed; zero never closes it.
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

// TextInPort takes appended instructions: the gate's opening cue and the
// observer's corrections. Two edges into one port, because both are the same
// thing to the provider — text appended to the session's standing
// instructions — and fan-in is what a channel already does.
func (b *LiveSession) TextInPort() chan<- string { return b.steer }

func (b *LiveSession) AudioOutPort() <-chan ports.AudioChunk { return b.audio.Subscribe() }
func (b *LiveSession) TextOutPort() <-chan string            { return b.text.Subscribe() }

// RequiredInputs names what the graph must supply. Audio is here even though
// the browser carries it, because the wiring should state where a player's
// voice goes.
func (b *LiveSession) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindAudio}
}

// RequiredOutputs names what must consume this block. Without it a genre could
// wire a character nobody can hear and still pass validation, since every input
// it needed was satisfied.
func (b *LiveSession) RequiredOutputs() []ports.Kind {
	return []ports.Kind{ports.KindAudio, ports.KindText}
}

func (b *LiveSession) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort reports whether the character is speaking. With no turn events
// to read this is derived from transcript deltas arriving, which makes it a
// display signal rather than something the provider stated.
func (b *LiveSession) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// Pause lets the conversation go now, rather than waiting for the silence to
// prove it. Somebody who knows they are stepping away should not have to pay
// out the timeout to say so.
func (b *LiveSession) Pause() {
	select {
	case b.pauseNow <- struct{}{}:
	default:
	}
}

// Pauses reports the conversation being let go for want of anybody speaking.
// A client shows this rather than a player discovering it by talking to
// somebody who is no longer there.
func (b *LiveSession) Pauses() <-chan ports.State { return b.paused.Subscribe() }

// Standing is where the conversation is right now. A page that was watching
// learns this from the stream; one that has just loaded has to ask, because the
// invitation it would have acted on was sent before it existed.
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

// awaiting says an offer would be taken, without forgetting that a conversation
// has already happened: "let go" and "never started" both wait here, and they
// want different things from a page — a control to press, or a moment's wait.
func (b *LiveSession) awaiting() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stand == "" || b.stand == StandingIdle {
		b.stand = StandingAwaiting
	}
}

// Invitations closes over the moments a player may open their own connection.
// A client waits for one rather than connecting when the page loads, so the
// portrait is on screen before anybody is asked to speak.
func (b *LiveSession) Invitations() <-chan ports.State { return b.invite.Subscribe() }

// Connect brokers one browser's SDP offer, blocking until the conversation
// exists, because the answer is what the caller came for.
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

// run waits for the game to begin, invites a player to connect, and then holds
// the conversation.
//
// Both waits are deliberate. Nothing may be said before the gate opens, so the
// cue is what starts this; and creating a session bills for it from the first
// second, so one is created when a player is actually there.
func (b *LiveSession) run(ctx context.Context) {
	cue, ok := b.awaitCue(ctx)
	if !ok {
		return
	}
	b.cue = cue

	// Each turn of this loop is one conversation. A paused session ends its
	// conversation and waits to be invited back, which is what stops an
	// abandoned game billing by the minute until somebody notices.
	//
	// waiting carries an offer that arrived during the conversation before it,
	// which is a player coming back to one their browser lost.
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
			// The same continuation a resume gets, with no pause in front of
			// it: the conversation they left is closed, and its successor
			// carries what was said either by forking the recording or by being
			// seeded with it.
			waiting = &next
			continue
		}

		b.stands(StandingPaused)
		b.paused.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}
}

// open starts or continues a conversation and answers the browser waiting on
// it. A resume names the conversation it follows and carries what was said, so
// whichever of the two the provider can honour, the character remembers.
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

// remember keeps what was said, for restoring the conversation if it is let go.
// It is held in memory for the session's lifetime and reaches no sink and no
// persistence: its only reader is the resume.
func (b *LiveSession) remember(who adapters.Speaker, text string) {
	if text == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	// Fragments of one utterance join up, so a restored conversation reads as
	// speech rather than as a list of deltas.
	if n := len(b.said); n > 0 && b.said[n-1].Speaker == who {
		b.said[n-1].Text += text
		return
	}
	b.said = append(b.said, adapters.Utterance{Speaker: who, Text: text})
}

// greet delivers the gate's opening cue — the instruction that has the
// character speak first, rather than wait to be spoken to.
//
// It waits for the session to say it is running, and does not send the cue when
// the conversation is created. The provider marks "running, and accepting
// commands" with an event of its own, and an instruction written before it is
// one the session never had: the symptom is a character who was told to greet
// somebody and says nothing at all.
//
// Delivered once, to the first conversation. Every one after it continues that
// conversation, and a character who greeted somebody ten minutes ago should not
// do it again because their browser reconnected.
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

// awaitCue blocks until the gate opens the game. A gate wired with no cue still
// reports, so the empty string is a legitimate answer rather than a timeout.
func (b *LiveSession) awaitCue(ctx context.Context) (string, bool) {
	select {
	case <-ctx.Done():
		return "", false
	case cue, open := <-b.steer:
		return cue, open
	}
}

// instruct carries appended instructions to the conversation: the gate's cue,
// and whatever the observer decides the character needs telling.
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

// consume runs one conversation and says why it ended, which is what decides
// whether anything follows it. A conversation let go for silence can be picked
// up again, and one a returning player displaced is continued at once; one that
// ended any other way is over.
func (b *LiveSession) consume(ctx context.Context, conn adapters.LiveConn) (end ending, next offer) {
	defer func() {
		// Said first, so the phases that follow rest at ready rather than at a
		// conversation that has already ended.
		b.converse(false)
		b.stopCloser()
		b.flush()
		// Closed here rather than by whoever follows: a graceful close is what
		// finalises the provider's recording, and that recording is the better
		// of the two ways the next conversation continues this one.
		conn.Close()
		if end == endingOver {
			b.closeAll()
		}
	}()

	go b.drainMic(ctx)

	// Silence is measured from the last thing anybody said. The provider marks
	// no such thing, and a gap between transcript events is not by itself
	// silence — so the clock is reset by speech from either side, and only
	// while nothing is being spoken does it run down.
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
			// Asked for rather than waited out, so it is taken at once — even
			// mid-sentence, because somebody pressing pause has stopped
			// listening anyway.
			return endingIdle, offer{}
		case o := <-b.offers:
			// A second offer for a conversation that is already running means
			// the browser holding the first one is gone: a reload takes its
			// audio connection with it and leaves the provider talking to
			// nobody. The player is here, asking to carry on.
			return endingReplaced, o
		case <-timeout:
			if b.speaking() {
				// Mid-sentence is no time to hang up. Wait again, and let the
				// silence after it be the one that counts.
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

// stir resets the silence clock. Non-blocking, because this is called from the
// event path and losing one nudge among many costs nothing.
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

// drainMic consumes the audio port. The browser sends the real samples straight
// to the provider, so what arrives here is whatever the wiring put on the edge —
// and leaving it unread would block whatever is upstream.
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
		// Either speaker keeps the conversation alive, and either is worth
		// remembering for a resume — a character restored with only its own
		// remarks would be answering nothing.
		b.stir()
		b.remember(e.Speaker, e.Text)
		if e.Speaker == adapters.SpeakerPlayer {
			// Received, remembered, and shown to nobody. Nobody re-reads what
			// they just said aloud, and it is kept out of the record.
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
		// The provider's running total rather than a count kept here, so a
		// dropped event costs accuracy for an instant rather than for the rest
		// of the conversation.
		b.usage.Send(ports.Usage{
			Node:         b.name,
			Model:        e.Usage.Model,
			AudioSeconds: e.Usage.AudioSeconds,
		})

	case adapters.EventClosed:
		b.stopCloser()
		b.flush()
		if e.CloseReason == "content" {
			// The provider's own safety filter stopped the conversation. That
			// is youth protection firing inside the model, and it should be
			// visible as itself rather than as a connection that went away.
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

// reschedule restarts the silence that ends an utterance. It cannot be decided
// when a fragment arrives, because the silence that matters is the one after
// it — so the boundary is a timer that each new fragment pushes back.
func (b *LiveSession) reschedule(fragment string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pending += fragment
	if b.closer != nil {
		b.closer.Stop()
	}
	b.closer = time.AfterFunc(silenceAfter(b.pending), b.endUtterance)
}

// endUtterance closes the utterance. The boundary travels in-band as an empty
// delta: a marker on the state stream could not bracket anything, because the
// two reach a reader through different goroutines and nothing would order it
// against the text it is meant to close.
// silenceAfter is how long the quiet following this much speech has to last
// before the speech counts as an utterance.
func silenceAfter(pending string) time.Duration {
	switch {
	case !endsSentence(pending):
		// Mid-sentence. Whatever the silence means, the speaker has not
		// finished saying this one.
		return danglingGap
	case sentences(pending) >= groupedRuns:
		return groupedGap
	}
	return pauseGap
}

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

// flush ends an utterance left open when the conversation stops, so a reader is
// not waiting for a boundary that is never coming.
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

// sentences counts the finished sentences in an utterance so far. Decimals and
// abbreviations inflate it a little, which costs an early cut on a long reply
// and nothing else.
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

// endsSentence reports whether the text so far reads as a finished sentence,
// looking past trailing quotes and brackets so a line of dialogue still counts.
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

// speak reports the character talking, emitting only on a change so a reader
// sees one pair per utterance.
// speak tracks whether the character is mid-utterance. It reports no phase:
// this block is working for as long as its conversation is open, and lighting
// it per utterance would have it go dark between sentences while it holds a
// connection that is still costing money.
func (b *LiveSession) speak(active bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.talking = active
}

// converse is the block's phase, because the conversation is the block's work.
// It opens a connection, moves audio both ways and bills by the second from the
// moment it exists, whether or not anybody is speaking into it.
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
