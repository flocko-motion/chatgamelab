// package: routes / the engine's HTTP handlers, mounted here
// type:    wiring
// job:     mounts the engine's transport behind the platform's access policy, supplies its create and resume functions, and documents its lists for the OpenAPI spec
// limits:  session creation and resume logic live in game; the engine owns its handlers and their JSON (-> engine/api)
package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/log"
	"engine"
	engineapi "engine/api"

	"github.com/google/uuid"
)

// Engine is the engine's HTTP transport; NewMux makes it and mounts its handlers.
var Engine *engineapi.API

// newEngine makes the transport with the platform's create and resume work.
func newEngine() *engineapi.API {
	return engineapi.New(engineapi.Options{
		Dev:    DevMode,
		Create: createSession,
		Resume: resumeSession,
		Caller: engineCaller,
	})
}

// mountEngine serves the engine's whole subtree under /api/engine/: the
// playback-engine, its sessions and its lists. Every request is authenticated
// by schemaf's session (cookie or Bearer) where there is one, and every request
// naming a session passes the access policy first.
func mountEngine(mux *http.ServeMux) {
	prefix := "/api/engine"
	engineRoutes := httpx.OptionalAuth(http.StripPrefix(prefix, guardSessions(Engine.Handler())).ServeHTTP)
	mux.Handle(prefix+"/", engineRoutes)
	// Creating may look up a share token, which counts towards the token lock.
	mux.Handle("POST "+prefix+"/sessions", httpx.TokenGuard(engineRoutes))
}

// createSession is the platform's work behind the engine's POST /sessions:
// a game for a signed-in player, or a share link for a guest or a signed-in
// player. The session outlives the request.
func createSession(ctx context.Context, req engineapi.CreateRequest) (*engine.Session, error) {
	player := httpx.UserFromContext(ctx)
	lifetime := context.WithoutCancel(ctx)

	var created *game.Created
	var err error
	if req.Share != "" {
		created, err = game.CreateShareSession(lifetime, player, req.Share, req.Language)
	} else {
		gameID, parseErr := uuid.Parse(req.Game)
		if parseErr != nil {
			return nil, engine.Refuse(engine.ProblemDenied, "no such game")
		}
		created, err = game.CreateGameSession(lifetime, player, gameID, req.Language)
	}
	if err != nil {
		return nil, err
	}
	for _, id := range created.Replaced {
		Engine.Forget(id.String())
	}
	return created.Session, nil
}

// resumeSession restores a stored session that is not in memory; the access
// policy has already let the request through.
func resumeSession(ctx context.Context, id string) (*engine.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, engine.Refuse(engine.ProblemUnknown, "there is no session "+id)
	}
	return game.ResumeSession(context.WithoutCancel(ctx), sessionID)
}

// engineCaller names who creates a session, so a one-time key answers only
// its maker: the signed-in account, or nobody for a guest, whose key the
// transport already binds to the share token.
func engineCaller(r *http.Request) string {
	if user := httpx.MaybeUserFromRequest(r); user != nil {
		return "user:" + user.ID.String()
	}
	return ""
}

// guardSessions applies v1's access policy to every request naming a session:
// a session a guest owns is reachable with its id, one a signed-in account
// owns only by that account. A session the platform no longer holds, deleted
// or purged, is dropped from memory and unknown. Inputs the player sends are
// counted as they are accepted.
func guardSessions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := sessionInPath(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		sessionID, err := uuid.Parse(id)
		if err != nil {
			writeRefusal(w, http.StatusNotFound, engine.ProblemUnknown, "there is no session "+id)
			return
		}
		session, err := db.GetGameSession(r.Context(), sessionID)
		if err != nil {
			Engine.Forget(id)
			writeRefusal(w, http.StatusNotFound, engine.ProblemUnknown, "there is no session "+id)
			return
		}
		if !db.IsGuestUser(r.Context(), session.UserID) {
			actor := httpx.MaybeUserFromRequest(r)
			if actor == nil || actor.ID != session.UserID {
				writeRefusal(w, http.StatusForbidden, engine.ProblemDenied, "this session belongs to a signed-in account")
				return
			}
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/input") {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status == http.StatusAccepted {
				if err := db.CountSessionMessage(r.Context(), sessionID); err != nil {
					log.Warn("failed to count a message", "session_id", sessionID, "error", err)
				}
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sessionInPath reads the session id from an engine path /sessions/{id}/….
func sessionInPath(path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, "/sessions/")
	if !ok {
		return "", false
	}
	id, _, _ := strings.Cut(rest, "/")
	return id, id != ""
}

// writeRefusal answers as the engine's transport does, so the playback-engine
// can explain it.
func writeRefusal(w http.ResponseWriter, status int, problem engine.Problem, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(engine.Refusal{Problem: problem, Message: message})
}

// statusRecorder notes the status a handler answered with.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// enginePlatformsDoc documents Engine.Platforms for the OpenAPI spec; swag reads
// annotations only from this module's functions.
//
//	@Summary		List AI platforms
//	@Description	Returns the engine's AI platforms with their toggles, playable genres and tiers
//	@Tags			platforms
//	@Produce		json
//	@Success		200	{array}	engine.PlatformInfo
//	@Router			/platforms [get]
//
//deadcode:keep // documentation only, read by swag
func enginePlatformsDoc() {}

// engineGenresDoc documents Engine.Genres for the OpenAPI spec.
//
//	@Summary		List genres
//	@Description	Returns every genre the engine runs, with the toggles it needs, for the game editor
//	@Tags			games
//	@Produce		json
//	@Success		200	{array}	engine.GenreInfo
//	@Security		BearerAuth
//	@Router			/genres [get]
//
//deadcode:keep // documentation only, read by swag
func engineGenresDoc() {}

// engineThemesDoc documents Engine.Themes for the OpenAPI spec.
//
//	@Summary		List themes
//	@Description	Returns every look the playback-engine can wear, the default first, for the game editor
//	@Tags			games
//	@Produce		json
//	@Success		200	{array}	engine.ThemeInfo
//	@Security		BearerAuth
//	@Router			/themes [get]
//
//deadcode:keep // documentation only, read by swag
func engineThemesDoc() {}
