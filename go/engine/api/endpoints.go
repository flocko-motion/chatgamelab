package api

import (
	"context"
	"fmt"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine"
)

// CreateRequest is what the playback-engine sends to start a session.
type CreateRequest struct {
	// Game or Share names what to play: a game, or a share token.
	Game  string `json:"game,omitempty"`
	Share string `json:"share,omitempty"`
	// Language is the player's, as the playback-engine knows it.
	Language string `json:"language,omitempty"`
	// Once is a key the playback-engine makes for one intent to start, so a
	// repeat within OnceWindow answers the same session.
	Once string `json:"once"`
}

// CreateResponse names the session created.
type CreateResponse struct {
	ID string `json:"id"`
}

// SessionRequest names a session.
type SessionRequest struct {
	ID string `path:"id"`
}

// Accepted says the session took the request; what follows comes on its stream.
type Accepted struct{}

// CreateEndpoint starts a session from a game or a share token.
//
// A repeat of the one-time key within ten minutes answers the session the
// first created, to the same caller only.
type CreateEndpoint struct{ API *API }

func (CreateEndpoint) Method() string             { return "POST" }
func (CreateEndpoint) Path() string               { return "/api/engine/sessions" }
func (CreateEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e CreateEndpoint) Handle(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	return e.API.create(ctx, req)
}

// InputRequest is one player action, typed or spoken.
type InputRequest struct {
	ID    string `path:"id"`
	Text  string `json:"text,omitempty"`
	Audio []byte `json:"audio,omitempty"`
}

// InputEndpoint submits one player action, typed or spoken as the genre accepts.
type InputEndpoint struct{ API *API }

func (InputEndpoint) Method() string             { return "POST" }
func (InputEndpoint) Path() string               { return "/api/engine/sessions/{id}/input" }
func (InputEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e InputEndpoint) Handle(ctx context.Context, req InputRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	switch {
	case req.Text != "":
		err = s.Say(req.Text)
	case len(req.Audio) > 0:
		err = s.Speak(req.Audio)
	default:
		return Accepted{}, badRequest("the input carries neither text nor audio")
	}
	if err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// ChooseRequest is the player's choice among the session's offers.
type ChooseRequest struct {
	ID     string         `path:"id"`
	Offer  string         `json:"offer"`
	Choice *engine.Choice `json:"choice,omitempty"`
}

// ChooseEndpoint takes the player's choice; one outside the offer's caveats is refused.
type ChooseEndpoint struct{ API *API }

func (ChooseEndpoint) Method() string             { return "POST" }
func (ChooseEndpoint) Path() string               { return "/api/engine/sessions/{id}/choose" }
func (ChooseEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e ChooseEndpoint) Handle(ctx context.Context, req ChooseRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	if err := s.Choose(req.Offer, req.Choice); err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// ToggleRequest switches one feature on or off.
type ToggleRequest struct {
	ID     string        `path:"id"`
	Toggle engine.Toggle `json:"toggle"`
	On     bool          `json:"on"`
}

// TogglesEndpoint switches a feature during play, such as pictures off.
type TogglesEndpoint struct{ API *API }

func (TogglesEndpoint) Method() string             { return "POST" }
func (TogglesEndpoint) Path() string               { return "/api/engine/sessions/{id}/toggles" }
func (TogglesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e TogglesEndpoint) Handle(ctx context.Context, req ToggleRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	if err := s.SetToggle(req.Toggle, req.On); err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// ConstraintFrequencyRequest sets how often a block's calls carry the
// youth-protection rule.
type ConstraintFrequencyRequest struct {
	ID string `path:"id"`
	// Block names a threaded block, such as "expand".
	Block string `json:"block"`
	// Frequency is 1 for every call, up to 10 for every tenth.
	Frequency int `json:"frequency"`
}

// ConstraintFrequencyEndpoint changes a block's constraint frequency during
// play; the session keeps the change.
type ConstraintFrequencyEndpoint struct{ API *API }

func (ConstraintFrequencyEndpoint) Method() string { return "POST" }
func (ConstraintFrequencyEndpoint) Path() string {
	return "/api/engine/sessions/{id}/constraint-frequency"
}
func (ConstraintFrequencyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e ConstraintFrequencyEndpoint) Handle(ctx context.Context, req ConstraintFrequencyRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	if err := s.SetConstraintFrequency(req.Block, req.Frequency); err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// ConnectRequest carries the browser's offer to talk to the character.
type ConnectRequest struct {
	ID string `path:"id"`
	// SDP is the browser's offer; answering it creates the billed conversation.
	SDP string `json:"sdp"`
}

// ConnectResponse is the provider's answer to the browser's offer.
type ConnectResponse struct {
	SDP string `json:"sdp"`
}

// ConnectEndpoint brokers the browser's offer to talk to the character.
//
// The provider's id stays server-side, keeping a spendable key and a
// steerable conversation off the client.
type ConnectEndpoint struct{ API *API }

func (ConnectEndpoint) Method() string             { return "POST" }
func (ConnectEndpoint) Path() string               { return "/api/engine/sessions/{id}/connect" }
func (ConnectEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e ConnectEndpoint) Handle(ctx context.Context, req ConnectRequest) (ConnectResponse, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return ConnectResponse{}, err
	}
	if req.SDP == "" {
		return ConnectResponse{}, badRequest("the request carries no SDP offer")
	}
	answer, err := s.Connect(ctx, req.SDP)
	if err != nil {
		return ConnectResponse{}, rejected(err)
	}
	return ConnectResponse{SDP: answer}, nil
}

// PauseEndpoint lets a live conversation go at the player's request, as idling would.
type PauseEndpoint struct{ API *API }

func (PauseEndpoint) Method() string             { return "POST" }
func (PauseEndpoint) Path() string               { return "/api/engine/sessions/{id}/pause" }
func (PauseEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e PauseEndpoint) Handle(ctx context.Context, req SessionRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	if err := s.Pause(); err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// StateEndpoint is where the session stands now.
//
// A reloading page fetches it with the history.
type StateEndpoint struct{ API *API }

func (StateEndpoint) Method() string             { return "GET" }
func (StateEndpoint) Path() string               { return "/api/engine/sessions/{id}/state" }
func (StateEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e StateEndpoint) Handle(ctx context.Context, req SessionRequest) (engine.Snapshot, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return engine.Snapshot{}, err
	}
	return s.Snapshot(), nil
}

// History is the conversation so far, in order.
type History = []engine.SessionEvent

// HistoryEndpoint is the conversation so far, in order.
type HistoryEndpoint struct{ API *API }

func (HistoryEndpoint) Method() string             { return "GET" }
func (HistoryEndpoint) Path() string               { return "/api/engine/sessions/{id}/history" }
func (HistoryEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e HistoryEndpoint) Handle(ctx context.Context, req SessionRequest) (History, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return s.History(), nil
}

// TopologyEndpoint is the session's wiring as data, for a view that draws it.
type TopologyEndpoint struct{ API *API }

func (TopologyEndpoint) Method() string             { return "GET" }
func (TopologyEndpoint) Path() string               { return "/api/engine/sessions/{id}/topology" }
func (TopologyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e TopologyEndpoint) Handle(ctx context.Context, req SessionRequest) (engine.Topology, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return engine.Topology{}, err
	}
	t := s.Topology()
	t.Dev = e.API.opts.Dev != nil && e.API.opts.Dev()
	return t, nil
}

// OptionsEndpoint is what the player may choose from, and what the session settled on.
type OptionsEndpoint struct{ API *API }

func (OptionsEndpoint) Method() string             { return "GET" }
func (OptionsEndpoint) Path() string               { return "/api/engine/sessions/{id}/options" }
func (OptionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e OptionsEndpoint) Handle(ctx context.Context, req SessionRequest) (engine.Options, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return engine.Options{}, err
	}
	return s.Options(), nil
}

// NodeRequest names one block of a session.
type NodeRequest struct {
	ID   string `path:"id"`
	Node string `path:"node"`
}

// NodeEndpoint is what one block is and has been doing.
type NodeEndpoint struct{ API *API }

func (NodeEndpoint) Method() string             { return "GET" }
func (NodeEndpoint) Path() string               { return "/api/engine/sessions/{id}/nodes/{node}" }
func (NodeEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e NodeEndpoint) Handle(ctx context.Context, req NodeRequest) (engine.NodeDetail, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return engine.NodeDetail{}, err
	}
	detail, found := s.Inspect(req.Node)
	if !found {
		return engine.NodeDetail{}, schemafapi.Error(schemafapi.ErrNotFound, "not_found",
			fmt.Sprintf("session %q has no block %q", req.ID, req.Node))
	}
	return detail, nil
}

// CompactEndpoint asks a block's thread to compact now, between calls: it
// summarises the thread whatever its fill, the block working meanwhile and
// paying for the summary. A block on no thread is refused.
type CompactEndpoint struct{ API *API }

func (CompactEndpoint) Method() string { return "POST" }
func (CompactEndpoint) Path() string {
	return "/api/engine/sessions/{id}/nodes/{node}/compact"
}
func (CompactEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e CompactEndpoint) Handle(ctx context.Context, req NodeRequest) (Accepted, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return Accepted{}, err
	}
	if err := s.Compact(req.Node); err != nil {
		return Accepted{}, rejected(err)
	}
	return Accepted{}, nil
}

// EdgeRequest names one wire of a session by its ends and kind.
type EdgeRequest struct {
	ID   string `path:"id"`
	From string `path:"from"`
	To   string `path:"to"`
	Kind string `path:"kind"`
}

// EdgeEndpoint is what one wire has carried.
type EdgeEndpoint struct{ API *API }

func (EdgeEndpoint) Method() string             { return "GET" }
func (EdgeEndpoint) Path() string               { return "/api/engine/sessions/{id}/edges/{from}/{to}/{kind}" }
func (EdgeEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e EdgeEndpoint) Handle(ctx context.Context, req EdgeRequest) (engine.EdgeDetail, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return engine.EdgeDetail{}, err
	}
	detail, found := s.InspectEdge(req.From, req.To, req.Kind)
	if !found {
		return engine.EdgeDetail{}, schemafapi.Error(schemafapi.ErrNotFound, "not_found", "the session has no such wire")
	}
	return detail, nil
}

// Prompts are the text a session runs on, by block.
type Prompts = map[string]string

// PromptsEndpoint is the text a session is running on.
type PromptsEndpoint struct{ API *API }

func (PromptsEndpoint) Method() string             { return "GET" }
func (PromptsEndpoint) Path() string               { return "/api/engine/sessions/{id}/prompts" }
func (PromptsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e PromptsEndpoint) Handle(ctx context.Context, req SessionRequest) (Prompts, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return s.Prompts(), nil
}

// GraphResponse is the running wiring as text.
type GraphResponse struct {
	Graph string `json:"graph"`
}

// GraphEndpoint renders the running wiring as text.
type GraphEndpoint struct{ API *API }

func (GraphEndpoint) Method() string             { return "GET" }
func (GraphEndpoint) Path() string               { return "/api/engine/sessions/{id}/graph" }
func (GraphEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e GraphEndpoint) Handle(ctx context.Context, req SessionRequest) (GraphResponse, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return GraphResponse{}, err
	}
	return GraphResponse{Graph: s.Describe()}, nil
}

// FlowchartResponse is the running wiring as a mermaid flowchart.
type FlowchartResponse struct {
	Flowchart string `json:"flowchart"`
}

// FlowchartEndpoint is the running wiring as a mermaid flowchart.
type FlowchartEndpoint struct{ API *API }

func (FlowchartEndpoint) Method() string             { return "GET" }
func (FlowchartEndpoint) Path() string               { return "/api/engine/sessions/{id}/flowchart" }
func (FlowchartEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e FlowchartEndpoint) Handle(ctx context.Context, req SessionRequest) (FlowchartResponse, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return FlowchartResponse{}, err
	}
	return FlowchartResponse{Flowchart: s.Mermaid()}, nil
}

// StreamEndpoint is the session's events as server-sent events.
//
// Each event's data is one SessionEvent; its Stream names the kind.
type StreamEndpoint struct{ API *API }

func (StreamEndpoint) Method() string             { return "GET" }
func (StreamEndpoint) Path() string               { return "/api/engine/sessions/{id}/stream" }
func (StreamEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e StreamEndpoint) Stream(ctx context.Context, req SessionRequest) (<-chan engine.SessionEvent, error) {
	s, err := e.API.Session(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	out := make(chan engine.SessionEvent)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, open := <-s.Events():
				if !open {
					return
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
