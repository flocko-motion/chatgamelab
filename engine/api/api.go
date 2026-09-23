package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"engine"
	"engine/internal/player"
)

// Package api is the engine's HTTP transport: ordinary handler methods the
// platform mounts in its own router, and a subtree a standalone build serves
// directly.
//
// API holds the engine's HTTP handlers. They are ordinary
// func(http.ResponseWriter, *http.Request) methods, so the platform registers
// them in its own router, with its own paths, middleware and swagger
// annotations, while a standalone build registers the same methods on a plain
// mux. One transport implementation either way.
//
// The engine still knows nothing about users: these handlers deal in session
// ids and engine calls. Everything before the route — authentication, roles,
// share resolution — belongs to whoever mounts them.
type API struct {
	mu       sync.RWMutex
	sessions map[string]*engine.Session
}

func New() *API { return &API{sessions: map[string]*engine.Session{}} }

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

// Register makes a launched session reachable over HTTP. The platform calls
// this after its own Launch route has run the cascade and built the spec.
func (a *API) Register(id string, s *engine.Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[id] = s
}

func (a *API) lookup(r *http.Request) (*engine.Session, string, bool) {
	id := r.PathValue("id")
	a.mu.RLock()
	defer a.mu.RUnlock()
	s, ok := a.sessions[id]
	return s, id, ok
}

// SessionStream is the session-scoped event stream. A turn is a bracketed span
// within it rather than a stream of its own.
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

// SessionInput submits one player action: typed for a turn-based genre, spoken
// for a live one. The genre decides which it accepts.
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

// SessionGraph renders the running wiring. Useful to a dev launcher, and the
// reason a genre's event schema is inspectable rather than documented.
func (a *API) SessionGraph(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown session %q", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, s.Describe())
}

// Handler is the engine's whole HTTP subtree, mounted at one prefix by whoever
// runs it. Owning the subtree is what lets the embedded player address the
// engine with relative URLs — the same ones in the monolith and standalone —
// instead of being told its base path at runtime.
//
// Mount it behind the platform's middleware; everything before the prefix
// (authentication, roles, share resolution) stays the platform's business.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sessions/{id}/stream", a.SessionStream)
	mux.HandleFunc("POST /sessions/{id}/input", a.SessionInput)
	mux.HandleFunc("GET /sessions/{id}/graph", a.SessionGraph)
	mux.HandleFunc("GET /sessions/{id}/live", a.SessionLive)
	mux.Handle("GET /player/", http.StripPrefix("/player/", http.FileServerFS(player.FS())))
	return mux
}
