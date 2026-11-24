// Package engine is the module's whole public surface; it owns no HTTP, so it runs standalone or embedded.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/blocks"
	"engine/internal/genre"
	"engine/internal/ports"
)

// KeyFunc yields the API key for a session's platform, injected by whoever embeds the engine.
type KeyFunc func(ctx context.Context) (string, error)

// Tier is re-exported so a caller can name one without importing internal/.
type Tier = adapters.Tier

const (
	TierLow    = adapters.TierLow
	TierMedium = adapters.TierMedium
	TierHigh   = adapters.TierHigh
)

// Platform selects which implementations back the adapter roles.
type Platform string

const (
	PlatformMock    Platform = "mock"
	PlatformOpenAI  Platform = "openai"
	PlatformMistral Platform = "mistral"
)

type Genre string

const (
	GenreAdventure Genre = "adventure"
	GenreNPCLive   Genre = "npc-live"
	// GenreNPCDuo is two live characters talking to each other, with the
	// player free to join in.
	GenreNPCDuo Genre = "npc-duo"
)

// SessionSpec is everything a session needs, fully resolved so the engine never calls back into the platform.
type SessionSpec struct {
	Genre Genre `json:"genre"`

	ID string `json:"id,omitempty"`

	// Title is handed back on the topology for a client's header.
	Title string `json:"title,omitempty"`

	Persist Persist `json:"-"`

	// Guardrail is the platform's resolved youth-protection rule.
	Guardrail string `json:"guardrail,omitempty"`

	// GuardrailSource is where the guardrail came from, for a reader: the
	// level of the platform's fallback chain that set it. Nil when unknown.
	GuardrailSource *GuardrailSource `json:"guardrailSource,omitempty"`

	// GuardrailEdit is the guardrail as staff or a head changed it during
	// play, kept to from then on in place of Guardrail; nil keeps Guardrail.
	GuardrailEdit *string `json:"guardrailEdit,omitempty"`

	// Scenario is the designer's text, kept apart from Guardrail so an author has no lever on youth protection.
	Scenario string `json:"scenario,omitempty"`

	// ImageStyle is how pictures are painted; empty takes the genre's default so the style stays stable.
	ImageStyle string `json:"imageStyle,omitempty"`

	// VoiceStyle is how the game's voice sounds, in the designer's words: the
	// narrator's when an Adventure is spoken, the character's in NPC-Live.
	// Empty leaves it to the voice.
	VoiceStyle string `json:"voiceStyle,omitempty"`

	// GalleryImage is the game's picture from the gallery, for a genre that
	// shows it. Never stored: a resumed session already holds it in its history.
	GalleryImage []byte `json:"-"`

	// Theme names the playback-engine's look, one of Themes(); empty is the default look.
	Theme string `json:"theme,omitempty"`

	// Language is the player's, fixed at creation, as a BCP 47 tag such as "de".
	Language string `json:"language,omitempty"`

	// InitPrompt is the game's starting message. NPC-Live's gate sends it once
	// preparation is done, so the character speaks first; Adventure's System
	// Message carries it as how the game starts.
	InitPrompt string `json:"initPrompt,omitempty"`

	// Description is the game's idea in a sentence, which Adventure's picture
	// prompts carry.
	Description string `json:"description,omitempty"`

	// Dice are Adventure's random values, each 1 to 100, rolled once at launch
	// and stored, so a resumed opening is the same one.
	Dice []int `json:"dice,omitempty"`

	// ConstraintFrequency sets, by block name, how often a threaded block's
	// calls carry the youth-protection rule: 1, the default, is every call, up
	// to 10. A change during play is stored here.
	ConstraintFrequency map[string]int `json:"constraintFrequency,omitempty"`

	// Preference is the player's stored wish, fitted to the offers at launch.
	// Without one the player chooses, unless a single fixed offer leaves nothing
	// to choose. The offers themselves are no part of the spec: they are live,
	// rebuilt from the platform's key cascade on every launch and resume.
	Preference *Preference `json:"preference,omitempty"`

	// Voice is a provider built-in voice name, fixed once the session exists:
	// one the genre's speaking model knows, gpt-realtime's for Adventure's
	// narration and GPT-Live's for NPC-Live. Empty takes the tier's.
	Voice string `json:"voice,omitempty"`

	// PromptOverride replaces a genre prompt by name; an unknown name errors so a typo cannot pass silently.
	PromptOverride map[string]string `json:"promptOverride,omitempty"`

	// Status seeds the property values a genre tracks.
	Status map[string]string `json:"status,omitempty"`

	// IdleSeconds of silence pause a live conversation, which bills per second. Zero: default; negative: never.
	IdleSeconds int `json:"idleSeconds,omitempty"`

	// MockPaceMs slows the mock platform down so a person can watch a turn assemble.
	MockPaceMs int `json:"mockPaceMs,omitempty"`

	// Script replaces typed player input with canned utterances; live audio bypasses the engine.
	Script []string `json:"script,omitempty"`
	// ScriptIntervalMs is an int because a time.Duration would read a JSON number as nanoseconds.
	ScriptIntervalMs int `json:"scriptIntervalMs,omitempty"`
}

// GuardrailSource is the level of the platform's fallback chain a guardrail
// came from: Level a code such as "workshop", "organisation" or "site13",
// Name the workshop's or organisation's name, empty for the site's.
type GuardrailSource struct {
	Level string `json:"level"`
	Name  string `json:"name,omitempty"`
}

// guardrail is the guardrail the session keeps to: an edit, else the
// resolved one.
func (spec SessionSpec) guardrail() string {
	if spec.GuardrailEdit != nil {
		return *spec.GuardrailEdit
	}
	return spec.Guardrail
}

// SessionEvent is one item on the session-scoped stream; Stream names the output block it came from.
type SessionEvent struct {
	Stream string
	Value  string
}

// BlockState mirrors ports.State on the wire, keeping the public surface free of internals.
type BlockState struct {
	Node  string `json:"node"`
	Phase string `json:"phase"`
}

// StreamHeartbeat is what a page's socket sends now and then, so the page
// can tell a quiet session from a dead socket. It carries nothing.
const StreamHeartbeat = "heartbeat"

// StreamNames is every stream a genre can wire, and the source of the generated TypeScript union.
func StreamNames() []string {
	return []string{"text", "text-b", "cast", "said", "audio", "image", "props", "state", "usage", "connect", "pause", "settled", "error", "problem", "journal", StreamOver, StreamHeartbeat}
}

type Session struct {
	// mu guards what settling, adapting and toggling change: the spec's offers
	// and preference, and the settlement.
	mu sync.Mutex
	// settled is nil while the player is choosing.
	settled *settlement
	// over is the game's end, nil while it goes on; guarded by mu.
	over *Over
	// offers are the live offers, never stored: passed at launch, replaced by Adapt.
	offers []Offer
	// changes say why the settlement differs from the preference.
	changes []string
	// toldRejected says the slots have told the player their key was rejected.
	toldRejected bool
	// ended says a problem sink has ended the session, so it ends once.
	ended sync.Once
	// pending holds events posted under mu, until flush hands them to the sender.
	pending []SessionEvent
	// out is the sender's queue; the sender alone writes to the stream.
	out outbox

	spec  SessionSpec
	slots slots
	// kit is the adapters in the slots, kept to switch the story thread's
	// model and to close them; guarded by mu.
	kit      equipment
	wiring   *genre.Wiring
	usage    *usageLedger
	recorder *recorder
	// journal is the engine's record of what happened, for debugging and Insights.
	journal *ports.Journal
	// journalKeep holds entries until keepJournal stores them; journalKept
	// closes once the last is stored.
	journalKeep chan SessionEvent
	journalKept chan struct{}
	// hub hands the session's events to whoever subscribes; see Subscribe.
	hub    *hub
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once

	// logMu keeps the stored history in the order the recorder holds it.
	logMu sync.Mutex
	// saveMu lets one event at a time into the store: a platform numbers a
	// session's events by reading the last number, so two writes at once would
	// take the same one and lose an event.
	saveMu sync.Mutex
	// dirty asks the keeper for a checkpoint; kept closes once its last one is written.
	dirty chan struct{}
	kept  chan struct{}
}

// slots hold the adapters the blocks call, filled when the session settles
// and replaced when it adapts.
type slots struct {
	voices  adapters.VoicesSlot
	live    adapters.LiveSlot
	tool    adapters.ToolSlot
	image   adapters.ImageSlot
	thread  adapters.ThreadSlot
	hearing adapters.HearingSlot
}

// SessionState is what changes every turn and must survive a restart: what
// each block exports, keyed by block name, and what the session has spent.
type SessionState struct {
	Blocks map[string]string `json:"blocks,omitempty"`
	Usage  []UsageRecord     `json:"usage,omitempty"`
	// Over is the game's end; a game that is over is read back, never played on.
	Over *Over `json:"over,omitempty"`
}

// Resume continues a stored session on the offers at hand, fitting its stored
// preference to them. It refuses with ErrStale a session stored under another
// SchemaVersion, or one whose blocks the wiring no longer has, and with
// ErrUnknown one the store does not hold.
//
// The context is the session's lifetime, as for Launch.
//
// The gallery image is handed in again, like the offers: it is never stored,
// and a session resumed before it was shown shows it then.
func Resume(ctx context.Context, id string, persist Persist, offers []Offer, galleryImage []byte) (*Session, error) {
	stored, err := persist.LoadState(ctx, id)
	if err != nil {
		return nil, err
	}
	if stored.Version != SchemaVersion {
		return nil, &Refusal{Problem: ProblemStale,
			Message: fmt.Sprintf("the session was stored in version %d, and the engine reads %d", stored.Version, SchemaVersion)}
	}
	history, err := persist.LoadHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	spec := stored.Spec
	spec.ID = id
	spec.Persist = persist
	spec.GalleryImage = galleryImage
	return launch(ctx, spec, offers, &resumption{state: stored.State, history: history})
}

// resumption is what a resumed session starts from.
type resumption struct {
	state   SessionState
	history []SessionEvent
}

// Launch validates a spec and builds its genre's wiring. The game starts at
// once when there is nothing to choose or the spec carries a preference;
// otherwise it waits for the player's choice, and nothing runs or bills before.
//
// The offers are live, like the key functions they carry: the platform builds
// them from its key cascade for each launch, and they are never stored.
//
// The context is the session's lifetime: a launch from a request handler
// passes one that outlives the request, such as context.WithoutCancel's. The
// session's first state is saved before Launch returns, and a failed save
// refuses the launch.
func Launch(ctx context.Context, spec SessionSpec, offers []Offer) (*Session, error) {
	if spec.Genre == GenreAdventure && len(spec.Dice) == 0 {
		spec.Dice = rollDice()
	}
	return launch(ctx, spec, offers, nil)
}

// diceCount is how many values Adventure's opening rolls.
const diceCount = 5

// rollDice rolls the opening's values, each 1 to 100.
func rollDice() []int {
	values := make([]int, diceCount)
	for i := range values {
		values[i] = rand.IntN(100) + 1
	}
	return values
}

func launch(ctx context.Context, spec SessionSpec, offers []Offer, from *resumption) (*Session, error) {
	if err := refusal(Validate(spec, offers)); err != nil {
		return nil, err
	}

	s := &Session{
		recorder: newRecorder(),
		hub:      newHub(time.Now()),
		usage:    newUsageLedger(),
	}
	s.openJournal()

	w, err := s.wire(spec)
	if err != nil {
		return nil, err
	}

	if err := w.Graph.Validate(); err != nil {
		return nil, err
	}
	w.Graph.UseJournal(s.journal)
	s.watchKeys()

	// Restored before anything starts, so the first turn runs on the continued thread.
	if from != nil {
		if err := w.Graph.RestoreState(from.state.Blocks); err != nil {
			return nil, &Refusal{Problem: ProblemStale, Message: err.Error()}
		}
		s.usage.restore(from.state.Usage)
		s.recorder.restore(from.history)
		s.restoreJournal(from.history)
		if conversed(from.history) || (w.Live != nil && w.Live.Conversed()) || (w.Partner != nil && w.Partner.Conversed()) {
			w.Gate.Reopen()
		}
	}

	if spec.Persist == nil {
		spec.Persist = NoopPersist{}
	}
	if spec.ID == "" {
		spec.ID = "dev-session"
	}
	s.spec = spec
	s.offers = offers
	s.wiring = w
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.out.ready = make(chan struct{}, 1)
	s.dirty = make(chan struct{}, 1)
	s.kept = make(chan struct{})

	// Saved before anything runs, so a session the platform has created can
	// always be resumed.
	if err := s.saveState(); err != nil {
		s.cancel()
		return nil, fmt.Errorf("store the session: %w", err)
	}
	if from != nil {
		s.journal.Note(JournalSession, "resumed", fmt.Sprintf("genre %s, %d history events", spec.Genre, len(s.recorder.replay())))
	} else {
		s.journal.Note(JournalSession, "launched", "genre "+string(spec.Genre))
	}
	go s.keep()
	go s.keepJournal()
	go s.deliver()
	go s.release()

	// Subscribed before Start, or the opening phases and the one invitation would be missed.
	states := w.Graph.ObserveStates(s.ctx)
	usage := w.Graph.ObserveUsage(s.ctx)

	// A game that is over is read back: its history and its state, and nothing runs.
	if from != nil && from.state.Over != nil {
		s.mu.Lock()
		s.over = from.state.Over
		s.mu.Unlock()
		return s, nil
	}

	var invitations, pauses, stops <-chan ports.State
	if w.Live != nil {
		invitations = w.Live.Invitations()
		pauses = w.Live.Pauses()
		stops = w.Live.Stops()
	}
	if w.Partner != nil {
		// Two conversations are one scene to the player: either inviting
		// has the browser connect both, either let go lets the other go too,
		// and either stopping ends the game.
		pauses = s.pairPauses(s.ctx, w.Live, w.Partner)
		invitations = mergeStates(s.ctx, invitations, w.Partner.Invitations())
		stops = mergeStates(s.ctx, stops, w.Partner.Stops())
	}

	s.merge(s.ctx)
	s.forwardStates(s.ctx, states)
	s.forwardUsage(s.ctx, usage)
	s.forwardInvitations(s.ctx, invitations)
	s.forwardSignal(s.ctx, pauses, "pause")
	s.forwardStops(stops)

	if settled, changes, choosing := spec.opening(offers); !choosing {
		s.mu.Lock()
		s.start(settled, changes)
		s.mu.Unlock()
		s.flush()
	}
	return s, nil
}

// merge folds every wired sink into one session-scoped stream. What reaches
// a problem sink ends the session instead.
func (s *Session) merge(ctx context.Context) {
	for _, sink := range s.wiring.Sinks() {
		go func(name string, ch chan string) {
			noted := &sinkJournal{s: s, stream: name}
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-ch:
					if !ok {
						return
					}
					if name == "problem" {
						s.journal.Fail(JournalSession, "problem", v)
						s.failed(v)
						continue
					}
					noted.note(v)
					ev := SessionEvent{Stream: name, Value: v}
					s.log(ev)
					if name == "props" {
						if props := readProps(v); props != nil {
							s.recorder.recordProps(props)
						}
					}
					s.hub.publish(ev)
				}
			}
		}(sink.Stream(), sink.Arrivals())
	}
}

// forwardStates puts every block's own report on the session stream, for debugging and the graph view.
func (s *Session) forwardStates(ctx context.Context, states <-chan ports.State) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case state, alive := <-states:
				if !alive {
					return
				}
				s.recorder.recordPhase(state.Node, string(state.Phase))
				if state.Failed() {
					s.journal.Fail(state.Node, "call-failed", state.Reason+": "+state.Message)
				}
				s.journal.Note(state.Node, "phase", string(state.Phase))

				blob, err := json.Marshal(BlockState{Node: state.Node, Phase: string(state.Phase)})
				if err != nil {
					continue
				}
				ev := SessionEvent{Stream: "state", Value: string(blob)}
				if state.Phase == ports.PhaseReady {
					s.checkpoint()
				}
				s.hub.publish(ev)
			}
		}
	}()
}

// forwardUsage sends the whole priced report on each change; the price table lives in the engine.
func (s *Session) forwardUsage(ctx context.Context, usage <-chan ports.Usage) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case used, alive := <-usage:
				if !alive {
					return
				}
				s.usage.record(used)

				blob, err := json.Marshal(s.usage.report())
				if err != nil {
					continue
				}
				ev := SessionEvent{Stream: "usage", Value: string(blob)}
				s.checkpoint()
				s.hub.publish(ev)
			}
		}
	}()
}

// forwardInvitations says a player may connect now, so the game has begun before anyone is asked to speak.
// Its value is the invitation's reason: empty to connect at once, "press" to
// wait for the player.
func (s *Session) forwardInvitations(ctx context.Context, invitations <-chan ports.State) {
	if invitations == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case invitation, alive := <-invitations:
				if !alive {
					return
				}
				s.journal.Note(JournalSession, "connect", invitation.Reason)
				s.hub.publish(SessionEvent{Stream: "connect", Value: invitation.Reason})
			}
		}
	}()
}

// forwardSignal puts an empty lifecycle event on the stream; the moment is the whole message.
func (s *Session) forwardSignal(ctx context.Context, signal <-chan ports.State, stream string) {
	if signal == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, alive := <-signal:
				if !alive {
					return
				}
				s.journal.Note(JournalSession, stream, "")
				s.hub.publish(SessionEvent{Stream: stream})
			}
		}
	}()
}

// Usage is what the session has spent so far, per block and per model.
func (s *Session) Usage() UsageReport { return s.usage.report() }

// Prompts are the prompts the session runs on now: the genre's own, after any
// the spec replaced or a person edited during play.
func (s *Session) Prompts() map[string]string { return s.wiring.CurrentPrompts() }

// GuardrailReport is the session's guardrail for a reader: the one the
// blocks keep to now and the one the platform resolved, where it came from,
// the blocks that keep to it and how often each threaded block repeats it.
// MayEdit says whether the reader may change it; whoever serves the engine
// decides, and a server without roles lets everyone.
type GuardrailReport struct {
	Text        string           `json:"text"`
	Resolved    string           `json:"resolved"`
	Edited      bool             `json:"edited"`
	Source      *GuardrailSource `json:"source,omitempty"`
	ReadBy      []string         `json:"readBy"`
	Frequencies []BlockFrequency `json:"frequencies"`
	MayEdit     bool             `json:"mayEdit"`
}

// BlockFrequency is how often a threaded block's calls repeat the
// guardrail: 1 every call, up to 10 every tenth.
type BlockFrequency struct {
	Block     string `json:"block"`
	Frequency int    `json:"frequency"`
}

// Guardrail reports the session's guardrail.
func (s *Session) Guardrail() GuardrailReport {
	s.mu.Lock()
	resolved, source := s.spec.Guardrail, s.spec.GuardrailSource
	s.mu.Unlock()
	text := s.wiring.CurrentGuardrail()
	report := GuardrailReport{
		Text: text, Resolved: resolved, Edited: text != resolved, Source: source,
		ReadBy: slices.Clone(s.wiring.GuardrailUse.Blocks), Frequencies: []BlockFrequency{}, MayEdit: true,
	}
	for _, block := range slices.Sorted(maps.Keys(s.wiring.Constrained)) {
		report.Frequencies = append(report.Frequencies, BlockFrequency{Block: block, Frequency: s.wiring.Constrained[block].ConstraintFrequency()})
	}
	return report
}

// Connect brokers a browser's offer to talk to the provider directly; errors on a genre with no conversation.
// The seat names which conversation, where a genre holds two: empty is the
// first, SeatPartner the second.
func (s *Session) Connect(ctx context.Context, offer, seat string) (string, error) {
	if err := s.playing(); err != nil {
		return "", err
	}
	live := s.wiring.Live
	switch seat {
	case "":
	case genre.SeatPartner:
		live = s.wiring.Partner
	default:
		return "", fmt.Errorf("no seat %q", seat)
	}
	if live == nil {
		return "", fmt.Errorf("genre %q holds no live conversation at seat %q", s.spec.Genre, seat)
	}
	s.journal.Note(JournalClient, "connect", "the browser offers to open the audio connection "+seat)
	answer, err := live.Connect(ctx, offer)
	if err != nil {
		s.journal.Fail(JournalClient, "connect-failed", err.Error())
	}
	return answer, err
}

// Snapshot is where the session stands now, for a reloaded or late-opened page.
func (s *Session) Snapshot() Snapshot {
	started := false
	select {
	case <-s.wiring.Gate.Ready():
		started = true
	default:
	}
	over := s.overNow()
	if over != nil {
		// It was played, whether or not this run of it started anything.
		started = true
	}
	snapshot := s.recorder.snapshot(started, s.usage.report(), s.conversation())
	snapshot.Over = over
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot.Choosing = s.settled == nil && over == nil
	snapshot.Settled = s.settledView()
	return snapshot
}

// conversation is where the live block stands, or empty without one.
func (s *Session) conversation() string {
	if s.wiring.Live == nil {
		return ""
	}
	return s.wiring.Live.Standing()
}

// The wiring as data, as a transport answers it: the drawable graph, and what
// one wire or block has carried.
type (
	Topology     = ports.Topology
	TopologyNode = ports.TopologyNode
	TopologyEdge = ports.TopologyEdge
	EdgeDetail   = ports.EdgeDetail
	Sample       = ports.Sample
)

// NodeDetail is everything the engine knows about one block.
type NodeDetail struct {
	ports.NodeDetail
	Phase string       `json:"phase"`
	Usage *UsageRecord `json:"usage,omitempty"`
	// Thread is how full the block's thread is, for a block on a thread; such
	// a block can compact it on demand.
	Thread *ThreadFill `json:"thread,omitempty"`
	// Injectable says the block takes an injection during play; Injections
	// are the kinds it takes.
	Injectable bool     `json:"injectable,omitempty"`
	Injections []string `json:"injections,omitempty"`
	// Transcript is a conversation's so far, both speakers, oldest first, for
	// the block that holds one.
	Transcript []TranscriptLine `json:"transcript,omitempty"`
	// Observer is where a block following a conversation from the side
	// stands: its pool, its timers, its figures and its log.
	Observer *ObserverState `json:"observer,omitempty"`
	// Prompts are the genre prompts the block runs on, each editable during
	// play; a block on a thread runs on the thread's too.
	Prompts []BlockPrompt `json:"prompts,omitempty"`
}

// BlockPrompt is one genre prompt as a block runs on it: its name, its text
// now, the genre's own text, and whether the two differ.
type BlockPrompt struct {
	Name    string `json:"name"`
	Text    string `json:"text"`
	Default string `json:"default"`
	Edited  bool   `json:"edited"`
}

// ObserverState is where an observer stands, for a reader.
type ObserverState = blocks.ObserverState

// TranscriptLine is one turn of a conversation: who spoke, and what.
type TranscriptLine = blocks.TranscriptLine

// ThreadFill is how full a thread is: the input of its last call, in tokens,
// and the context window of the model it ran on, zero when unknown. Both are
// zero before a call, and after a compaction until the next one.
type ThreadFill = blocks.ThreadFill

// Inspect reports what is known about one block.
func (s *Session) Inspect(name string) (NodeDetail, bool) {
	base, found := s.wiring.Graph.Inspect(name)
	if !found {
		return NodeDetail{}, false
	}

	detail := NodeDetail{NodeDetail: base, Phase: "ready"}
	snapshot := s.recorder.snapshot(false, UsageReport{}, "")
	if phase, known := snapshot.Phases[name]; known {
		detail.Phase = phase
	}
	for _, record := range s.usage.report().ByNode {
		if record.Node == name {
			spent := record
			detail.Usage = &spent
			break
		}
	}
	if b, onThread := s.wiring.OnThread[name]; onThread {
		fill := b.Thread().Fill()
		detail.Thread = &fill
	}
	for _, live := range []*blocks.LiveSession{s.wiring.Live, s.wiring.Partner} {
		if live != nil && live.NodeName() == name {
			detail.Transcript = live.Transcript()
		}
	}
	if observer, ok := s.wiring.Observers[name]; ok {
		state := observer.State()
		detail.Observer = &state
	}
	if names := s.wiring.PromptsOf(name); len(names) > 0 {
		current := s.wiring.CurrentPrompts()
		for _, prompt := range names {
			detail.Prompts = append(detail.Prompts, BlockPrompt{
				Name: prompt, Text: current[prompt], Default: s.wiring.Defaults[prompt],
				Edited: current[prompt] != s.wiring.Defaults[prompt],
			})
		}
	}
	if _, ok := s.wiring.Injectable[name]; ok {
		detail.Injectable = true
		for _, kind := range ports.InjectionKinds {
			detail.Injections = append(detail.Injections, string(kind))
		}
	}
	return detail, true
}

// SetBeat changes how long an observer's fragments wait before they go to
// its model, during play. A block that is no observer is refused.
func (s *Session) SetBeat(block string, beat time.Duration) error {
	observer, ok := s.wiring.Observers[block]
	if !ok {
		return fmt.Errorf("no observer named %q", block)
	}
	if err := observer.SetBeat(beat); err != nil {
		return err
	}
	s.journal.Note(block, "beat", beat.String())
	return nil
}

// Compact asks a block's thread to compact now, between calls: it summarises
// the thread whatever its fill, the block working meanwhile and paying for
// the summary. A block on no thread is refused.
func (s *Session) Compact(block string) error {
	if err := s.playing(); err != nil {
		return err
	}
	return s.wiring.Compact(block)
}

// Inject passes an injection to a block that takes one, such as an observer,
// which puts it into the conversation it watches: a direction the model
// follows, background it may use, or a line it says. A block that takes none
// is refused, and so are an unknown kind and blank text.
func (s *Session) Inject(block, kind, text string) error {
	if err := s.playing(); err != nil {
		return err
	}
	return s.wiring.Inject(block, ports.InjectionKind(kind), text)
}

// InspectEdge reports what one wire has carried.
func (s *Session) InspectEdge(from, to, kind string) (ports.EdgeDetail, bool) {
	return s.wiring.Graph.InspectEdge(from, to, kind)
}

// History is the conversation so far, applied by a client through the same path as live events.
func (s *Session) History() []SessionEvent { return s.recorder.replay() }

// Events is the subscription opened with the session, so a caller in the
// same process sees every event from the start. The first page to subscribe
// ends it, as does falling a whole buffer behind.
func (s *Session) Events() <-chan SessionEvent { return s.hub.firstEvents() }

// Subscribe opens a page's own copy of the session's events, from now on;
// cancel ends it. A subscriber that falls a whole buffer behind is cut off,
// its channel closed, and rebuilds from the history once it reconnects.
func (s *Session) Subscribe() (<-chan SessionEvent, func()) { return s.hub.subscribe() }

// AloneSince is when the last page left the session; false while one is subscribed.
func (s *Session) AloneSince() (time.Time, bool) { return s.hub.aloneSince() }

// Say submits typed player input, erroring on a genre that takes none.
func (s *Session) Say(text string) error {
	if err := s.playing(); err != nil {
		return err
	}
	if s.wiring.Say == nil {
		return fmt.Errorf("genre %q takes no typed input", s.spec.Genre)
	}
	s.wiring.Say(text)
	return nil
}

// Speak submits player audio: on a turn-based genre one clip of PCM16 at
// 24 kHz mono is one utterance. It is refused while the genre cannot take it,
// such as while voice input is off.
func (s *Session) Speak(audio []byte) error {
	if err := s.playing(); err != nil {
		return err
	}
	if s.wiring.Speak == nil {
		return fmt.Errorf("genre %q takes no audio input", s.spec.Genre)
	}
	s.journal.Note(JournalClient, "speak", fmt.Sprintf("a clip of %d bytes", len(audio)))
	err := s.wiring.Speak(audio)
	if err != nil {
		s.journal.Warn(JournalClient, "speak-refused", err.Error())
	}
	return err
}

// Pause closes the per-second-billed live conversation; the session stays and a resume continues it.
func (s *Session) Pause() error {
	if err := s.playing(); err != nil {
		return err
	}
	if s.wiring.Live == nil {
		return fmt.Errorf("genre %q holds no live conversation", s.spec.Genre)
	}
	s.wiring.Live.Pause()
	if s.wiring.Partner != nil {
		s.wiring.Partner.Pause()
	}
	return nil
}

// pairPauses lets the other of two conversations go when one is let go, and
// reports each pause once: the scene pauses as one.
func (s *Session) pairPauses(ctx context.Context, first, second *blocks.LiveSession) <-chan ports.State {
	out := make(chan ports.State, 4)
	follow := func(from <-chan ports.State, other *blocks.LiveSession) {
		for {
			select {
			case <-ctx.Done():
				return
			case st, alive := <-from:
				if !alive {
					return
				}
				other.Pause()
				select {
				case out <- st:
				case <-ctx.Done():
					return
				}
			}
		}
	}
	go follow(first.Pauses(), second)
	go follow(second.Pauses(), first)
	return out
}

// mergeStates is two state streams as one, until both close.
func mergeStates(ctx context.Context, a, b <-chan ports.State) <-chan ports.State {
	out := make(chan ports.State, 4)
	var wg sync.WaitGroup
	pump := func(in <-chan ports.State) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case st, alive := <-in:
				if !alive {
					return
				}
				select {
				case out <- st:
				case <-ctx.Done():
					return
				}
			}
		}
	}
	for _, in := range []<-chan ports.State{a, b} {
		if in != nil {
			wg.Add(1)
			go pump(in)
		}
	}
	go func() { wg.Wait(); close(out) }()
	return out
}

// Ready closes when every block the first turn depends on has reported done.
func (s *Session) Ready() <-chan struct{} { return s.wiring.Gate.Ready() }

// Describe renders the running wiring, so a launcher can show the graph it got.
func (s *Session) Describe() string { return s.wiring.Graph.Describe() }

// Topology is the wiring in drawable form, with what a player may do.
func (s *Session) Topology() ports.Topology {
	t := s.wiring.Graph.Topology()
	t.Title = s.spec.Title
	t.Theme = s.spec.Theme
	t.Language = s.spec.Language
	t.Imagery = string(s.wiring.Imagery)
	if s.wiring.Partner != nil {
		t.Seats = []string{"", genre.SeatPartner}
	}
	t.Inputs = make([]string, 0, len(s.wiring.Inputs))
	for _, mode := range s.wiring.Inputs {
		t.Inputs = append(t.Inputs, string(mode))
	}
	t.Switches = make([]string, 0, len(s.wiring.Switches))
	for toggle, blocks := range s.wiring.Switches {
		if len(blocks) > 0 {
			t.Switches = append(t.Switches, string(toggle))
		}
	}
	slices.Sort(t.Switches)
	return t
}

// Mermaid is the wiring as a flowchart.
func (s *Session) Mermaid() string { return s.wiring.Graph.Mermaid() }

// ID is the session's ID, the handle that plays it.
func (s *Session) ID() string { return s.spec.ID }

// release closes the adapters once the session has ended.
func (s *Session) release() {
	<-s.ctx.Done()
	s.hub.close()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kit.close()
	s.kit = equipment{}
}

// Close ends the session once its last checkpoint is written.
func (s *Session) Close() {
	s.journal.Note(JournalSession, "closed", "")
	s.once.Do(s.cancel)
	<-s.kept
	<-s.journalKept
}

// readProps reads a props event's value, the status values as a JSON object;
// one that does not read is nil.
func readProps(value string) map[string]string {
	var props map[string]string
	if json.Unmarshal([]byte(value), &props) != nil {
		return nil
	}
	return props
}

// initPrompt's default asks the character to speak first, leaving what to say to the scenario.
func (spec SessionSpec) initPrompt() string {
	if spec.InitPrompt != "" {
		return spec.InitPrompt
	}
	return "Begin. Greet whoever has arrived, in character, in one or two sentences."
}

// defaultIdle exceeds the fifteen seconds billed up front per conversation, below which pausing costs more.
const defaultIdle = time.Minute

func (spec SessionSpec) idleAfter() time.Duration {
	switch {
	case spec.IdleSeconds < 0:
		return 0
	case spec.IdleSeconds == 0:
		return defaultIdle
	default:
		return time.Duration(spec.IdleSeconds) * time.Second
	}
}

// voice is what a speaking model speaks in: the spec's, or the tier's.
func (spec SessionSpec) voice(model adapters.Model) string {
	if spec.Voice != "" {
		return spec.Voice
	}
	return model.Params[adapters.ParamVoice]
}

// equipment is the adapters a settlement runs on, one per slot. The story
// thread has two: thread on the text story model, and speech on the speaking
// one, which it runs on while narration is on; speech is nil on a platform
// without narration.
type equipment struct {
	voices  adapters.Voices
	live    adapters.Live
	tool    adapters.Tool
	image   adapters.Image
	thread  adapters.Thread
	speech  adapters.Thread
	hearing adapters.Hearing
}

// story is the story thread's adapter for the narration toggle.
func (kit equipment) story(narrating bool) adapters.Thread {
	if narrating && kit.speech != nil {
		return kit.speech
	}
	return kit.thread
}

// close ends what the adapters hold open, such as a Realtime session.
func (kit equipment) close() {
	for _, held := range []any{kit.live, kit.tool, kit.image, kit.thread, kit.speech, kit.hearing} {
		if closer, ok := held.(io.Closer); ok {
			_ = closer.Close()
		}
	}
}

// adapters builds the settled offer's implementations from its tier's models,
// so a block never chooses its own model. A role the platform cannot fill
// comes back nil.
func (spec SessionSpec) adapters(settled settlement) (equipment, error) {
	entry, _ := catalogueEntry(settled.offer.Platform)
	models, err := entry.Models(settled.Tier)
	if err != nil {
		return equipment{}, err
	}

	story, speech := models[adapters.ModelStory], models[adapters.ModelSpeech]
	var kit equipment
	switch settled.offer.Platform {
	case PlatformMock:
		pace := time.Duration(spec.MockPaceMs) * time.Millisecond
		kit = equipment{
			voices:  &mock.Voices{Pace: pace},
			live:    mock.Live{Pace: pace},
			tool:    mock.Tool{Pace: pace},
			image:   mock.Image{Pace: pace},
			thread:  mock.Thread{Pace: pace, Name: story.Name},
			speech:  mock.Thread{Pace: pace, Name: speech.Name},
			hearing: mock.Hearing{Pace: pace},
		}
	case PlatformOpenAI:
		keys := adapters.KeyFunc(settled.offer.Keys)
		picture := models[adapters.ModelImage]
		kit = equipment{
			voices:  openai.NewVoices(keys),
			live:    openai.NewLive(keys, models[adapters.ModelLive].Name),
			tool:    openai.NewTool(keys, models[adapters.ModelTooling].Name),
			image:   openai.NewImage(keys, picture.Name, picture.Params[adapters.ParamQuality], picture.Params[adapters.ParamSize]),
			thread:  openai.NewThread(keys, story.Name),
			speech:  openai.NewRealtime(keys, speech.Name, spec.voice(speech)),
			hearing: openai.NewHearing(keys, models[adapters.ModelHearing].Name),
		}
	case PlatformMistral:
		// Live and image stay nil: the platform offers neither.
		keys := adapters.KeyFunc(settled.offer.Keys)
		kit = equipment{
			tool:    mistral.NewTool(keys, models[adapters.ModelTooling].Name),
			thread:  mistral.NewThread(keys, models[adapters.ModelStory].Name),
			hearing: mistral.NewHearing(keys, models[adapters.ModelHearing].Name),
		}
	default:
		return equipment{}, fmt.Errorf("unknown platform %q", settled.offer.Platform)
	}
	return kit, nil
}

// wire builds the spec's genre on the session's slots: its graph, unstarted.
func (s *Session) wire(spec SessionSpec) (*genre.Wiring, error) {
	switch spec.Genre {
	case GenreAdventure:
		return genre.NewAdventure(genre.AdventureConfig{
			Tool:                &s.slots.tool,
			Hearing:             &s.slots.hearing,
			Thread:              &s.slots.thread,
			Window:              s.usage.window,
			Image:               &s.slots.image,
			Scenario:            spec.Scenario,
			Guardrail:           spec.guardrail(),
			Description:         spec.Description,
			ImageStyle:          spec.ImageStyle,
			VoiceStyle:          spec.VoiceStyle,
			StartingMessage:     spec.InitPrompt,
			Status:              spec.Status,
			Language:            spec.Language,
			Dice:                spec.Dice,
			PromptOverride:      spec.PromptOverride,
			ConstraintFrequency: spec.ConstraintFrequency,
		})
	case GenreNPCLive:
		return genre.NewNPCLive(genre.NPCLiveConfig{
			Live:           &s.slots.live,
			GalleryImage:   spec.GalleryImage,
			Tool:           &s.slots.tool,
			GameStart:      spec.InitPrompt,
			Language:       spec.Language,
			Voice:          spec.Voice,
			VoiceStyle:     spec.VoiceStyle,
			Voices:         &s.slots.voices,
			BuiltinVoices:  openai.LiveVoices,
			Guardrail:      spec.guardrail(),
			Scenario:       spec.Scenario,
			InitPrompt:     spec.initPrompt(),
			PromptOverride: spec.PromptOverride,
			IdleAfter:      spec.idleAfter(),
		})
	case GenreNPCDuo:
		return genre.NewNPCDuo(genre.NPCDuoConfig{
			Live:           &s.slots.live,
			Tool:           &s.slots.tool,
			GalleryImage:   spec.GalleryImage,
			Scenario:       spec.Scenario,
			Guardrail:      spec.Guardrail,
			GameStart:      spec.InitPrompt,
			BuiltinVoices:  openai.LiveVoices,
			Language:       spec.Language,
			IdleAfter:      spec.idleAfter(),
			PromptOverride: spec.PromptOverride,
		})
	}
	return nil, fmt.Errorf("unknown genre %q", spec.Genre)
}
