// package: api / the engine, mounted in-process
// type:    wiring
// job:     declares one endpoint per engine route behind the platform's access policy, and supplies the engine's create and resume functions
// limits:  session creation and resume logic live in game; the engine owns the routes' work and their JSON (-> engine/api)
package api

import (
	"context"
	"net/http"

	"cgl/db"
	"cgl/game"
	"cgl/log"
	"cgl/obj"
	"engine"
	engineapi "engine/api"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// Engine is the engine's transport; the platform's engine endpoints call its endpoints.
var Engine *engineapi.API

func init() {
	Engine = engineapi.New(engineapi.Options{
		Dev:    devMode,
		Create: createSession,
		Resume: resumeSession,
		Caller: engineCaller,
	})
}

// The engine's request and response types, as the platform's endpoints name them.
type (
	EngineCreateReq     = engineapi.CreateRequest
	EngineCreateResp    = engineapi.CreateResponse
	EngineInputReq      = engineapi.InputRequest
	EngineAccepted      = engineapi.Accepted
	EngineChooseReq     = engineapi.ChooseRequest
	EngineToggleReq     = engineapi.ToggleRequest
	EngineConnectReq    = engineapi.ConnectRequest
	EngineConnectResp   = engineapi.ConnectResponse
	EngineSessionReq    = engineapi.SessionRequest
	EngineSnapshot      = engine.Snapshot
	EngineHistory       = engineapi.History
	EngineTopology      = engine.Topology
	EngineOptions       = engine.Options
	EngineNodeReq       = engineapi.NodeRequest
	EngineNodeDetail    = engine.NodeDetail
	EngineEdgeReq       = engineapi.EdgeRequest
	EngineEdgeDetail    = engine.EdgeDetail
	EnginePrompts       = engineapi.Prompts
	EngineGraphResp     = engineapi.GraphResponse
	EngineFlowchartResp = engineapi.FlowchartResponse
	EngineSessionEvent  = engine.SessionEvent
)

// enginePlayer resolves the caller of an engine route, if any, into the
// context the engine's endpoints run in.
func enginePlayer(ctx context.Context) (context.Context, error) {
	user, err := maybeCaller(ctx)
	if err != nil {
		return ctx, err
	}
	return withPlayer(ctx, user), nil
}

// engineSession applies the access policy to the session a request names: a
// session a guest owns is reachable with its id, one a signed-in account owns
// only by that account. A session the platform no longer holds, deleted or
// purged, is dropped from memory and unknown.
func engineSession(ctx context.Context, id string) (context.Context, uuid.UUID, error) {
	ctx, err := enginePlayer(ctx)
	if err != nil {
		return ctx, uuid.Nil, err
	}
	unknown := engineapi.Refusal(engine.Refuse(engine.ProblemUnknown, "there is no session "+id))
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return ctx, uuid.Nil, unknown
	}
	session, err := db.GetGameSession(ctx, sessionID)
	if err != nil {
		Engine.Forget(id)
		return ctx, uuid.Nil, unknown
	}
	if !db.IsGuestUser(ctx, session.UserID) {
		actor := playerFrom(ctx)
		if actor == nil || actor.ID != session.UserID {
			return ctx, uuid.Nil, engineapi.Refusal(engine.Refuse(engine.ProblemDenied, "this session belongs to a signed-in account"))
		}
	}
	return ctx, sessionID, nil
}

// EngineCreateSessionEndpoint starts a session from a game or a share token.
//
// A game for a signed-in player, a share token for a guest or a signed-in
// player. A repeat of the one-time key within ten minutes answers the session
// the first created, to the same caller only. Counts towards the token lock,
// as it may look up a share token.
type EngineCreateSessionEndpoint struct{}

func (EngineCreateSessionEndpoint) Method() string             { return "POST" }
func (EngineCreateSessionEndpoint) Path() string               { return "/api/engine/sessions" }
func (EngineCreateSessionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineCreateSessionEndpoint) Handle(ctx context.Context, req EngineCreateReq) (EngineCreateResp, error) {
	if err := tokenLocked(); err != nil {
		return EngineCreateResp{}, err
	}
	ctx, err := enginePlayer(ctx)
	if err != nil {
		return EngineCreateResp{}, err
	}
	return engineapi.CreateEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineInputEndpoint submits one player action to a session, typed or spoken.
//
// As the genre accepts; an accepted input counts as one of the session's
// messages.
type EngineInputEndpoint struct{}

func (EngineInputEndpoint) Method() string             { return "POST" }
func (EngineInputEndpoint) Path() string               { return "/api/engine/sessions/{id}/input" }
func (EngineInputEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineInputEndpoint) Handle(ctx context.Context, req EngineInputReq) (EngineAccepted, error) {
	ctx, sessionID, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineAccepted{}, err
	}
	accepted, err := engineapi.InputEndpoint{API: Engine}.Handle(ctx, req)
	if err != nil {
		return accepted, err
	}
	if err := db.CountSessionMessage(ctx, sessionID); err != nil {
		log.Warn("failed to count a message", "session_id", sessionID, "error", err)
	}
	return accepted, nil
}

// EngineChooseEndpoint takes the player's choice among a session's offers.
//
// A choice outside the offer's caveats is refused.
type EngineChooseEndpoint struct{}

func (EngineChooseEndpoint) Method() string             { return "POST" }
func (EngineChooseEndpoint) Path() string               { return "/api/engine/sessions/{id}/choose" }
func (EngineChooseEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineChooseEndpoint) Handle(ctx context.Context, req EngineChooseReq) (EngineAccepted, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineAccepted{}, err
	}
	return engineapi.ChooseEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineTogglesEndpoint switches a feature of a session during play.
//
// Such as pictures off; on only as far as the session's offer allows.
type EngineTogglesEndpoint struct{}

func (EngineTogglesEndpoint) Method() string             { return "POST" }
func (EngineTogglesEndpoint) Path() string               { return "/api/engine/sessions/{id}/toggles" }
func (EngineTogglesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineTogglesEndpoint) Handle(ctx context.Context, req EngineToggleReq) (EngineAccepted, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineAccepted{}, err
	}
	return engineapi.TogglesEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineConnectEndpoint brokers the browser's offer to talk to a session's character.
//
// The provider's id stays server-side, keeping a spendable key and a
// steerable conversation off the client.
type EngineConnectEndpoint struct{}

func (EngineConnectEndpoint) Method() string             { return "POST" }
func (EngineConnectEndpoint) Path() string               { return "/api/engine/sessions/{id}/connect" }
func (EngineConnectEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineConnectEndpoint) Handle(ctx context.Context, req EngineConnectReq) (EngineConnectResp, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineConnectResp{}, err
	}
	return engineapi.ConnectEndpoint{API: Engine}.Handle(ctx, req)
}

// EnginePauseEndpoint lets a session's live conversation go at the player's request.
//
// As idling would.
type EnginePauseEndpoint struct{}

func (EnginePauseEndpoint) Method() string             { return "POST" }
func (EnginePauseEndpoint) Path() string               { return "/api/engine/sessions/{id}/pause" }
func (EnginePauseEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EnginePauseEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineAccepted, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineAccepted{}, err
	}
	return engineapi.PauseEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineStateEndpoint answers where a session stands now.
//
// A reloading page fetches it with the history.
type EngineStateEndpoint struct{}

func (EngineStateEndpoint) Method() string             { return "GET" }
func (EngineStateEndpoint) Path() string               { return "/api/engine/sessions/{id}/state" }
func (EngineStateEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineStateEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineSnapshot, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineSnapshot{}, err
	}
	return engineapi.StateEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineHistoryEndpoint answers a session's conversation so far, in order.
type EngineHistoryEndpoint struct{}

func (EngineHistoryEndpoint) Method() string             { return "GET" }
func (EngineHistoryEndpoint) Path() string               { return "/api/engine/sessions/{id}/history" }
func (EngineHistoryEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineHistoryEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineHistory, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return engineapi.HistoryEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineTopologyEndpoint answers a session's wiring as data, for a view that draws it.
type EngineTopologyEndpoint struct{}

func (EngineTopologyEndpoint) Method() string             { return "GET" }
func (EngineTopologyEndpoint) Path() string               { return "/api/engine/sessions/{id}/topology" }
func (EngineTopologyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineTopologyEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineTopology, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineTopology{}, err
	}
	return engineapi.TopologyEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineOptionsEndpoint answers what the player of a session may choose from.
//
// And what the session settled on.
type EngineOptionsEndpoint struct{}

func (EngineOptionsEndpoint) Method() string             { return "GET" }
func (EngineOptionsEndpoint) Path() string               { return "/api/engine/sessions/{id}/options" }
func (EngineOptionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineOptionsEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineOptions, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineOptions{}, err
	}
	return engineapi.OptionsEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineNodeEndpoint answers what one block of a session is and has been doing.
type EngineNodeEndpoint struct{}

func (EngineNodeEndpoint) Method() string             { return "GET" }
func (EngineNodeEndpoint) Path() string               { return "/api/engine/sessions/{id}/nodes/{node}" }
func (EngineNodeEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineNodeEndpoint) Handle(ctx context.Context, req EngineNodeReq) (EngineNodeDetail, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineNodeDetail{}, err
	}
	return engineapi.NodeEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineEdgeEndpoint answers what one wire of a session has carried.
//
// The wire is named by its ends and its kind.
type EngineEdgeEndpoint struct{}

func (EngineEdgeEndpoint) Method() string { return "GET" }
func (EngineEdgeEndpoint) Path() string {
	return "/api/engine/sessions/{id}/edges/{from}/{to}/{kind}"
}
func (EngineEdgeEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineEdgeEndpoint) Handle(ctx context.Context, req EngineEdgeReq) (EngineEdgeDetail, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineEdgeDetail{}, err
	}
	return engineapi.EdgeEndpoint{API: Engine}.Handle(ctx, req)
}

// EnginePromptsEndpoint answers the text a session is running on, by block.
type EnginePromptsEndpoint struct{}

func (EnginePromptsEndpoint) Method() string             { return "GET" }
func (EnginePromptsEndpoint) Path() string               { return "/api/engine/sessions/{id}/prompts" }
func (EnginePromptsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EnginePromptsEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EnginePrompts, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return engineapi.PromptsEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineGraphEndpoint renders a session's running wiring as text.
type EngineGraphEndpoint struct{}

func (EngineGraphEndpoint) Method() string             { return "GET" }
func (EngineGraphEndpoint) Path() string               { return "/api/engine/sessions/{id}/graph" }
func (EngineGraphEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineGraphEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineGraphResp, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineGraphResp{}, err
	}
	return engineapi.GraphEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineFlowchartEndpoint renders a session's running wiring as a mermaid flowchart.
type EngineFlowchartEndpoint struct{}

func (EngineFlowchartEndpoint) Method() string             { return "GET" }
func (EngineFlowchartEndpoint) Path() string               { return "/api/engine/sessions/{id}/flowchart" }
func (EngineFlowchartEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineFlowchartEndpoint) Handle(ctx context.Context, req EngineSessionReq) (EngineFlowchartResp, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return EngineFlowchartResp{}, err
	}
	return engineapi.FlowchartEndpoint{API: Engine}.Handle(ctx, req)
}

// EngineStreamEndpoint streams a session's events as server-sent events.
//
// Each event's data is one SessionEvent; its Stream names the kind.
type EngineStreamEndpoint struct{}

func (EngineStreamEndpoint) Method() string             { return "GET" }
func (EngineStreamEndpoint) Path() string               { return "/api/engine/sessions/{id}/stream" }
func (EngineStreamEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineStreamEndpoint) Stream(ctx context.Context, req EngineSessionReq) (<-chan EngineSessionEvent, error) {
	ctx, _, err := engineSession(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return engineapi.StreamEndpoint{API: Engine}.Stream(ctx, req)
}

// EngineLiveEndpoint opens the duplex WebSocket a session of a voice genre needs.
//
// The browser sends binary frames of microphone audio and text frames of
// typed input, a JSON LiveInput; it receives the session's events back,
// binary frames for audio and text frames of JSON SessionEvent for everything
// else. A rejected input comes back as an `error` event.
type EngineLiveEndpoint struct{}

func (EngineLiveEndpoint) Method() string             { return "GET" }
func (EngineLiveEndpoint) Path() string               { return "/api/engine/sessions/{id}/live" }
func (EngineLiveEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EngineLiveEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	ctx, _, err := engineSession(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return engineapi.LiveEndpoint{API: Engine}.HandleRaw(w, r.WithContext(ctx))
}

// EnginePlaybackIndexEndpoint serves the playback-engine's page.
//
// Opened with ?session=<id> to play a session, or ?new&game=<id> or
// ?new&share=<token> to start one.
type EnginePlaybackIndexEndpoint struct{}

func (EnginePlaybackIndexEndpoint) Method() string             { return "GET" }
func (EnginePlaybackIndexEndpoint) Path() string               { return "/api/engine/playback-engine/" }
func (EnginePlaybackIndexEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (EnginePlaybackIndexEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	return engineapi.PlaybackIndexEndpoint{}.HandleRaw(w, r)
}

// EnginePlaybackFileEndpoint serves a playback-engine file beside its page.
//
// Its scripts, stylesheet and worklet.
type EnginePlaybackFileEndpoint struct{}

func (EnginePlaybackFileEndpoint) Method() string { return "GET" }
func (EnginePlaybackFileEndpoint) Path() string   { return "/api/engine/playback-engine/{file}" }
func (EnginePlaybackFileEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthOptional
}
func (EnginePlaybackFileEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	return engineapi.PlaybackFileEndpoint{}.HandleRaw(w, r)
}

// EnginePlaybackNestedFileEndpoint serves a playback-engine file in a folder.
//
// Its chunks, fonts, art and language files.
type EnginePlaybackNestedFileEndpoint struct{}

func (EnginePlaybackNestedFileEndpoint) Method() string { return "GET" }
func (EnginePlaybackNestedFileEndpoint) Path() string {
	return "/api/engine/playback-engine/{dir}/{file}"
}
func (EnginePlaybackNestedFileEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthOptional
}
func (EnginePlaybackNestedFileEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	return engineapi.PlaybackNestedFileEndpoint{}.HandleRaw(w, r)
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

// ctxKeyPlayer carries the signed-in account through the engine's endpoints.
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
func createSession(ctx context.Context, req EngineCreateReq) (*engine.Session, error) {
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
func engineCaller(ctx context.Context) string {
	if user := playerFrom(ctx); user != nil {
		return "user:" + user.ID.String()
	}
	return ""
}
