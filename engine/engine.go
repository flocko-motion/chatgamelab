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
	"engine/internal/blocks"
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
	// its own, which is why it sits outside the Model* family.
	Voice string `json:"voice,omitempty"`

	// Status seeds the property values a genre tracks — Adventure's status
	// fields. The engine neither invents nor interprets them.
	Status map[string]string `json:"status,omitempty"`

	// Script replaces live player input with canned utterances, for a dev
	// launcher or a test.
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
	return []string{"text", "audio", "image", "props", "flag", "state", "usage", "error"}
}

type Session struct {
	spec   SessionSpec
	wiring *genre.Wiring
	usage  *usageLedger
	events chan Event
	cancel context.CancelFunc
	once   sync.Once
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
		w = genre.NewNPCLive(genre.NPCLiveConfig{
			Live:  live,
			Tool:  tool,
			Image: image,
			LiveCfg: adapters.LiveConfig{
				Voice:        spec.Voice,
				Instructions: spec.Scenario + "\n" + spec.Guardrail,
			},
			Guardrail:  spec.Guardrail,
			Scenario:   spec.Scenario,
			InitPrompt: spec.initPrompt(),
			Script: blocks.InputScript{
				Lines:    spec.Script,
				Interval: time.Duration(spec.ScriptIntervalMs) * time.Millisecond,
			},
		})
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
		spec:   spec,
		wiring: w,
		usage:  newUsageLedger(spec.Platform),
		events: make(chan Event, 256),
	}
	s.cancel = cancel

	// Subscribed before anything starts: a block reports its opening phase as it
	// starts, and a subscriber attached afterwards would never see it.
	states := w.Graph.ObserveStates(ctx)
	usage := w.Graph.ObserveUsage(ctx)

	w.Graph.Start(ctx)
	s.merge(ctx)
	s.forwardStates(ctx, states)
	s.forwardUsage(ctx, usage)
	return s, nil
}

// merge folds every wired sink into one session-scoped stream. A turn is a
// bracketed span of these events rather than a stream of its own.
func (s *Session) merge(ctx context.Context) {
	for name, ch := range s.wiring.Sinks {
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

// Usage is what the session has spent so far, per block and per model.
func (s *Session) Usage() UsageReport { return s.usage.report() }

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

// Ready closes when every block the first turn depends on has reported done.
// A genre with nothing to prepare is ready at once, so a caller never has to
// ask which kind it got.
func (s *Session) Ready() <-chan struct{} { return s.wiring.Gate.Ready() }

// Describe renders the running wiring, so a launcher can show the graph it got.
func (s *Session) Describe() string { return s.wiring.Graph.Describe() }

// Topology is the wiring in a form a view can draw.
func (s *Session) Topology() ports.Topology { return s.wiring.Graph.Topology() }

// Mermaid is the wiring as a flowchart.
func (s *Session) Mermaid() string { return s.wiring.Graph.Mermaid() }

// State is what Resume needs back. It is read at a checkpoint rather than at the
// end, because a conversation may never reach an end.
func (s *Session) State() SessionState {
	return SessionState{Blocks: s.wiring.Graph.ExportState()}
}

func (s *Session) Close() { s.once.Do(s.cancel) }

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
		return mock.Live{}, mock.Tool{}, mock.Image{}, nil
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

		keys := adapters.KeyFunc(spec.Keys)
		return openai.NewLive(keys, models.Live),
			openai.NewTool(keys, models.Tool),
			openai.NewImage(keys, models.Image), nil
	default:
		return nil, nil, nil, fmt.Errorf("unknown platform %q", spec.Platform)
	}
}
