// package: api / the engine, mounted in-process
// type:    wiring
// job:     serves the engine's transport under /api/engine/ behind the platform's access policy, and supplies its create and resume functions
// limits:  session creation and resume logic live in game; the engine owns its handlers and their JSON (-> engine/api)
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"cgl/db"
	"cgl/game"
	"cgl/log"
	"cgl/obj"
	"engine"
	engineapi "engine/api"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// enginePrefix is where the engine's subtree is mounted.
const enginePrefix = "/api/engine"

// Engine is the engine's HTTP transport; engineHandler serves it with the
// access policy in front.
var (
	Engine        *engineapi.API
	engineHandler http.Handler
)

func init() {
	Engine = engineapi.New(engineapi.Options{
		Dev:    devMode,
		Create: createSession,
		Resume: resumeSession,
		Caller: engineCaller,
	})
	engineHandler = http.StripPrefix(enginePrefix, guardSessions(Engine.Handler()))
}

// EngineGetEndpoint passes GET requests under /api/engine/ to the engine.
//
// The playback-engine, its session stream (WebSocket included), its state and
// its lists. Accounts and guests alike; a request naming a session passes the
// access policy first.
type EngineGetEndpoint struct{}

func (EngineGetEndpoint) Method() string             { return "GET" }
func (EngineGetEndpoint) Path() string               { return "/api/engine/{rest...}" }
func (EngineGetEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineGetEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	return serveEngine(w, r)
}

// EnginePostEndpoint passes POST requests under /api/engine/ to the engine.
//
// Creating a session from a game or a share token, and a session's input,
// choices, toggles, connect and pause. Accounts and guests alike; a request
// naming a session passes the access policy first. Creating counts towards the
// token lock, as it may look up a share token.
type EnginePostEndpoint struct{}

func (EnginePostEndpoint) Method() string             { return "POST" }
func (EnginePostEndpoint) Path() string               { return "/api/engine/{rest...}" }
func (EnginePostEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EnginePostEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path == enginePrefix+"/sessions" {
		if err := tokenLocked(); err != nil {
			return err
		}
	}
	return serveEngine(w, r)
}

// serveEngine resolves the caller, if any, and passes the request on.
func serveEngine(w http.ResponseWriter, r *http.Request) error {
	user, err := maybeCaller(r.Context())
	if err != nil {
		return err
	}
	engineHandler.ServeHTTP(w, r.WithContext(withPlayer(r.Context(), user)))
	return nil
}

// PlatformsResp is the engine's AI platforms.
type PlatformsResp = []engine.PlatformInfo

// GenresResp is the engine's genres.
type GenresResp = []engine.GenreInfo

// ThemesResp is the playback-engine's looks.
type ThemesResp = []engine.ThemeInfo

// EngineListReq takes no parameters.
type EngineListReq struct{}

// ListPlatformsEndpoint lists the engine's AI platforms.
//
// With their toggles, playable genres and tiers.
type ListPlatformsEndpoint struct{}

func (ListPlatformsEndpoint) Method() string             { return "GET" }
func (ListPlatformsEndpoint) Path() string               { return "/api/platforms" }
func (ListPlatformsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListPlatformsEndpoint) Handle(ctx context.Context, _ EngineListReq) (PlatformsResp, error) {
	if _, err := caller(ctx); err != nil {
		return nil, err
	}
	return Engine.PlatformList(), nil
}

// ListGenresEndpoint lists every genre the engine runs.
//
// With the toggles each needs, for the game editor.
type ListGenresEndpoint struct{}

func (ListGenresEndpoint) Method() string             { return "GET" }
func (ListGenresEndpoint) Path() string               { return "/api/genres" }
func (ListGenresEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGenresEndpoint) Handle(ctx context.Context, _ EngineListReq) (GenresResp, error) {
	if _, err := caller(ctx); err != nil {
		return nil, err
	}
	return engine.GenreList(), nil
}

// ListThemesEndpoint lists every look the playback-engine can wear.
//
// The default first, for the game editor.
type ListThemesEndpoint struct{}

func (ListThemesEndpoint) Method() string             { return "GET" }
func (ListThemesEndpoint) Path() string               { return "/api/themes" }
func (ListThemesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListThemesEndpoint) Handle(ctx context.Context, _ EngineListReq) (ThemesResp, error) {
	if _, err := caller(ctx); err != nil {
		return nil, err
	}
	return engine.Themes(), nil
}

// ctxKeyPlayer carries the signed-in account through the engine's handlers.
type ctxKeyPlayer struct{}

func withPlayer(ctx context.Context, user *obj.User) context.Context {
	if user == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyPlayer{}, user)
}

// playerFrom is the signed-in account behind an engine request, or nil for a guest.
func playerFrom(ctx context.Context) *obj.User {
	user, _ := ctx.Value(ctxKeyPlayer{}).(*obj.User)
	return user
}

// createSession is the platform's work behind the engine's POST /sessions:
// a game for a signed-in player, or a share link for a guest or a signed-in
// player. The session outlives the request.
func createSession(ctx context.Context, req engineapi.CreateRequest) (*engine.Session, error) {
	player := playerFrom(ctx)
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
	if user := playerFrom(r.Context()); user != nil {
		return "user:" + user.ID.String()
	}
	return ""
}

// guardSessions applies the access policy to every request naming a session:
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
			actor := playerFrom(r.Context())
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
