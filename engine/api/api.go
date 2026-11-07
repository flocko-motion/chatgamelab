package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"engine"
	"engine/web"
)

// Package api is the engine's HTTP transport, mounted by the platform's router or served standalone.
// It deals only in sessions and the catalogue; auth belongs to whoever mounts it.
type API struct {
	mu       sync.RWMutex
	sessions map[string]*engine.Session
	// only is a standalone server's single session, which every request reaches whatever id it names.
	only string
}

func New() *API { return &API{sessions: map[string]*engine.Session{}} }

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

// Register makes a launched session reachable over HTTP.
func (a *API) Register(id string, s *engine.Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[id] = s
}

// RegisterOnly is for a standalone single-game server, so links and bookmarks need no session id.
func (a *API) RegisterOnly(id string, s *engine.Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[id] = s
	a.only = id
}

func (a *API) lookup(r *http.Request) (*engine.Session, string, bool) {
	id := r.PathValue("id")
	a.mu.RLock()
	defer a.mu.RUnlock()
	if s, ok := a.sessions[id]; ok {
		return s, id, true
	}
	if a.only != "" {
		return a.sessions[a.only], id, true
	}
	return nil, id, false
}

// SessionStream is the session-scoped event stream.
func (a *API) SessionStream(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
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
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
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
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
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
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
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
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Prompts())
}

// SessionGraph renders the running wiring as text.
func (a *API) SessionGraph(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, s.Describe())
}

// SessionTopology is the wiring as data, for a view that draws it.
func (a *API) SessionTopology(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Topology())
}

// SessionFlowchart is the same wiring as mermaid.
func (a *API) SessionFlowchart(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, s.Mermaid())
}

// SessionSnapshot is where the session stands now; a reloading page fetches it with the history.
func (a *API) SessionSnapshot(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Snapshot())
}

// SessionHistory is the conversation so far, in order.
func (a *API) SessionHistory(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.History())
}

// SessionNode is what one block is and has been doing.
func (a *API) SessionNode(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}

	node := r.PathValue("node")
	detail, found := s.Inspect(node)
	if !found {
		http.Error(w, fmt.Sprintf("session %q has no block %q", id, node), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

// SessionEdge is what one wire has carried.
func (a *API) SessionEdge(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
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

// Platforms is the engine's catalogue of AI platforms; its JSON shape is the engine's.
func (a *API) Platforms(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(engine.Platforms())
}

// Handler is the engine's whole HTTP subtree, so the embedded player uses the same relative URLs everywhere.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /platforms", a.Platforms)
	mux.HandleFunc("GET /sessions/{id}/stream", a.SessionStream)
	mux.HandleFunc("POST /sessions/{id}/input", a.SessionInput)
	mux.HandleFunc("POST /sessions/{id}/connect", a.SessionConnect)
	mux.HandleFunc("POST /sessions/{id}/pause", a.SessionPause)
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
