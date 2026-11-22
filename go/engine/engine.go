// Package engine is the module's whole public surface; it owns no HTTP, so it runs standalone or embedded.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
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
)

// SessionSpec is everything a session needs, fully resolved so the engine never calls back into the platform.
type SessionSpec struct {
	Genre Genre `json:"genre"`

	ID string `json:"id,omitempty"`

	// Title is handed back on the topology for a client's header.
	Title string `json:"title,omitempty"`

	Persist Persist `json:"-"`

	// Guardrail is the platform's resolved youth-protection constraint.
	Guardrail string `json:"guardrail,omitempty"`

	// Scenario is the designer's text, kept apart from Guardrail so an author has no lever on youth protection.
	Scenario string `json:"scenario,omitempty"`

	// ImageStyle is how pictures are painted; empty takes the genre's default so the style stays stable.
	ImageStyle string `json:"imageStyle,omitempty"`

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

	// MuteObserver keeps the observer node but has it judge nothing.
	MuteObserver bool `json:"muteObserver,omitempty"`

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

// StreamNames is every stream a genre can wire, and the source of the generated TypeScript union.
func StreamNames() []string {
	return []string{"text", "said", "audio", "image", "props", "state", "usage", "connect", "pause", "settled", "error", "problem"}
}

type Session struct {
	// mu guards what settling, adapting and toggling change: the spec's offers
	// and preference, and the settlement.
	mu sync.Mutex
	// settled is nil while the player is choosing.
	settled *settlement
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
	events   chan SessionEvent
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once

	// logMu keeps the stored history in the order the recorder holds it.
	logMu sync.Mutex
	// dirty asks the keeper for a checkpoint; kept closes once its last one is written.
	dirty chan struct{}
	kept  chan struct{}
}

// slots hold the adapters the blocks call, filled when the session settles
// and replaced when it adapts.
type slots struct {
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
}

// Resume continues a stored session on the offers at hand, fitting its stored
// preference to them. It refuses with ErrStale a session stored under another
// SchemaVersion, or one whose blocks the wiring no longer has, and with
// ErrUnknown one the store does not hold.
//
// The context is the session's lifetime, as for Launch.
func Resume(ctx context.Context, id string, persist Persist, offers []Offer) (*Session, error) {
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
		events:   make(chan SessionEvent, 256),
		usage:    newUsageLedger(),
	}

	var w *genre.Wiring
	switch spec.Genre {
	case GenreAdventure:
		var err error
		w, err = genre.NewAdventure(genre.AdventureConfig{
			Tool:                &s.slots.tool,
			Hearing:             &s.slots.hearing,
			Thread:              &s.slots.thread,
			Window:              s.usage.window,
			Image:               &s.slots.image,
			Scenario:            spec.Scenario,
			Guardrail:           spec.Guardrail,
			Description:         spec.Description,
			ImageStyle:          spec.ImageStyle,
			StartingMessage:     spec.InitPrompt,
			Status:              spec.Status,
			Language:            spec.Language,
			Dice:                spec.Dice,
			PromptOverride:      spec.PromptOverride,
			ConstraintFrequency: spec.ConstraintFrequency,
		})
		if err != nil {
			return nil, err
		}
	case GenreNPCLive:
		var err error
		w, err = genre.NewNPCLive(genre.NPCLiveConfig{
			Live:           &s.slots.live,
			Tool:           &s.slots.tool,
			Image:          &s.slots.image,
			Voice:          spec.Voice,
			Guardrail:      spec.Guardrail,
			Scenario:       spec.Scenario,
			ImageStyle:     spec.ImageStyle,
			InitPrompt:     spec.initPrompt(),
			MuteObserver:   spec.MuteObserver,
			PromptOverride: spec.PromptOverride,
			IdleAfter:      spec.idleAfter(),
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown genre %q", spec.Genre)
	}

	if err := w.Graph.Validate(); err != nil {
		return nil, err
	}
	s.watchKeys()

	// Restored before anything starts, so the first turn runs on the continued thread.
	if from != nil {
		if err := w.Graph.RestoreState(from.state.Blocks); err != nil {
			return nil, &Refusal{Problem: ProblemStale, Message: err.Error()}
		}
		s.usage.restore(from.state.Usage)
		s.recorder.restore(from.history)
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
	go s.keep()
	go s.deliver()
	go s.release()

	// Subscribed before Start, or the opening phases and the one invitation would be missed.
	states := w.Graph.ObserveStates(s.ctx)
	usage := w.Graph.ObserveUsage(s.ctx)

	var invitations, pauses <-chan ports.State
	if w.Live != nil {
		invitations = w.Live.Invitations()
		pauses = w.Live.Pauses()
	}

	s.merge(s.ctx)
	s.forwardStates(s.ctx, states)
	s.forwardUsage(s.ctx, usage)
	s.forwardInvitations(s.ctx, invitations)
	s.forwardSignal(s.ctx, pauses, "pause")

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
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-ch:
					if !ok {
						return
					}
					if name == "problem" {
						s.failed(v)
						continue
					}
					ev := SessionEvent{Stream: name, Value: v}
					s.log(ev)
					if name == "props" {
						if props := readProps(v); props != nil {
							s.recorder.recordProps(props)
						}
					}
					select {
					case s.events <- ev:
					case <-ctx.Done():
						return
					}
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

				blob, err := json.Marshal(BlockState{Node: state.Node, Phase: string(state.Phase)})
				if err != nil {
					continue
				}
				ev := SessionEvent{Stream: "state", Value: string(blob)}
				if state.Phase == ports.PhaseReady {
					s.checkpoint()
				}
				select {
				case s.events <- ev:
				case <-ctx.Done():
					return
				}
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
				select {
				case s.events <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

// forwardInvitations says a player may connect now, so the game has begun before anyone is asked to speak.
func (s *Session) forwardInvitations(ctx context.Context, invitations <-chan ports.State) {
	s.forwardSignal(ctx, invitations, "connect")
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
				select {
				case s.events <- SessionEvent{Stream: stream}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

// Usage is what the session has spent so far, per block and per model.
func (s *Session) Usage() UsageReport { return s.usage.report() }

// Prompts are the genre's own prompts, after any the spec replaced.
func (s *Session) Prompts() map[string]string { return s.wiring.Prompts }

// Connect brokers a browser's offer to talk to the provider directly; errors on a genre with no conversation.
func (s *Session) Connect(ctx context.Context, offer string) (string, error) {
	if err := s.playing(); err != nil {
		return "", err
	}
	if s.wiring.Live == nil {
		return "", fmt.Errorf("genre %q holds no live conversation", s.spec.Genre)
	}
	return s.wiring.Live.Connect(ctx, offer)
}

// Snapshot is where the session stands now, for a reloaded or late-opened page.
func (s *Session) Snapshot() Snapshot {
	started := false
	select {
	case <-s.wiring.Gate.Ready():
		started = true
	default:
	}
	snapshot := s.recorder.snapshot(started, s.usage.report(), s.conversation())
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot.Choosing = s.settled == nil
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
}

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
	return detail, true
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

// InspectEdge reports what one wire has carried.
func (s *Session) InspectEdge(from, to, kind string) (ports.EdgeDetail, bool) {
	return s.wiring.Graph.InspectEdge(from, to, kind)
}

// History is the conversation so far, applied by a client through the same path as live events.
func (s *Session) History() []SessionEvent { return s.recorder.replay() }

// Events is the session-scoped stream the transport serialises.
func (s *Session) Events() <-chan SessionEvent { return s.events }

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
	return s.wiring.Speak(audio)
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
	return nil
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
	t.Inputs = make([]string, 0, len(s.wiring.Inputs))
	for _, mode := range s.wiring.Inputs {
		t.Inputs = append(t.Inputs, string(mode))
	}
	return t
}

// Mermaid is the wiring as a flowchart.
func (s *Session) Mermaid() string { return s.wiring.Graph.Mermaid() }

// ID is the session's ID, the handle that plays it.
func (s *Session) ID() string { return s.spec.ID }

// release closes the adapters once the session has ended.
func (s *Session) release() {
	<-s.ctx.Done()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kit.close()
	s.kit = equipment{}
}

// Close ends the session once its last checkpoint is written.
func (s *Session) Close() {
	s.once.Do(s.cancel)
	<-s.kept
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
