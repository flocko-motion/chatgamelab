package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"engine"
	"engine/web"
)

// Package api is the engine's HTTP transport, mounted by the platform's router or served standalone.
// It deals only in sessions and the catalogue; auth belongs to whoever mounts it.
type API struct {
	mu       sync.RWMutex
	sessions map[string]*engine.Session
	// restoring holds a lazy restore in flight, so concurrent requests share it.
	restoring map[string]*pending
	// creating holds creations by one-time key, so a repeat shares the first.
	creating map[string]*pending
	opts     Options
}

// Options configures the transport for whoever mounts it.
type Options struct {
	// Dev lists the mock platform in the catalogue; a production platform leaves it out.
	Dev bool
	// Create does the work of POST /sessions; without it the endpoint is absent.
	Create CreateFunc
	// Resume loads a session that is not in memory, such as after a restart;
	// without it such a session is unknown.
	Resume ResumeFunc
	// Caller names who is asking, such as the signed-in account, so a one-time
	// key answers only the caller who made it. Without it every caller is one.
	Caller func(r *http.Request) string
}

// CreateFunc launches a session for the playback-engine's request. The
// context is the request's, carrying whatever the mounting server's middleware
// put there; the session must be launched on one that outlives it, such as
// context.WithoutCancel's. An error that is an *engine.Refusal reaches the
// playback-engine with its problem.
type CreateFunc func(ctx context.Context, req CreateRequest) (*engine.Session, error)

// ResumeFunc loads a stored session by its ID, through engine.Resume, under
// the same contexts as CreateFunc.
type ResumeFunc func(ctx context.Context, id string) (*engine.Session, error)

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

// OnceWindow is how long a one-time key answers the session it created.
const OnceWindow = 10 * time.Minute

// pending is a creation or restore in flight, then its outcome.
type pending struct {
	done    chan struct{}
	session *engine.Session
	err     error
	at      time.Time
	// caller made the creation; a repeat from anyone else is refused.
	caller string
}

// New makes the transport for whoever mounts it.
func New(opts Options) *API {
	return &API{
		sessions:  map[string]*engine.Session{},
		restoring: map[string]*pending{},
		creating:  map[string]*pending{},
		opts:      opts,
	}
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

// Register makes a launched session reachable over HTTP.
func (a *API) Register(id string, s *engine.Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[id] = s
}

// Forget closes a session and drops it from memory, as when the platform
// deletes it; a later request restores it only if it is still stored.
func (a *API) Forget(id string) {
	a.mu.Lock()
	s, ok := a.sessions[id]
	delete(a.sessions, id)
	a.mu.Unlock()
	if ok {
		s.Close()
	}
}

// session finds the request's session, restoring it if it is stored but not
// in memory, and answers the refusal itself when there is none.
func (a *API) session(w http.ResponseWriter, r *http.Request) (*engine.Session, bool) {
	s, err := a.find(r.Context(), r.PathValue("id"))
	if err != nil {
		refuse(w, err)
		return nil, false
	}
	return s, true
}

func (a *API) find(ctx context.Context, id string) (*engine.Session, error) {
	a.mu.Lock()
	if s, ok := a.sessions[id]; ok {
		a.mu.Unlock()
		return s, nil
	}
	if a.opts.Resume == nil {
		a.mu.Unlock()
		return nil, engine.Refuse(engine.ProblemUnknown, fmt.Sprintf("there is no session %q", id))
	}
	p, inFlight := a.restoring[id]
	if !inFlight {
		p = &pending{done: make(chan struct{})}
		a.restoring[id] = p
	}
	a.mu.Unlock()

	if !inFlight {
		p.session, p.err = a.opts.Resume(ctx, id)
		a.mu.Lock()
		delete(a.restoring, id)
		if p.err == nil {
			a.sessions[id] = p.session
		}
		a.mu.Unlock()
		close(p.done)
	}
	<-p.done
	return p.session, p.err
}

// Create starts a session through the mounting server's CreateFunc. A repeat
// of a one-time key within OnceWindow answers the session the first created,
// to the same caller only; anyone else is refused and learns nothing.
func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Once == "" || (req.Game == "") == (req.Share == "") {
		http.Error(w, "a creation names a game or a share, and carries a one-time key", http.StatusBadRequest)
		return
	}

	key := req.Once + "\x00" + req.Game + "\x00" + req.Share
	caller := ""
	if a.opts.Caller != nil {
		caller = a.opts.Caller(r)
	}
	now := time.Now()
	a.mu.Lock()
	for k, p := range a.creating {
		if now.Sub(p.at) > OnceWindow {
			delete(a.creating, k)
		}
	}
	p, repeat := a.creating[key]
	if !repeat {
		p = &pending{done: make(chan struct{}), at: now, caller: caller}
		a.creating[key] = p
	}
	a.mu.Unlock()
	if repeat && p.caller != caller {
		refuse(w, engine.Refuse(engine.ProblemDenied, "this one-time key belongs to another caller"))
		return
	}

	if !repeat {
		p.session, p.err = a.opts.Create(r.Context(), req)
		a.mu.Lock()
		if p.err == nil {
			a.sessions[p.session.ID()] = p.session
		} else {
			// A failure is not remembered, so trying again tries again.
			delete(a.creating, key)
		}
		a.mu.Unlock()
		close(p.done)
	}
	<-p.done
	if p.err != nil {
		refuse(w, p.err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateResponse{ID: p.session.ID()})
}

// refuse answers an error as a refusal the playback-engine can explain.
func refuse(w http.ResponseWriter, err error) {
	refusal := engine.RefusalOf(err)
	status := http.StatusUnprocessableEntity
	switch refusal.Problem {
	case engine.ProblemUnknown:
		status = http.StatusNotFound
	case engine.ProblemStale:
		status = http.StatusGone
	case engine.ProblemDenied:
		status = http.StatusForbidden
	case engine.ProblemUnreachable:
		status = http.StatusBadGateway
	case engine.ProblemFailed:
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(refusal)
}

// SessionStream is the session-scoped event stream.
func (a *API) SessionStream(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case e, open := <-s.Events():
			if !open {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: ", e.Stream)
			if err := enc.Encode(e); err != nil {
				return
			}
			fmt.Fprint(w, "\n")
			flusher.Flush()
		}
	}
}

type inputRequest struct {
	Text  string `json:"text,omitempty"`
	Audio []byte `json:"audio,omitempty"`
}

// SessionInput submits one player action, typed or spoken as the genre accepts.
func (a *API) SessionInput(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}

	var req inputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	switch {
	case req.Text != "":
		err = s.Say(req.Text)
	case len(req.Audio) > 0:
		err = s.Speak(req.Audio)
	default:
		http.Error(w, "body carries neither text nor audio", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type connectRequest struct {
	// SDP is the browser's offer; answering it creates the billed conversation.
	SDP string `json:"sdp"`
}

type connectResponse struct {
	SDP string `json:"sdp"`
}

// SessionConnect brokers the browser's offer to talk to the character.
// The provider's id stays server-side, keeping a spendable key and a steerable conversation off the client.
func (a *API) SessionConnect(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}

	var req connectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.SDP == "" {
		http.Error(w, "body carries no SDP offer", http.StatusBadRequest)
		return
	}

	answer, err := s.Connect(r.Context(), req.SDP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(connectResponse{SDP: answer})
}

// SessionPause lets a live conversation go at the player's request, as idling would.
func (a *API) SessionPause(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := s.Pause(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// SessionPrompts is the text a session is running on.
func (a *API) SessionPrompts(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Prompts())
}

// SessionGraph renders the running wiring as text.
func (a *API) SessionGraph(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, s.Describe())
}

// SessionTopology is the wiring as data, for a view that draws it.
func (a *API) SessionTopology(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Topology())
}

// SessionFlowchart is the same wiring as mermaid.
func (a *API) SessionFlowchart(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, s.Mermaid())
}

// SessionSnapshot is where the session stands now; a reloading page fetches it with the history.
func (a *API) SessionSnapshot(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Snapshot())
}

// SessionHistory is the conversation so far, in order.
func (a *API) SessionHistory(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.History())
}

// SessionNode is what one block is and has been doing.
func (a *API) SessionNode(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}

	node := r.PathValue("node")
	detail, found := s.Inspect(node)
	if !found {
		http.Error(w, fmt.Sprintf("session %q has no block %q", r.PathValue("id"), node), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

// SessionEdge is what one wire has carried.
func (a *API) SessionEdge(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	detail, found := s.InspectEdge(q.Get("from"), q.Get("to"), q.Get("kind"))
	if !found {
		http.Error(w, "no such edge", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

// SessionOptions is what the player may choose from, and what the session settled on.
func (a *API) SessionOptions(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Options())
}

type chooseRequest struct {
	Offer  string         `json:"offer"`
	Choice *engine.Choice `json:"choice,omitempty"`
}

// SessionChoose takes the player's choice; one outside the offer's caveats is refused.
func (a *API) SessionChoose(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	var req chooseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Choose(req.Offer, req.Choice); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type toggleRequest struct {
	Toggle engine.Toggle `json:"toggle"`
	On     bool          `json:"on"`
}

// SessionToggle switches a feature during play, such as pictures off.
func (a *API) SessionToggle(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.SetToggle(req.Toggle, req.On); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// Platforms is the engine's catalogue of AI platforms; its JSON shape is the engine's.
// The mock is listed in dev mode only.
func (a *API) Platforms(w http.ResponseWriter, _ *http.Request) {
	platforms := engine.Platforms()
	if !a.opts.Dev {
		platforms = slices.DeleteFunc(platforms, func(p engine.PlatformInfo) bool { return p.ID == engine.PlatformMock })
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(platforms)
}

// Themes is every look the playback-engine can wear, for the game editor.
func (a *API) Themes(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(engine.Themes())
}

// Genres is every genre with the toggles it needs, for the game editor.
func (a *API) Genres(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(engine.GenreList())
}

// Handler is the engine's whole HTTP subtree, so the embedded player uses the same relative URLs everywhere.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /platforms", a.Platforms)
	mux.HandleFunc("GET /themes", a.Themes)
	mux.HandleFunc("GET /genres", a.Genres)
	if a.opts.Create != nil {
		mux.HandleFunc("POST /sessions", a.Create)
	}
	mux.HandleFunc("GET /sessions/{id}/stream", a.SessionStream)
	mux.HandleFunc("POST /sessions/{id}/input", a.SessionInput)
	mux.HandleFunc("POST /sessions/{id}/connect", a.SessionConnect)
	mux.HandleFunc("POST /sessions/{id}/pause", a.SessionPause)
	mux.HandleFunc("GET /sessions/{id}/options", a.SessionOptions)
	mux.HandleFunc("POST /sessions/{id}/choose", a.SessionChoose)
	mux.HandleFunc("POST /sessions/{id}/toggles", a.SessionToggle)
	mux.HandleFunc("GET /sessions/{id}/prompts", a.SessionPrompts)
	mux.HandleFunc("GET /sessions/{id}/graph", a.SessionGraph)
	mux.HandleFunc("GET /sessions/{id}/topology", a.SessionTopology)
	mux.HandleFunc("GET /sessions/{id}/state", a.SessionSnapshot)
	mux.HandleFunc("GET /sessions/{id}/history", a.SessionHistory)
	mux.HandleFunc("GET /sessions/{id}/nodes/{node}", a.SessionNode)
	mux.HandleFunc("GET /sessions/{id}/edge", a.SessionEdge)
	mux.HandleFunc("GET /sessions/{id}/flowchart", a.SessionFlowchart)
	mux.HandleFunc("GET /sessions/{id}/live", a.SessionLive)
	mux.Handle("GET /player/", http.StripPrefix("/player/", http.FileServerFS(web.FS())))
	return mux
}
