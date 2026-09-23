// Package engine is the whole public surface. Everything else lives under
// internal/, where the compiler refuses an import from outside this module —
// not a convention, not a lint rule.
//
// The engine owns no HTTP. A caller wraps these calls in whatever transport it
// runs, which is what lets the same code be a standalone server or a package
// compiled into the platform's binary.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/genre"
	"engine/internal/ports"
)

// KeyFunc yields the API key for a session's platform. The platform injects one
// closing over its existing resolver; a standalone launcher injects one reading
// a flag or a config file. The engine itself never learns where a key lives.
type KeyFunc func(ctx context.Context) (string, error)

// Tier is re-exported so a caller can name one without importing internal/.
type Tier = adapters.Tier

const (
	TierEconomy  = adapters.TierEconomy
	TierBalanced = adapters.TierBalanced
	TierPremium  = adapters.TierPremium
	TierMax      = adapters.TierMax
)

// Platform selects which implementations back the adapter roles. The engine
// never resolves this itself: it arrives in the spec, already decided.
type Platform string

const (
	PlatformMock   Platform = "mock"
	PlatformOpenAI Platform = "openai"
)

type Genre string

const (
	GenreAdventure Genre = "adventure"
	GenreNPCLive   Genre = "npc-live"
)

// SessionSpec is everything the engine needs to run a session. It arrives fully
// resolved: the engine never calls back into platform data.
type SessionSpec struct {
	Genre Genre `json:"genre"`

	// ID identifies the session to whatever Persist was injected.
	ID string `json:"id,omitempty"`

	// Title is what this game is called, for a client with a header to put it
	// in. The engine does nothing with it but hand it back on the topology:
	// naming a game is the author's business, and a player asking what they are
	// playing is not answered by the name of the genre.
	Title string `json:"title,omitempty"`

	// Persist is injected by whoever embeds the engine, never read from a spec
	// file: it is behaviour, not configuration.
	Persist Persist `json:"-"`

	// Guardrail is the platform's resolved youth-protection constraint. It is
	// not authored by a game designer.
	Guardrail string `json:"guardrail,omitempty"`

	// Scenario is the game designer's text: the setting and rules for Adventure,
	// who the character is and what they must not concede for NPC-Live. Distinct
	// from Guardrail on purpose — collapsing the two would hand a game author a
	// lever on youth protection.
	Scenario string `json:"scenario,omitempty"`

	// InitPrompt is the first message sent once preparation is done — what v1
	// calls the initialization prompt. It kicks the game off: for a live genre
	// the character greets whoever arrived instead of waiting to be spoken to.
	// Empty means a sensible default.
	InitPrompt string `json:"initPrompt,omitempty"`

	// Platform selects which implementations back the adapter roles. An empty
	// Platform means mock: no key, no network, no cost.
	Platform Platform `json:"platform,omitempty"`

	// Keys supplies the API key on demand. It is injected rather than stored,
	// for two reasons: the spec is persisted as a blob, so a key field would
	// write the secret into the database once per session; and resolving at the
	// point of use means a rotated or revoked key takes effect without
	// relaunching.
	Keys KeyFunc `json:"-"`

	// ModelTier is the default every unset Model* field resolves through.
	// Empty means balanced.
	ModelTier Tier `json:"modelTier,omitempty"`

	// The Model* fields pin one role for this session. A value starting with
	// "$" names a tier — "$max" means "this role's model at the max tier" —
	// and anything else is a literal model name. The sigil is what keeps the
	// two unambiguous.
	ModelLive     string `json:"modelLive,omitempty"`
	ModelTool     string `json:"modelTool,omitempty"`
	ModelThreaded string `json:"modelThreaded,omitempty"`
	ModelImage    string `json:"modelImage,omitempty"`
	ModelAudio    string `json:"modelAudio,omitempty"`

	// Voice is a parameter of the live and audio models rather than a model of
	// its own, which is why it sits outside the Model* family. One of the
	// provider's built-in names; empty takes the provider's default. It cannot
	// change once a session exists.
	Voice string `json:"voice,omitempty"`

	// PromptOverride replaces one of a genre's own prompts by name, so a
	// session can be retuned without a new build. The names a genre has are
	// reported by Prompts; an unknown one is an error rather than ignored,
	// because a misspelling would otherwise run the default silently.
	PromptOverride map[string]string `json:"promptOverride,omitempty"`

	// MuteObserver runs a genre's observer as a node that judges nothing. The
	// voice path is worth proving before a second model is added to it.
	MuteObserver bool `json:"muteObserver,omitempty"`

	// Status seeds the property values a genre tracks — Adventure's status
	// fields. The engine neither invents nor interprets them.
	Status map[string]string `json:"status,omitempty"`

	// IdleSeconds is how long a live conversation may go unspoken before it is
	// closed and the game paused. It costs money for every second it stays
	// open, so an abandoned one is let go rather than left running; a player
	// picks it up again and the character remembers.
	//
	// Zero takes a sensible default. Negative never pauses.
	IdleSeconds int `json:"idleSeconds,omitempty"`

	// MockPaceMs slows the mock platform down to something a person can watch.
	// Zero is instant, which is what a test wants; a dev run showing how a turn
	// is assembled wants blocks that visibly take time. It reaches nothing but
	// the mock platform.
	MockPaceMs int `json:"mockPaceMs,omitempty"`

	// Script replaces player input with canned utterances, for a dev launcher
	// or a test. It reaches a turn-based genre only: a live conversation's
	// audio never passes through the engine, so there is no stream for a script
	// to stand in for.
	Script []string `json:"script,omitempty"`
	// ScriptIntervalMs is milliseconds, spelled out because a time.Duration
	// field would silently read a JSON number as nanoseconds.
	ScriptIntervalMs int `json:"scriptIntervalMs,omitempty"`
}

// Event is one item on the session-scoped stream. Stream names the output block
// it came from, so the set of possible values is the genre's wiring.
type Event struct {
	Stream string
	Value  string
}

// BlockState is a block reporting on itself, as it appears on the wire. It is
// declared here rather than reusing the internal type so the public surface —
// and the generated client — does not depend on internals.
type BlockState struct {
	Node  string `json:"node"`
	Phase string `json:"phase"`
}

// StreamNames is every stream a genre can wire. It is the source the generated
// TypeScript union is built from, so adding an output block here is what makes
// the client aware of it.
func StreamNames() []string {
	return []string{"text", "audio", "image", "props", "state", "usage", "connect", "pause", "error"}
}

type Session struct {
	spec     SessionSpec
	wiring   *genre.Wiring
	usage    *usageLedger
	recorder *recorder
	events   chan Event
	cancel   context.CancelFunc
	once     sync.Once
}

// SessionState is what a session must carry across a restart, separate from the
// spec because the spec is fixed at launch and this changes every turn. Keyed by
// block name: a genre may wire two independent threads, and one field could not
// hold both.
type SessionState struct {
	Blocks map[string]string `json:"blocks,omitempty"`
}

// Resume launches a session and restores each block's provider-side state, so a
// threaded genre continues its conversation rather than starting a new one.
//
// A stored state naming a block the wiring no longer has is an error, not
// something to limp past: the wiring changed under the session, and that is
// exactly the case that should invalidate it.
func Resume(ctx context.Context, spec SessionSpec, state SessionState) (*Session, error) {
	return launch(ctx, spec, &state)
}

// Launch validates a genre's wiring and starts it. The returned session is a
// self-sufficient handle: whoever holds it can drive the conversation.
func Launch(ctx context.Context, spec SessionSpec) (*Session, error) {
	return launch(ctx, spec, nil)
}

func launch(ctx context.Context, spec SessionSpec, state *SessionState) (*Session, error) {
	var w *genre.Wiring
	switch spec.Genre {
	case GenreAdventure:
		w = genre.NewAdventure(spec.Status)
	case GenreNPCLive:
		live, tool, image, err := spec.adapters()
		if err != nil {
			return nil, err
		}
		w, err = genre.NewNPCLive(genre.NPCLiveConfig{
			Live:           live,
			Tool:           tool,
			Image:          image,
			Voice:          spec.Voice,
			Guardrail:      spec.Guardrail,
			Scenario:       spec.Scenario,
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

	// Restored before anything starts, so the first turn already runs on the
	// continued thread.
	if state != nil && len(state.Blocks) > 0 {
		if err := w.Graph.RestoreState(state.Blocks); err != nil {
			return nil, err
		}
	}

	if spec.Persist == nil {
		spec.Persist = NoopPersist{}
	}
	if spec.ID == "" {
		spec.ID = "dev-session"
	}

	ctx, cancel := context.WithCancel(ctx)

	s := &Session{
		spec:     spec,
		wiring:   w,
		usage:    newUsageLedger(spec.Platform),
		recorder: newRecorder(),
		events:   make(chan Event, 256),
	}
	s.cancel = cancel

	// Subscribed before anything starts: a block reports its opening phase as it
	// starts, and a subscriber attached afterwards would never see it.
	states := w.Graph.ObserveStates(ctx)
	usage := w.Graph.ObserveUsage(ctx)

	// Subscribed before the graph starts, for the same reason as the others: the
	// invitation is one event, and a subscriber attached afterwards would miss
	// the only one there is.
	var invitations, pauses <-chan ports.State
	if w.Live != nil {
		invitations = w.Live.Invitations()
		pauses = w.Live.Pauses()
	}

	w.Graph.Start(ctx)
	s.merge(ctx)
	s.forwardStates(ctx, states)
	s.forwardUsage(ctx, usage)
	s.forwardInvitations(ctx, invitations)
	s.forwardSignal(ctx, pauses, "pause")
	return s, nil
}

// merge folds every wired sink into one session-scoped stream. A turn is a
// bracketed span of these events rather than a stream of its own.
func (s *Session) merge(ctx context.Context) {
	for name, ch := range s.wiring.Sinks() {
		go func(name string, ch chan string) {
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-ch:
					if !ok {
						return
					}
					ev := Event{Stream: name, Value: v}
					s.recorder.record(ev)
					if name == "props" {
						s.recorder.recordProps(parseProps(v))
					}
					_ = s.spec.Persist.Save(ctx, s.spec.ID, ev)
					select {
					case s.events <- ev:
					case <-ctx.Done():
						return
					}
				}
			}
		}(name, ch)
	}
}

// observeStates puts every block's own report on the session stream. It is a
// debugging surface first — watching which block is working, and for how long,
// is how you see what a turn actually did — and the data a graph view draws.
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
				ev := Event{Stream: "state", Value: string(blob)}
				_ = s.spec.Persist.Save(ctx, s.spec.ID, ev)
				select {
				case s.events <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

// forwardUsage keeps the ledger current and puts the whole report on the stream
// whenever it changes. The report rather than the delta, because pricing needs
// the price table and that lives here: a client renders what it is given.
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
				ev := Event{Stream: "usage", Value: string(blob)}
				_ = s.spec.Persist.Save(ctx, s.spec.ID, ev)
				select {
				case s.events <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

// forwardInvitations puts "a player may connect now" on the session stream.
// The client waits for it rather than connecting when its page loads, so the
// game has begun — and the portrait is on screen — before anyone is asked to
// speak. It carries nothing: the moment is the whole message.
func (s *Session) forwardInvitations(ctx context.Context, invitations <-chan ports.State) {
	s.forwardSignal(ctx, invitations, "connect")
}

// forwardSignal puts a session lifecycle moment on the stream.
//
// These carry nothing: the moment is the whole message. They are the third kind
// of thing the stream holds — beside what reached a sink, and what the graph
// reports about itself — and like the second they come from introspection
// rather than from an edge, because their reader is the client rather than
// another block.
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
				case s.events <- Event{Stream: stream}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

// Usage is what the session has spent so far, per block and per model.
func (s *Session) Usage() UsageReport { return s.usage.report() }

// Prompts are the genre's own prompts, after any the spec replaced. A session
// runs on text somebody wrote, and on a platform that teaches how AI works that
// text is the subject rather than an implementation detail.
func (s *Session) Prompts() map[string]string { return s.wiring.Prompts }

// Connect brokers a browser's offer to talk to the character directly. The
// conversation runs between the player and the provider; what comes back here
// is only what the browser needs to complete it.
//
// It reports an error on a genre that holds no conversation, rather than
// leaving a caller waiting for an answer that will never come.
func (s *Session) Connect(ctx context.Context, offer string) (string, error) {
	if s.wiring.Live == nil {
		return "", fmt.Errorf("genre %q holds no live conversation", s.spec.Genre)
	}
	return s.wiring.Live.Connect(ctx, offer)
}

// Snapshot is where the session stands now, for a client that was not watching
// — a reloaded page, or one opened halfway through.
func (s *Session) Snapshot() Snapshot {
	started := false
	select {
	case <-s.wiring.Gate.Ready():
		started = true
	default:
	}
	return s.recorder.snapshot(started, s.usage.report(), s.conversation())
}

// conversation is where the live block stands, or empty for a genre with no
// live block at all.
func (s *Session) conversation() string {
	if s.wiring.Live == nil {
		return ""
	}
	return s.wiring.Live.Standing()
}

// NodeDetail is everything the engine knows about one block: what it is, what
// it is doing, what recently passed through it, and what it has cost.
type NodeDetail struct {
	ports.NodeDetail
	Phase string       `json:"phase"`
	Usage *UsageRecord `json:"usage,omitempty"`
}

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
	return detail, true
}

// InspectEdge reports what one wire has carried.
func (s *Session) InspectEdge(from, to, kind string) (ports.EdgeDetail, bool) {
	return s.wiring.Graph.InspectEdge(from, to, kind)
}

// History is the conversation so far, in order. A client applies these through
// exactly the same path as live events, so there is no second way to render a
// session.
func (s *Session) History() []Event { return s.recorder.replay() }

// Events is the session-scoped stream the transport serialises.
func (s *Session) Events() <-chan Event { return s.events }

// Say submits typed player input. It reports an error on a genre that takes no
// typed input, rather than discarding it silently.
func (s *Session) Say(text string) error {
	if s.wiring.Say == nil {
		return fmt.Errorf("genre %q takes no typed input", s.spec.Genre)
	}
	s.wiring.Say(text)
	return nil
}

// Speak submits player audio.
func (s *Session) Speak(audio []byte) error {
	if s.wiring.Speak == nil {
		return fmt.Errorf("genre %q takes no audio input", s.spec.Genre)
	}
	s.wiring.Speak(audio)
	return nil
}

// Pause lets a live conversation go now. The session stays; what ends is the
// part that costs money by the second, and a resume picks it up again with the
// character remembering what was said.
func (s *Session) Pause() error {
	if s.wiring.Live == nil {
		return fmt.Errorf("genre %q holds no live conversation", s.spec.Genre)
	}
	s.wiring.Live.Pause()
	return nil
}

// Ready closes when every block the first turn depends on has reported done.
// A genre with nothing to prepare is ready at once, so a caller never has to
// ask which kind it got.
func (s *Session) Ready() <-chan struct{} { return s.wiring.Gate.Ready() }

// Describe renders the running wiring, so a launcher can show the graph it got.
func (s *Session) Describe() string { return s.wiring.Graph.Describe() }

// Topology is the wiring in a form a view can draw, together with what a player
// may do with it.
func (s *Session) Topology() ports.Topology {
	t := s.wiring.Graph.Topology()
	t.Title = s.spec.Title
	t.Imagery = string(s.wiring.Imagery)
	t.Inputs = make([]string, 0, len(s.wiring.Inputs))
	for _, mode := range s.wiring.Inputs {
		t.Inputs = append(t.Inputs, string(mode))
	}
	return t
}

// Mermaid is the wiring as a flowchart.
func (s *Session) Mermaid() string { return s.wiring.Graph.Mermaid() }

// State is what Resume needs back. It is read at a checkpoint rather than at the
// end, because a conversation may never reach an end.
func (s *Session) State() SessionState {
	return SessionState{Blocks: s.wiring.Graph.ExportState()}
}

func (s *Session) Close() { s.once.Do(s.cancel) }

// parseProps reads back the rendered map a props sink emits. The engine renders
// it for the stream and parses it here rather than carrying two shapes, which is
// a wart worth removing when props become structured on the wire.
func parseProps(value string) map[string]string {
	inner := strings.TrimSuffix(strings.TrimPrefix(value, "map["), "]")
	props := map[string]string{}
	for _, pair := range strings.Fields(inner) {
		key, val, found := strings.Cut(pair, ":")
		if !found {
			continue
		}
		props[key] = val
	}
	return props
}

// tierSigil marks a Model* value as a reference to a tier rather than a model
// name. No model name begins with it, so the two never have to be guessed apart.
const tierSigil = "$"

// initPrompt is what the gate sends to begin the game. The default asks the
// character to speak first without telling it what to say, which is the
// scenario's job.
func (spec SessionSpec) initPrompt() string {
	if spec.InitPrompt != "" {
		return spec.InitPrompt
	}
	return "Begin. Greet whoever has arrived, in character, in one or two sentences."
}

// defaultIdle is long enough not to interrupt somebody reading the scene, and
// well past the point where closing is worth it: creating a conversation bills
// fifteen seconds up front, so pausing pays only for silences longer than that.
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

func (spec SessionSpec) modelTier() Tier {
	if spec.ModelTier == "" {
		return TierBalanced
	}
	return spec.ModelTier
}

// resolveModels turns the spec's tier and per-role pins into concrete model
// names. Roles left unset come from ModelTier; a role pinned to "$tier" comes
// from that tier instead; anything else is used verbatim.
//
// byTier is the platform's own preset table, so the engine holds no model names
// of its own.
func (spec SessionSpec) resolveModels(byTier func(Tier) (adapters.ModelSet, error)) (adapters.ModelSet, error) {
	models, err := byTier(spec.modelTier())
	if err != nil {
		return models, err
	}

	for _, role := range []struct {
		name string
		pin  string
		at   func(*adapters.ModelSet) *string
	}{
		{"modelLive", spec.ModelLive, func(m *adapters.ModelSet) *string { return &m.Live }},
		{"modelTool", spec.ModelTool, func(m *adapters.ModelSet) *string { return &m.Tool }},
		{"modelThreaded", spec.ModelThreaded, func(m *adapters.ModelSet) *string { return &m.Threaded }},
		{"modelImage", spec.ModelImage, func(m *adapters.ModelSet) *string { return &m.Image }},
		{"modelAudio", spec.ModelAudio, func(m *adapters.ModelSet) *string { return &m.Audio }},
	} {
		if role.pin == "" {
			continue
		}
		if !strings.HasPrefix(role.pin, tierSigil) {
			*role.at(&models) = role.pin
			continue
		}

		tier := Tier(strings.TrimPrefix(role.pin, tierSigil))
		other, err := byTier(tier)
		if err != nil {
			return models, fmt.Errorf("%s: %w", role.name, err)
		}
		*role.at(&models) = *role.at(&other)
	}
	return models, nil
}

// adapters resolves the spec's platform choice into implementations. Defaults
// live here rather than in a block, so a block never has an opinion about which
// model runs it.
func (spec SessionSpec) adapters() (adapters.Live, adapters.Tool, adapters.Image, error) {
	switch spec.Platform {
	case "", PlatformMock:
		pace := time.Duration(spec.MockPaceMs) * time.Millisecond
		return mock.Live{Pace: pace}, mock.Tool{Pace: pace}, mock.Image{Pace: pace}, nil
	case PlatformOpenAI:
		if spec.Keys == nil {
			return nil, nil, nil, fmt.Errorf("platform %q needs a Keys function", spec.Platform)
		}
		models, err := spec.resolveModels(openai.Models)
		if err != nil {
			return nil, nil, nil, err
		}
		if models.Image == "" {
			return nil, nil, nil, fmt.Errorf("tier %q generates no images, which this genre needs", spec.modelTier())
		}

		if spec.Voice != "" && !openai.ValidVoice(spec.Voice) {
			return nil, nil, nil, fmt.Errorf("unknown voice %q", spec.Voice)
		}

		keys := adapters.KeyFunc(spec.Keys)
		return openai.NewLive(keys, models.Live),
			openai.NewTool(keys, models.Tool),
			openai.NewImage(keys, models.Image, models.ImageQuality, models.ImageSize), nil
	default:
		return nil, nil, nil, fmt.Errorf("unknown platform %q", spec.Platform)
	}
}
