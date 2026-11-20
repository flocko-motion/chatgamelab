// Package api is the engine's HTTP transport: one schemaf endpoint per route,
// under /api/engine/ in the platform and in the dev server alike. It deals only
// in sessions; auth belongs to whoever serves it.
package api

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine"
)

// API holds the sessions the transport serves and how it makes and restores them.
type API struct {
	mu       sync.RWMutex
	sessions map[string]*engine.Session
	// restoring holds a lazy restore in flight, so concurrent requests share it.
	restoring map[string]*pending
	// creating holds creations by one-time key, so a repeat shares the first.
	creating map[string]*pending
	opts     Options
}

// Options configures the transport for whoever serves it.
type Options struct {
	// Dev says, per call, whether the catalogue lists the mock platform; a
	// production platform leaves it out, as does a nil Dev.
	Dev func() bool
	// Create does the work of POST /sessions; without it the route is absent.
	Create CreateFunc
	// Resume loads a session that is not in memory, such as after a restart;
	// without it such a session is unknown.
	Resume ResumeFunc
	// Caller names who is asking, such as the signed-in account, from the
	// request's context, so a one-time key answers only the caller who made
	// it. Without it every caller is one.
	Caller func(ctx context.Context) string
}

// CreateFunc launches a session for the playback-engine's request. The
// context is the request's, carrying whatever the serving side put there; the
// session must be launched on one that outlives it, such as
// context.WithoutCancel's. An error that is an *engine.Refusal reaches the
// playback-engine with its problem.
type CreateFunc func(ctx context.Context, req CreateRequest) (*engine.Session, error)

// ResumeFunc loads a stored session by its ID, through engine.Resume, under
// the same contexts as CreateFunc.
type ResumeFunc func(ctx context.Context, id string) (*engine.Session, error)

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

// New makes the transport for whoever serves it.
func New(opts Options) *API {
	return &API{
		sessions:  map[string]*engine.Session{},
		restoring: map[string]*pending{},
		creating:  map[string]*pending{},
		opts:      opts,
	}
}

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

// Close ends every session in memory, each once its last checkpoint is
// written, and drops them all, as when the platform shuts down: call it while
// the store still takes writes. A later request restores a session from the
// store.
func (a *API) Close() {
	a.mu.Lock()
	sessions := a.sessions
	a.sessions = map[string]*engine.Session{}
	a.mu.Unlock()
	var wg sync.WaitGroup
	for _, s := range sessions {
		wg.Go(s.Close)
	}
	wg.Wait()
}

// PlatformList is the AI platforms this transport offers: the mock platform in dev only.
func (a *API) PlatformList() []engine.PlatformInfo {
	platforms := engine.Platforms()
	if a.opts.Dev == nil || !a.opts.Dev() {
		platforms = slices.DeleteFunc(platforms, func(p engine.PlatformInfo) bool { return p.ID == engine.PlatformMock })
	}
	return platforms
}

// Session finds a session by its ID, restoring it if it is stored but not in
// memory. Without one, the error is the refusal to answer with.
func (a *API) Session(ctx context.Context, id string) (*engine.Session, error) {
	s, err := a.find(ctx, id)
	if err != nil {
		return nil, Refusal(err)
	}
	return s, nil
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

// create starts a session through the serving side's CreateFunc. A repeat of
// a one-time key within OnceWindow answers the session the first created, to
// the same caller only; anyone else is refused and learns nothing.
func (a *API) create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if req.Once == "" || (req.Game == "") == (req.Share == "") {
		return CreateResponse{}, badRequest("a creation names a game or a share, and carries a one-time key")
	}

	key := req.Once + "\x00" + req.Game + "\x00" + req.Share
	caller := ""
	if a.opts.Caller != nil {
		caller = a.opts.Caller(ctx)
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
		return CreateResponse{}, Refusal(engine.Refuse(engine.ProblemDenied, "this one-time key belongs to another caller"))
	}

	if !repeat {
		p.session, p.err = a.opts.Create(ctx, req)
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
		return CreateResponse{}, Refusal(p.err)
	}
	return CreateResponse{ID: p.session.ID()}, nil
}

// Refusal is the error an endpoint answers for err, read as a refusal: its
// problem is the error's code, with the problem's status.
// An error that names no problem is ProblemFailed.
func Refusal(err error) error {
	refusal := engine.RefusalOf(err)
	return problemError(refusal.Problem, refusal.Message)
}

// rejected is the error an endpoint answers when a session turns an action
// down: a refusal keeps its problem, anything else is unprocessable.
func rejected(err error) error {
	if refusal := engine.RefusalOf(err); refusal.Problem != engine.ProblemFailed {
		return problemError(refusal.Problem, refusal.Message)
	}
	return schemafapi.Error(schemafapi.ErrUnprocessable, "unprocessable", err.Error())
}

func problemError(problem engine.Problem, message string) error {
	code := string(problem)
	switch problem {
	case engine.ProblemUnknown:
		return schemafapi.Error(schemafapi.ErrNotFound, code, message)
	case engine.ProblemDenied:
		return schemafapi.Error(schemafapi.ErrForbidden, code, message)
	case engine.ProblemStale:
		return schemafapi.Error(schemafapi.ErrGone, code, message)
	case engine.ProblemUnreachable:
		return schemafapi.Error(schemafapi.ErrBadGateway, code, message)
	case engine.ProblemFailed:
		// No sentinel: schemaf answers 500, withholds the message and keeps the code.
		return failed{message}
	default:
		return schemafapi.Error(schemafapi.ErrUnprocessable, code, message)
	}
}

// failed is a refusal for ProblemFailed, a 500 with its code.
type failed struct{ message string }

func (f failed) Error() string { return f.message }
func (f failed) Code() string  { return string(engine.ProblemFailed) }

// badRequest is a request schemaf decoded but the engine cannot act on.
func badRequest(message string) error {
	return schemafapi.Error(schemafapi.ErrBadRequest, "bad_request", message)
}

// Route is one of the engine's routes, for comparing them with another server's.
type Route struct {
	Method, Path string
}

// Routes is every route the engine serves, the creation included.
func Routes() []Route {
	var out []Route
	for _, r := range New(Options{Create: func(context.Context, CreateRequest) (*engine.Session, error) { return nil, nil }}).SchemafRoutes() {
		out = append(out, Route{Method: r.Method, Path: r.Path})
	}
	return out
}

// SchemafRoutes is the engine's endpoints as schemaf routes, each running
// schemaf's whole pipeline on a plain http.ServeMux:
//
//	mux.Handle(r.Method+" "+r.Path, r.Handler)
func (a *API) SchemafRoutes() []schemafapi.Route {
	var routes []schemafapi.Route
	if a.opts.Create != nil {
		routes = append(routes, schemafapi.NewRoute(CreateEndpoint{a}, "", ""))
	}
	return append(routes,
		schemafapi.NewRoute(InputEndpoint{a}, "", ""),
		schemafapi.NewRoute(ChooseEndpoint{a}, "", ""),
		schemafapi.NewRoute(TogglesEndpoint{a}, "", ""),
		schemafapi.NewRoute(ConnectEndpoint{a}, "", ""),
		schemafapi.NewRoute(PauseEndpoint{a}, "", ""),
		schemafapi.NewRoute(StateEndpoint{a}, "", ""),
		schemafapi.NewRoute(HistoryEndpoint{a}, "", ""),
		schemafapi.NewRoute(TopologyEndpoint{a}, "", ""),
		schemafapi.NewRoute(OptionsEndpoint{a}, "", ""),
		schemafapi.NewRoute(NodeEndpoint{a}, "", ""),
		schemafapi.NewRoute(EdgeEndpoint{a}, "", ""),
		schemafapi.NewRoute(PromptsEndpoint{a}, "", ""),
		schemafapi.NewRoute(GraphEndpoint{a}, "", ""),
		schemafapi.NewRoute(FlowchartEndpoint{a}, "", ""),
		schemafapi.NewStreamRoute(StreamEndpoint{a}, "", ""),
		schemafapi.NewRawRoute(LiveEndpoint{a}, "", ""),
		schemafapi.NewRawRoute(PlaybackIndexEndpoint{}, "", ""),
		schemafapi.NewRawRoute(PlaybackFileEndpoint{}, "", ""),
		schemafapi.NewRawRoute(PlaybackNestedFileEndpoint{}, "", ""),
	)
}

// Mux serves the routes on a plain http.ServeMux, as the dev server does.
func Mux(routes []schemafapi.Route) *http.ServeMux {
	mux := http.NewServeMux()
	for _, r := range routes {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return mux
}
