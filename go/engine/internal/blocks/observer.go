package blocks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Injectable is a block that takes an injection during play and puts it into
// the conversation it watches.
type Injectable interface {
	Inject(kind ports.InjectionKind, text string) error
}

// Observer follows a live conversation from the side and steers it. Its input
// is the conversation's transcript, both speakers with their timing; its
// output is what it puts into the conversation.
//
// Fragments wait in a pool and leave it on a beat, sorted by when they were
// said, neighbouring fragments of one speaker joined into a line. While
// nobody speaks, a notice of how long each side has been quiet joins them.
// The lines go to a model, one call at a time, on a log of its own: the
// transcript, the notices and its earlier replies. It thinks aloud; a line of
// its reply starting with a marker becomes an injection. What a player
// injects by hand is passed on at once and joins the log too, so the model
// knows it.
type Observer struct {
	name string
	cfg  ObserverConfig

	in ports.FragmentInput
	// conversation reports whether the conversation it follows is open:
	// working while it is, ready while it is not.
	conversation ports.StateInput
	queued       chan ports.Injection
	out          ports.InjectionBroadcast
	state        ports.StateBroadcast
	meter        meter

	journal *ports.Journal

	mu sync.Mutex
	// pool holds fragments not yet released; pending, lines waiting for the model.
	pool    []ports.Fragment
	pending []ObserverEntry
	log     []ObserverEntry
	beat    time.Duration
	// latest is the latest provider time released, for telling a late fragment.
	latest int64
	// lastSaid is when each speaker last said something, and lastAny when anyone did.
	lastSaid map[string]time.Time
	lastAny  time.Time
	// quietNotice is how long the next silence notice waits, doubling while it lasts.
	quietNotice time.Duration
	noticedAt   time.Time
	busySince   time.Time
	// called is when its model was last called, for the gap between calls.
	called time.Time
	inARow int
	// reminded is when the rules last went to the character.
	reminded time.Time
	// open says the conversation is open; with none, it neither calls nor notices.
	open bool
	// held is a direction waiting for a pause in the conversation; heldTo
	// the seat it is for, empty for all.
	held   string
	heldTo string
	// lastDirection is the last direction sent; awaiting says the character
	// has not yet finished an answer since, and directedAt is when it went in.
	lastDirection string
	awaiting      bool
	directedAt    time.Time
	// openedAt is when the conversation last opened; standingDown says the
	// player has been silent for Absent since their last words or since then.
	openedAt     time.Time
	standingDown bool
	figures      ObserverFigures
}

// ObserverConfig is what a genre gives an observer.
type ObserverConfig struct {
	// Tool is what it thinks with; nil leaves it to pass on what a player injects.
	Tool adapters.Tool
	// Brief is its system prompt: what it watches for and how it answers.
	Brief string
	// Beat is how long a fragment waits in the pool; zero takes DefaultBeat.
	Beat time.Duration
	// Quiet is the silence after which the first notice joins the lines; zero
	// takes DefaultQuiet. Each further notice waits twice as long.
	Quiet time.Duration
	// InARow is how many replies in a row may carry a direction; zero takes
	// DefaultInARow. One more is dropped, and a reply without one resets it.
	InARow int
	// Pause is how long both sides must be quiet before a direction goes in,
	// and how long the player must be before their line counts as finished;
	// zero takes DefaultPause. A direction interrupts the character, so it
	// waits for a pause in the conversation.
	Pause time.Duration
	// Gap is the least time between two calls of its model; zero takes
	// DefaultGap. What arrives meanwhile waits and goes in with the next.
	Gap time.Duration
	// Absent is how long the player may say nothing before the observer takes
	// them for gone and stops nudging, so the conversation stalls and the live
	// session's idle close pauses the game; zero takes DefaultAbsent. It
	// carries on once the player speaks again.
	Absent time.Duration
	// Rules are what the conversation keeps to, whatever else is said. When
	// the model marks them at risk, they go to the character whole, as its
	// own knowledge, after RulesPreface; empty leaves the mark unanswered.
	Rules        string
	RulesPreface string
	// RulesGap is the least time between two reminders of the rules; zero
	// takes DefaultRulesGap.
	RulesGap time.Duration
	// Cast are the seats of the characters it watches where there are
	// several, each a speaker in the transcript and an addressee of what it
	// sends; empty is one character, SpeakerCharacter. With a cast, a marker
	// names its seat, or BOTH, e.g. "[BACKGROUND A]", and "[SPEECH A]" has
	// that character say something in its own words.
	Cast []string
	// KeepNudging never stands down for a silent player: the characters
	// carry the conversation themselves, so a player who only listens is
	// still there.
	KeepNudging bool
}

// The observer's defaults.
const (
	// DefaultBeat collects a few seconds per round, so the model reads
	// exchanges rather than scraps, and is called a few times a minute.
	DefaultBeat     = 3 * time.Second
	DefaultQuiet    = 2 * time.Second
	DefaultInARow   = 2
	DefaultPause    = 1500 * time.Millisecond
	DefaultGap      = 3 * time.Second
	DefaultAbsent   = time.Minute
	DefaultRulesGap = 2 * time.Minute
	// logLimit is how many characters of its log a call carries, newest kept.
	logLimit = 12000
	// logKept is how many entries it keeps for a reader.
	logKept = 400
)

// Quiet is the whole reply of a model that finds nothing to do: counted, and
// kept off the log, which it rereads each time.
const Quiet = "."

// The markers that turn a line of the model's reply into an injection.
// MarkRules says the rules are at risk; the observer then reminds the
// character of them in their own words, never the model's.
const (
	MarkDirection  = "[DIRECTION]"
	MarkBackground = "[BACKGROUND]"
	MarkRules      = "[RULES]"
	// MarkSpeech has a character of a cast say something in its own words.
	MarkSpeech = "[SPEECH]"
	// SeatAll addresses every character of a cast.
	SeatAll = "BOTH"
)

// The kinds of entry on an observer's log.
const (
	EntryLine       = "line"
	EntrySilence    = "silence"
	EntryInjection  = "injection"
	EntryThought    = "thought"
	EntryDirection  = "direction"
	EntryBackground = "background"
	EntryDropped    = "dropped"
	EntryError      = "error"
	// EntryHeld is a direction waiting for a pause in the conversation.
	EntryHeld = "held"
	// EntryConversation is the conversation opening or closing.
	EntryConversation = "conversation"
	// EntryAbsent is the observer standing down for a silent player, or
	// listening again when they speak.
	EntryAbsent = "absent"
	// EntryRules is the rules sent to the character as its own knowledge.
	EntryRules = "rules"
	// EntrySpeech is a line a character of a cast was given to say.
	EntrySpeech = "speech"
)

// ObserverEntry is one thing on an observer's log.
type ObserverEntry struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Speaker string    `json:"speaker,omitempty"`
	Text    string    `json:"text"`
	// Late says a line held a fragment that arrived after later ones had left the pool.
	Late bool `json:"late,omitempty"`
}

// ObserverFigures count what an observer has done.
type ObserverFigures struct {
	Fragments      int   `json:"fragments"`
	Flushes        int   `json:"flushes"`
	Lines          int   `json:"lines"`
	Late           int   `json:"late"`
	LargestLateMs  int64 `json:"largestLateMs"`
	SilenceNotices int   `json:"silenceNotices"`
	Calls          int   `json:"calls"`
	Errors         int   `json:"errors"`
	LastCallMs     int64 `json:"lastCallMs"`
	Directions     int   `json:"directions"`
	Backgrounds    int   `json:"backgrounds"`
	Dropped        int   `json:"dropped"`
	Injected       int   `json:"injected"`
	// Reminders counts the times the rules went to the character.
	Reminders int `json:"reminders"`
	// Quiet counts the replies that found nothing to do.
	Quiet int `json:"quiet"`
}

// ObserverState is where an observer stands now, for a reader.
type ObserverState struct {
	Thinking bool  `json:"thinking"`
	BeatMs   int64 `json:"beatMs"`
	// GapMs is the least time between two calls of its model.
	GapMs   int64 `json:"gapMs"`
	QuietMs int64 `json:"quietMs"`
	// ThinkingForMs is how long the call in flight has run.
	ThinkingForMs int64 `json:"thinkingForMs"`
	// Pool is the fragments waiting to leave it; Pending, the lines waiting for the model.
	Pool    int `json:"pool"`
	Pending int `json:"pending"`
	// CharacterAgoMs and PlayerAgoMs are how long ago each last spoke, -1 if never;
	// QuietForMs how long nobody has.
	CharacterAgoMs int64 `json:"characterAgoMs"`
	PlayerAgoMs    int64 `json:"playerAgoMs"`
	QuietForMs     int64 `json:"quietForMs"`
	// InARow is the replies in a row that carried a direction, of InARowLimit.
	InARow int `json:"inARow"`
	// Held is a direction waiting for a pause; Awaiting says the character
	// has not finished an answer since the last direction.
	Held        string `json:"held,omitempty"`
	Awaiting    bool   `json:"awaiting"`
	InARowLimit int    `json:"inARowLimit"`
	// StandingDown says the player has been silent for AbsentMs and the
	// observer has stopped nudging until they speak.
	StandingDown bool            `json:"standingDown"`
	AbsentMs     int64           `json:"absentMs"`
	Figures      ObserverFigures `json:"figures"`
	Log          []ObserverEntry `json:"log"`
}

// maxInjection keeps an injection inside what a live provider takes in one
// append (GPT-Live: 500 tokens), at a safe four characters a token.
const maxInjection = 2000

var (
	errBlankInjection = errors.New("an injection needs text")
	errLongInjection  = fmt.Errorf("an injection takes at most %d characters", maxInjection)
	errQueueFull      = errors.New("too many injections are still waiting to be passed on")
)

func NewObserver(name string, cfg ObserverConfig) *Observer {
	if cfg.Beat <= 0 {
		cfg.Beat = DefaultBeat
	}
	if cfg.Quiet <= 0 {
		cfg.Quiet = DefaultQuiet
	}
	if cfg.InARow <= 0 {
		cfg.InARow = DefaultInARow
	}
	if cfg.Pause <= 0 {
		cfg.Pause = DefaultPause
	}
	if cfg.Gap <= 0 {
		cfg.Gap = DefaultGap
	}
	if cfg.RulesGap <= 0 {
		cfg.RulesGap = DefaultRulesGap
	}
	if cfg.Absent <= 0 {
		cfg.Absent = DefaultAbsent
	}
	return &Observer{
		name:         name,
		cfg:          cfg,
		in:           make(ports.FragmentInput, 256),
		conversation: make(ports.StateInput, 16),
		queued:       make(chan ports.Injection, 8),
		meter:        meter{name: name},
		beat:         cfg.Beat,
		lastSaid:     map[string]time.Time{},
		quietNotice:  cfg.Quiet,
	}
}

func (b *Observer) NodeName() string                        { return b.name }
func (b *Observer) TranscriptInPort() chan<- ports.Fragment { return b.in }

// StateInPort takes the state of the conversation it follows: working while
// it is open. While none is, the observer does nothing: no calls, no notices
// of silence, nothing waiting for the model; what a call in flight brings
// back is not sent.
func (b *Observer) StateInPort() chan<- ports.State          { return b.conversation }
func (b *Observer) InjectionOutPort() <-chan ports.Injection { return b.out.Subscribe() }
func (b *Observer) StateOutPort() <-chan ports.State         { return b.state.Subscribe() }
func (b *Observer) UsageOutPort() <-chan ports.Usage         { return b.meter.usage.Subscribe() }
func (b *Observer) UseJournal(j *ports.Journal)              { b.journal = j }
func (b *Observer) RequiredInputs() []ports.Kind             { return []ports.Kind{ports.KindTranscript} }
func (b *Observer) RequiredOutputs() []ports.Kind            { return []ports.Kind{ports.KindInjection} }

// SetBeat changes how long fragments wait in the pool, from the next beat on.
func (b *Observer) SetBeat(beat time.Duration) error {
	if beat < 100*time.Millisecond || beat > 10*time.Second {
		return fmt.Errorf("a beat of %s: want 100ms to 10s", beat)
	}
	b.mu.Lock()
	b.beat = beat
	b.mu.Unlock()
	return nil
}

// Inject puts text into the conversation, of a kind that says how the model
// takes it, and onto the log. An unknown kind, blank or overlong text is
// refused, and so is an injection while too many are still waiting.
func (b *Observer) Inject(kind ports.InjectionKind, text string) error {
	if !slices.Contains(ports.InjectionKinds, kind) {
		return fmt.Errorf("no injection kind %q (want one of %v)", kind, ports.InjectionKinds)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errBlankInjection
	}
	if utf8.RuneCountInString(text) > maxInjection {
		return errLongInjection
	}
	select {
	case b.queued <- ports.Injection{Kind: kind, Text: text}:
		return nil
	default:
		return errQueueFull
	}
}

// State is where the observer stands now.
func (b *Observer) State() ObserverState {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	ago := func(t time.Time) int64 {
		if t.IsZero() {
			return -1
		}
		return now.Sub(t).Milliseconds()
	}
	s := ObserverState{
		Thinking:       !b.busySince.IsZero(),
		BeatMs:         b.beat.Milliseconds(),
		GapMs:          b.cfg.Gap.Milliseconds(),
		QuietMs:        b.cfg.Quiet.Milliseconds(),
		Pool:           len(b.pool),
		Pending:        len(b.pending),
		CharacterAgoMs: ago(b.characterSaidLocked()),
		PlayerAgoMs:    ago(b.lastSaid[SpeakerPlayer]),
		QuietForMs:     ago(b.lastAny),
		InARow:         b.inARow,
		Held:           b.held,
		Awaiting:       b.awaiting,
		InARowLimit:    b.cfg.InARow,
		StandingDown:   b.standingDown,
		AbsentMs:       b.cfg.Absent.Milliseconds(),
		Figures:        b.figures,
		Log:            append([]ObserverEntry(nil), b.log...),
	}
	if s.Thinking {
		s.ThinkingForMs = now.Sub(b.busySince).Milliseconds()
	}
	return s
}

// reply is what one call came back with.
type reply struct {
	text string
	took time.Duration
	err  error
}

func (b *Observer) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close(); b.meter.usage.Close() }()
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		replies := make(chan reply, 1)
		tick := time.NewTimer(b.currentBeat())
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case f, open := <-b.in:
				if !open {
					return
				}
				b.collect(f)
			case st, alive := <-b.conversation:
				if !alive {
					b.conversation = nil
					continue
				}
				b.follow(st.Phase == ports.PhaseWorking)
			case in := <-b.queued:
				// Passed on at once, and onto the log, so the model knows.
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				b.out.Send(in)
				b.note(ObserverEntry{At: time.Now(), Kind: EntryInjection, Speaker: string(in.Kind), Text: in.Text}, true)
				// One line in the journal for what went in: the observer's, as it passed it on.
				b.journal.Note(b.name, "injected", string(in.Kind)+": "+in.Text)
				b.mu.Lock()
				b.figures.Injected++
				thinking := !b.busySince.IsZero()
				b.mu.Unlock()
				if !thinking {
					b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
				}
			case <-tick.C:
				now := time.Now()
				b.flush(now)
				b.release(now)
				b.think(ctx, replies)
				tick.Reset(b.currentBeat())
			case r := <-replies:
				b.answered(r)
				b.think(ctx, replies)
			}
		}
	}()
}

func (b *Observer) currentBeat() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.beat
}

// follow notes the conversation opening or closing; closed, what waits is let go.
func (b *Observer) follow(open bool) {
	b.mu.Lock()
	if b.open == open {
		b.mu.Unlock()
		return
	}
	b.open = open
	b.standingDown = false
	if open {
		// A player's silence counts from this conversation, not from before a pause.
		b.openedAt = time.Now()
	}
	dropped := 0
	if !open {
		// Closed, what waits belongs to no conversation; opening keeps what
		// arrived with it.
		dropped = len(b.pool) + len(b.pending)
		b.pool, b.pending = nil, nil
		b.held, b.awaiting = "", false
		// A silence counts from the next conversation, not from before a pause.
		b.lastAny, b.noticedAt, b.quietNotice = time.Time{}, time.Time{}, b.cfg.Quiet
	}
	b.mu.Unlock()
	text := "the conversation is open"
	if !open {
		text = "the conversation is closed; nothing is sent until it opens again"
		if dropped > 0 {
			text += fmt.Sprintf(" (%d waiting let go)", dropped)
		}
	}
	b.note(ObserverEntry{At: time.Now(), Kind: EntryConversation, Text: text}, false)
}

// collect puts a fragment in the pool; speech ends a silence, and the
// player's words end standing down.
func (b *Observer) collect(f ports.Fragment) {
	if f.At.IsZero() {
		f.At = time.Now()
	}
	b.mu.Lock()
	b.figures.Fragments++
	b.pool = append(b.pool, f)
	b.lastSaid[f.Speaker] = f.At
	b.lastAny = f.At
	b.quietNotice = b.cfg.Quiet
	b.noticedAt = time.Time{}
	back := b.standingDown && f.Speaker == SpeakerPlayer
	if back {
		b.standingDown = false
	}
	b.mu.Unlock()
	if back {
		b.note(ObserverEntry{At: f.At, Kind: EntryAbsent, Text: "the player speaks again; listening again"}, false)
		b.journal.Note(b.name, "player-back", "")
	}
}

// absentLocked says the player has said nothing for Absent, since their last
// words or, if none in this conversation, since it opened. The caller holds mu.
func (b *Observer) absentLocked(now time.Time) bool {
	if b.cfg.KeepNudging || !b.open || b.openedAt.IsZero() {
		return false
	}
	since := b.openedAt
	if last := b.lastSaid[SpeakerPlayer]; last.After(since) {
		since = last
	}
	return now.Sub(since) >= b.cfg.Absent
}

// flush releases what has waited a beat, as lines, and notices a silence.
func (b *Observer) flush(now time.Time) {
	b.mu.Lock()
	var out, kept []ports.Fragment
	for _, f := range b.pool {
		if now.Sub(f.At) >= b.beat {
			out = append(out, f)
		} else {
			kept = append(kept, f)
		}
	}
	b.pool = kept
	// A player mid-sentence holds everything back, so their line reaches the
	// model whole and the order stays as said.
	if b.playerUnfinishedLocked(out, now) {
		b.pool = append(out, b.pool...)
		out = nil
	}
	// Sorted by when they were said, where the provider says; else by arrival.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartMs != 0 && out[j].StartMs != 0 {
			return out[i].StartMs < out[j].StartMs
		}
		return out[i].At.Before(out[j].At)
	})
	var lines []ObserverEntry
	for _, f := range out {
		late := f.StartMs != 0 && f.StartMs < b.latest
		if late {
			b.figures.Late++
			b.figures.LargestLateMs = max(b.figures.LargestLateMs, b.latest-f.StartMs)
		}
		b.latest = max(b.latest, f.StartMs)
		if n := len(lines); n > 0 && lines[n-1].Speaker == f.Speaker {
			lines[n-1].Text += f.Text
			lines[n-1].Late = lines[n-1].Late || late
			continue
		}
		lines = append(lines, ObserverEntry{At: f.At, Kind: EntryLine, Speaker: f.Speaker, Text: f.Text, Late: late})
	}
	if len(lines) > 0 {
		b.figures.Flushes++
		b.figures.Lines += len(lines)
	}
	// A player silent for Absent may be gone: no more nudges, so the
	// conversation stalls and the live session's idle close pauses the game.
	// What waits for the model, and a held direction, are let go.
	var gone *ObserverEntry
	if !b.standingDown && b.absentLocked(now) {
		b.standingDown = true
		b.pending, b.held = nil, ""
		gone = &ObserverEntry{At: now, Kind: EntryAbsent, Text: fmt.Sprintf(
			"the player has said nothing for %.0f s and may be gone; no more nudges until they speak", b.cfg.Absent.Seconds())}
	}
	// A silence, once the conversation has begun, is noticed at Quiet, then
	// at twice the wait each time while it lasts.
	var notice *ObserverEntry
	if b.open && !b.standingDown && len(b.pool) == 0 && !b.lastAny.IsZero() {
		since := b.lastAny
		if !b.noticedAt.IsZero() {
			since = b.noticedAt
		}
		if now.Sub(since) >= b.quietNotice {
			notice = &ObserverEntry{At: now, Kind: EntrySilence, Text: b.silenceLocked(now)}
			b.noticedAt = now
			b.quietNotice *= 2
			b.figures.SilenceNotices++
		}
	}
	b.mu.Unlock()
	for _, line := range lines {
		line.Text = strings.TrimSpace(line.Text)
		if line.Text != "" {
			b.note(line, true)
		}
	}
	if notice != nil {
		b.note(*notice, true)
	}
	if gone != nil {
		b.note(*gone, false)
		b.journal.Note(b.name, "player-absent", gone.Text)
	}
}

// silenceLocked says how long each side has been quiet; the caller holds mu.
func (b *Observer) silenceLocked(now time.Time) string {
	ago := func(who string) string {
		t, said := b.lastSaid[who]
		if !said {
			return "nothing yet"
		}
		return fmt.Sprintf("%.0f s ago", now.Sub(t).Seconds())
	}
	if len(b.cfg.Cast) > 0 {
		var each []string
		for _, seat := range b.cfg.Cast {
			each = append(each, seat+" last spoke "+ago(seat))
		}
		each = append(each, "the player "+ago(SpeakerPlayer))
		return fmt.Sprintf("Nobody has spoken for %.0f s. %s.", now.Sub(b.lastAny).Seconds(), strings.Join(each, ", "))
	}
	return fmt.Sprintf("Nobody has spoken for %.0f s. The character last spoke %s, the player %s.",
		now.Sub(b.lastAny).Seconds(), ago(SpeakerCharacter), ago(SpeakerPlayer))
}

// note puts an entry on the log, and, for the model, among the lines waiting.
func (b *Observer) note(e ObserverEntry, forModel bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.log = append(b.log, e)
	if len(b.log) > logKept {
		b.log = b.log[len(b.log)-logKept:]
	}
	// Standing down, nothing waits for the model; the log keeps it as history.
	if forModel && b.cfg.Tool != nil && !b.standingDown {
		b.pending = append(b.pending, e)
	}
}

// think calls the model with what waits, unless a call is in flight already.
func (b *Observer) think(ctx context.Context, replies chan<- reply) {
	b.mu.Lock()
	if b.cfg.Tool == nil || !b.open || b.standingDown || !b.busySince.IsZero() || len(b.pending) == 0 ||
		(!b.called.IsZero() && time.Since(b.called) < b.cfg.Gap) {
		b.mu.Unlock()
		return
	}
	prompt := b.promptLocked()
	b.pending = nil
	b.busySince = time.Now()
	b.called = b.busySince
	b.figures.Calls++
	b.mu.Unlock()
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})

	go func() {
		started := time.Now()
		text, used, err := b.cfg.Tool.Query(ctx, b.cfg.Brief, prompt)
		if err == nil {
			b.meter.bill(used)
		}
		select {
		case replies <- reply{text: text, took: time.Since(started), err: err}:
		case <-ctx.Done():
		}
	}()
}

// promptLocked is the log so far, newest kept, the lines since the last call
// marked as new; the caller holds mu.
func (b *Observer) promptLocked() string {
	var older strings.Builder
	start := len(b.log) - len(b.pending)
	for _, e := range b.log[:max(start, 0)] {
		older.WriteString(render(e))
	}
	history := older.String()
	if len(history) > logLimit {
		history = "…" + history[len(history)-logLimit:]
	}
	var fresh strings.Builder
	for _, e := range b.pending {
		fresh.WriteString(render(e))
	}
	return "So far:\n" + history + "\nNew since your last reply:\n" + fresh.String()
}

// render is one log entry as the model reads it.
func render(e ObserverEntry) string {
	clock := e.At.Format("15:04:05")
	switch e.Kind {
	case EntryLine:
		late := ""
		if e.Late {
			late = " (arrived late; said earlier)"
		}
		return fmt.Sprintf("%s %s%s: %s\n", clock, e.Speaker, late, e.Text)
	case EntrySilence:
		return fmt.Sprintf("%s [silence] %s\n", clock, e.Text)
	case EntryInjection:
		return fmt.Sprintf("%s [injected by the player, %s] %s\n", clock, e.Speaker, e.Text)
	case EntryThought:
		return fmt.Sprintf("%s [you] %s\n", clock, e.Text)
	case EntryDirection, EntryBackground, EntrySpeech:
		if e.Speaker != "" {
			return fmt.Sprintf("%s [sent, %s to %s] %s\n", clock, e.Kind, e.Speaker, e.Text)
		}
		return fmt.Sprintf("%s [sent, %s] %s\n", clock, e.Kind, e.Text)
	case EntryDropped:
		return fmt.Sprintf("%s [not sent] %s\n", clock, e.Text)
	case EntryRules:
		return fmt.Sprintf("%s [sent: the character was reminded of the rules]\n", clock)
	case EntryHeld:
		return fmt.Sprintf("%s [held until the conversation pauses] %s\n", clock, e.Text)
	case EntryAbsent:
		return fmt.Sprintf("%s [stood down] %s\n", clock, e.Text)
	}
	return ""
}

// answered takes a reply: its thinking onto the log, its marked lines into
// the conversation.
func (b *Observer) answered(r reply) {
	now := time.Now()
	b.mu.Lock()
	b.busySince = time.Time{}
	b.figures.LastCallMs = r.took.Milliseconds()
	if r.err != nil {
		b.figures.Errors++
	}
	b.mu.Unlock()
	defer b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

	if r.err != nil {
		b.note(ObserverEntry{At: now, Kind: EntryError, Text: r.err.Error()}, false)
		b.journal.Warn(b.name, "call-failed", r.err.Error())
		return
	}
	if strings.TrimSpace(r.text) == Quiet || strings.TrimSpace(r.text) == "" {
		b.mu.Lock()
		b.figures.Quiet++
		b.mu.Unlock()
		return
	}
	b.note(ObserverEntry{At: now, Kind: EntryThought, Text: strings.TrimSpace(r.text)}, false)
	b.mu.Lock()
	open := b.open && !b.standingDown
	b.mu.Unlock()
	if !open {
		// The conversation closed, or the player went quiet, while it thought:
		// nothing to send into, or no one to nudge for.
		for _, line := range []string{marked(r.text, MarkDirection), marked(r.text, MarkBackground)} {
			if line != "" {
				b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: line}, false)
			}
		}
		for _, o := range addressed(r.text, b.cfg.Cast) {
			b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: o.text}, false)
		}
		return
	}

	// The rules at risk come first, and in their own words.
	if hasMark(r.text, MarkRules) {
		b.remind(now)
	}
	if len(b.cfg.Cast) > 0 {
		b.answeredCast(r.text, now)
		return
	}

	direction, background := marked(r.text, MarkDirection), marked(r.text, MarkBackground)
	// A direction interrupts the character and stays for the whole session,
	// so it is rare: not before the character has answered the last one, not
	// the same again, not too many in a row; and it waits for a pause.
	why := ""
	b.mu.Lock()
	switch {
	case direction == "":
		b.inARow = 0
	case b.awaiting:
		why = "the character has not answered the last direction yet"
	case sameIntent(direction, b.lastDirection) || sameIntent(direction, b.held):
		why = "the same as the last direction"
	case b.inARow >= b.cfg.InARow:
		why = "too many directions in a row"
	default:
		b.inARow++
		b.held = direction
	}
	if why != "" {
		b.figures.Dropped++
	}
	b.mu.Unlock()

	switch {
	case why != "":
		b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: why + ": " + direction}, false)
		b.journal.Note(b.name, "direction-dropped", why+": "+direction)
	case direction != "":
		b.note(ObserverEntry{At: now, Kind: EntryHeld, Text: direction}, false)
		b.release(now)
	}
	// Background does not request speech, so it goes in at once.
	if background != "" {
		b.send(EntryBackground, ports.InjectBackground, background, "", now)
	}
}

// release sends a held direction once both sides have paused, and notes when
// the character has finished an answer since the last one.
func (b *Observer) release(now time.Time) {
	b.mu.Lock()
	character, player := b.characterSaidLocked(), b.lastSaid[SpeakerPlayer]
	quiet := func(t time.Time) bool { return t.IsZero() || now.Sub(t) >= b.cfg.Pause }
	if b.awaiting && character.After(b.directedAt) && quiet(character) {
		b.awaiting = false
	}
	direction := b.held
	if direction == "" || !b.open || !quiet(character) || !quiet(player) {
		b.mu.Unlock()
		return
	}
	to := b.heldTo
	b.held, b.heldTo = "", ""
	b.lastDirection = direction
	// A cast has no one answer to wait for: the characters answer each other.
	b.awaiting = len(b.cfg.Cast) == 0
	b.directedAt = now
	b.mu.Unlock()
	b.send(EntryDirection, ports.InjectDirection, direction, to, now)
}

// characterSaidLocked is when a character last spoke: the one, or the
// latest of a cast. The caller holds mu.
func (b *Observer) characterSaidLocked() time.Time {
	if len(b.cfg.Cast) == 0 {
		return b.lastSaid[SpeakerCharacter]
	}
	var latest time.Time
	for _, seat := range b.cfg.Cast {
		if t := b.lastSaid[seat]; t.After(latest) {
			latest = t
		}
	}
	return latest
}

// order is one marked line of a reply to a cast: what kind, for which seat
// (empty: all), and the text.
type order struct {
	kind ports.InjectionKind
	to   string
	text string
}

// addressed reads the marked lines of a reply to a cast, "[KIND SEAT] text",
// in order; a line naming no seat, or BOTH, is for all, and one naming a
// seat not in the cast is skipped.
func addressed(reply string, cast []string) []order {
	if len(cast) == 0 {
		return nil
	}
	kinds := map[string]ports.InjectionKind{
		"DIRECTION": ports.InjectDirection, "BACKGROUND": ports.InjectBackground, "SPEECH": ports.InjectSpeech,
	}
	var out []order
	for _, line := range strings.Split(reply, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		end := strings.Index(line, "]")
		if end < 0 {
			continue
		}
		fields := strings.Fields(strings.ToUpper(line[1:end]))
		if len(fields) == 0 {
			continue
		}
		kind, ok := kinds[fields[0]]
		if !ok {
			continue
		}
		to := ""
		if len(fields) > 1 && fields[1] != SeatAll {
			to = fields[1]
			if !slices.Contains(cast, to) {
				continue
			}
		}
		if text := strings.TrimSpace(line[end+1:]); text != "" {
			out = append(out, order{kind: kind, to: to, text: text})
		}
	}
	return out
}

// answeredCast puts a reply to a cast into the conversations: background at
// once to whom it names, at most one line to say, and a direction held for a
// pause, as rare as with one character.
func (b *Observer) answeredCast(reply string, now time.Time) {
	directed, spoke := false, false
	for _, o := range addressed(reply, b.cfg.Cast) {
		switch o.kind {
		case ports.InjectBackground:
			b.send(EntryBackground, ports.InjectBackground, o.text, o.to, now)
		case ports.InjectSpeech:
			// Two characters told to speak at once talk over each other.
			if spoke || o.to == "" {
				b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: "only one character speaks at a time: " + o.text}, false)
				continue
			}
			spoke = true
			b.send(EntrySpeech, ports.InjectSpeech, o.text, o.to, now)
		case ports.InjectDirection:
			why := ""
			b.mu.Lock()
			switch {
			case directed:
				why = "one direction per reply"
			case sameIntent(o.text, b.lastDirection) || sameIntent(o.text, b.held):
				why = "the same as the last direction"
			case b.inARow >= b.cfg.InARow:
				why = "too many directions in a row"
			default:
				directed = true
				b.inARow++
				b.held, b.heldTo = o.text, o.to
			}
			if why != "" {
				b.figures.Dropped++
			}
			b.mu.Unlock()
			if why != "" {
				b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: why + ": " + o.text}, false)
				continue
			}
			b.note(ObserverEntry{At: now, Kind: EntryHeld, Speaker: o.to, Text: o.text}, false)
		}
	}
	if !directed {
		b.mu.Lock()
		b.inARow = 0
		b.mu.Unlock()
	}
	b.release(now)
}

// playerUnfinishedLocked says the player is still mid-sentence among what
// would leave the pool: they spoke within the pause, and their words so far
// end no sentence. The caller holds mu.
func (b *Observer) playerUnfinishedLocked(out []ports.Fragment, now time.Time) bool {
	var words strings.Builder
	for _, f := range out {
		if f.Speaker == SpeakerPlayer {
			words.WriteString(f.Text)
		}
	}
	if words.Len() == 0 {
		return false
	}
	last := b.lastSaid[SpeakerPlayer]
	return now.Sub(last) < b.cfg.Pause && !endsSentence(words.String())
}

// sameIntent says two directions say much the same: most of their words shared.
func sameIntent(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	words := func(s string) map[string]bool {
		set := map[string]bool{}
		for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			if utf8.RuneCountInString(w) > 2 {
				set[w] = true
			}
		}
		return set
	}
	x, y := words(a), words(b)
	if len(x) == 0 || len(y) == 0 {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	shared := 0
	for w := range x {
		if y[w] {
			shared++
		}
	}
	return float64(shared) >= 0.6*float64(min(len(x), len(y)))
}

// send puts one of the model's injections into the conversation.
func (b *Observer) send(entry string, kind ports.InjectionKind, text, to string, at time.Time) {
	if utf8.RuneCountInString(text) > maxInjection {
		text = string([]rune(text)[:maxInjection])
	}
	b.out.Send(ports.Injection{Kind: kind, Text: text, To: to})
	b.note(ObserverEntry{At: at, Kind: entry, Speaker: to, Text: text}, false)
	b.mu.Lock()
	if kind == ports.InjectDirection {
		b.figures.Directions++
	} else {
		b.figures.Backgrounds++
	}
	b.mu.Unlock()
	b.journal.Note(b.name, entry, text)
}

// marked is the text after the first line of a reply that starts with mark.
func marked(reply, mark string) string {
	for _, line := range strings.Split(reply, "\n") {
		line = strings.TrimSpace(line)
		if rest, found := strings.CutPrefix(line, mark); found {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// remind sends the rules to the character as its own knowledge, which does not
// interrupt it; at most once a RulesGap, so it cannot be flooded.
func (b *Observer) remind(now time.Time) {
	b.mu.Lock()
	rules := strings.TrimSpace(b.cfg.Rules)
	soon := !b.reminded.IsZero() && now.Sub(b.reminded) < b.cfg.RulesGap
	if rules != "" && !soon {
		b.reminded = now
		b.figures.Reminders++
	} else if rules != "" {
		b.figures.Dropped++
	}
	b.mu.Unlock()
	switch {
	case rules == "":
		return
	case soon:
		b.note(ObserverEntry{At: now, Kind: EntryDropped, Text: "the rules were reminded of shortly before"}, false)
		return
	}
	text := strings.TrimSpace(b.cfg.RulesPreface + "\n" + rules)
	if utf8.RuneCountInString(text) > maxInjection {
		text = string([]rune(text)[:maxInjection])
	}
	b.out.Send(ports.Injection{Kind: ports.InjectBackground, Text: text})
	b.note(ObserverEntry{At: now, Kind: EntryRules, Text: text}, false)
	b.journal.Warn(b.name, "rules-reminded", "the rules were at risk; the character was reminded of them")
}

// hasMark says a line of the reply starts with mark.
func hasMark(reply, mark string) bool {
	for _, line := range strings.Split(reply, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), mark) {
			return true
		}
	}
	return false
}
