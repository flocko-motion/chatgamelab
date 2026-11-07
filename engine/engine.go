// Package engine is the module's whole public surface; it owns no HTTP, so it runs standalone or embedded.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
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

	// InitPrompt is sent once preparation is done, so the character speaks first.
	InitPrompt string `json:"initPrompt,omitempty"`

	// Platform empty means mock: no key, no network, no cost.
	Platform Platform `json:"platform,omitempty"`

	// Keys is a function so the persisted spec holds no secret and a rotated key applies on the next call.
	Keys KeyFunc `json:"-"`

	// ModelTier picks every model not pinned in Models; empty means medium.
	ModelTier Tier `json:"modelTier,omitempty"`

	// Models pins one model type: "$high" takes that tier's model, anything else names a model.
	Models map[ModelType]string `json:"models,omitempty"`

	// Voice is a provider built-in voice name, fixed once the session exists.
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

// Event is one item on the session-scoped stream; Stream names the output block it came from.
type Event struct {
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

// SessionState is what changes every turn and must survive a restart, keyed by block name.
// Store it with a schema_version and invalidate on mismatch; JSON decoding misses semantic changes.
type SessionState struct {
	Blocks map[string]string `json:"blocks,omitempty"`
}

// Resume errors on a stored block the wiring lacks: the wiring changed under the session, invalidating it.
func Resume(ctx context.Context, spec SessionSpec, state SessionState) (*Session, error) {
	return launch(ctx, spec, &state)
}

// Launch validates a genre's wiring and starts it.
func Launch(ctx context.Context, spec SessionSpec) (*Session, error) {
	return launch(ctx, spec, nil)
}

func launch(ctx context.Context, spec SessionSpec, state *SessionState) (*Session, error) {
	if err := spec.playable(); err != nil {
		return nil, err
	}

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

	// Restored before anything starts, so the first turn runs on the continued thread.
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

	// Subscribed before Start, or the opening phases and the one invitation would be missed.
	states := w.Graph.ObserveStates(ctx)
	usage := w.Graph.ObserveUsage(ctx)

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

// merge folds every wired sink into one session-scoped stream.
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

// Prompts are the genre's own prompts, after any the spec replaced.
func (s *Session) Prompts() map[string]string { return s.wiring.Prompts }

// Connect brokers a browser's offer to talk to the provider directly; errors on a genre with no conversation.
func (s *Session) Connect(ctx context.Context, offer string) (string, error) {
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
	return s.recorder.snapshot(started, s.usage.report(), s.conversation())
}

// conversation is where the live block stands, or empty without one.
func (s *Session) conversation() string {
	if s.wiring.Live == nil {
		return ""
	}
	return s.wiring.Live.Standing()
}

// NodeDetail is everything the engine knows about one block.
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

// History is the conversation so far, applied by a client through the same path as live events.
func (s *Session) History() []Event { return s.recorder.replay() }

// Events is the session-scoped stream the transport serialises.
func (s *Session) Events() <-chan Event { return s.events }

// Say submits typed player input, erroring on a genre that takes none.
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

// Pause closes the per-second-billed live conversation; the session stays and a resume continues it.
func (s *Session) Pause() error {
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
	t.Imagery = string(s.wiring.Imagery)
	t.Inputs = make([]string, 0, len(s.wiring.Inputs))
	for _, mode := range s.wiring.Inputs {
		t.Inputs = append(t.Inputs, string(mode))
	}
	return t
}

// Mermaid is the wiring as a flowchart.
func (s *Session) Mermaid() string { return s.wiring.Graph.Mermaid() }

// State is what Resume needs back, read at checkpoints because a conversation may never end.
func (s *Session) State() SessionState {
	return SessionState{Blocks: s.wiring.Graph.ExportState()}
}

func (s *Session) Close() { s.once.Do(s.cancel) }

// parseProps reads back the rendered map a props sink emits; remove once props are structured on the wire.
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

// tierSigil marks a Model* value as a tier; no model name begins with it.
const tierSigil = "$"

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

func (spec SessionSpec) modelTier() Tier {
	if spec.ModelTier == "" {
		return TierMedium
	}
	return spec.ModelTier.Normal()
}

// resolveModels turns the tier and the pins into a model per type from the platform's byTier table.
func (spec SessionSpec) resolveModels(byTier func(Tier) (adapters.ModelSet, error)) (adapters.ModelSet, error) {
	models, err := byTier(spec.modelTier())
	if err != nil {
		return models, err
	}

	for _, kind := range slices.Sorted(maps.Keys(spec.Models)) {
		if !slices.Contains(adapters.ModelTypes(), kind) {
			return models, fmt.Errorf("models: unknown model type %q", kind)
		}
		pin := spec.Models[kind]
		if pin == "" {
			continue
		}
		if !strings.HasPrefix(pin, tierSigil) {
			// A pinned name keeps the tier's parameters, which ride with the price.
			model := models[kind]
			model.Name = pin
			models[kind] = model
			continue
		}

		other, err := byTier(Tier(strings.TrimPrefix(pin, tierSigil)).Normal())
		if err != nil {
			return models, fmt.Errorf("models.%s: %w", kind, err)
		}
		model, ok := other[kind]
		if !ok {
			return models, fmt.Errorf("models.%s: tier %q has no %s model", kind, pin, kind)
		}
		models[kind] = model
	}
	return models, nil
}

// adapters resolves the platform into implementations, so a block never chooses its own model.
// A role the platform cannot fill comes back nil.
func (spec SessionSpec) adapters() (adapters.Live, adapters.Tool, adapters.Image, error) {
	switch spec.Platform {
	case "", PlatformMock:
		pace := time.Duration(spec.MockPaceMs) * time.Millisecond
		return mock.Live{Pace: pace}, mock.Tool{Pace: pace}, mock.Image{Pace: pace}, nil
	case PlatformOpenAI:
		if spec.Keys == nil {
			return nil, nil, nil, fmt.Errorf("platform %q needs a Keys function", spec.Platform)
		}
		models, err := spec.resolveModels(openai.Platform().Models)
		if err != nil {
			return nil, nil, nil, err
		}

		if spec.Voice != "" && !openai.ValidVoice(spec.Voice) {
			return nil, nil, nil, fmt.Errorf("unknown voice %q", spec.Voice)
		}

		keys := adapters.KeyFunc(spec.Keys)
		image := models[adapters.ModelImage]
		return openai.NewLive(keys, models[adapters.ModelLive].Name),
			openai.NewTool(keys, models[adapters.ModelTooling].Name),
			openai.NewImage(keys, image.Name, image.Params[adapters.ParamQuality], image.Params[adapters.ParamSize]), nil
	case PlatformMistral:
		if spec.Keys == nil {
			return nil, nil, nil, fmt.Errorf("platform %q needs a Keys function", spec.Platform)
		}
		models, err := spec.resolveModels(mistral.Platform().Models)
		if err != nil {
			return nil, nil, nil, err
		}

		// Live stays nil: Mistral has no live voice model.
		keys := adapters.KeyFunc(spec.Keys)
		return nil,
			mistral.NewTool(keys, models[adapters.ModelTooling].Name),
			mistral.NewImage(keys, models[adapters.ModelImage].Name), nil
	default:
		return nil, nil, nil, fmt.Errorf("unknown platform %q", spec.Platform)
	}
}
