// package: api / schemaf endpoints
// type:    logic
// job:     lists the caller's sessions and deletes one
// limits:  decides nothing: the core does (core/session.go); playing them is engine.go's
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// ListUserSessionsEndpoint lists the caller's sessions, newest activity
// first, at most 20. search narrows them to games whose name holds it;
// sortBy "game" or "model" orders them by game name or AI platform.
type ListUserSessionsEndpoint struct{}

func (ListUserSessionsEndpoint) Method() string             { return "GET" }
func (ListUserSessionsEndpoint) Path() string               { return "/api/sessions" }
func (ListUserSessionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListUserSessionsEndpoint) Handle(ctx context.Context, req ListUserSessionsReq) (UserSessionsResp, error) {
	return wire.SessionViewsOf(core.MySessions(ctx, opt(req.Search), opt(req.SortBy)))
}

// ListUserSessionsReq is what the caller's list is narrowed and ordered by.
type ListUserSessionsReq struct {
	Search *string `query:"search"`
	SortBy *string `query:"sortBy"`
}

// UserSessionsResp is the caller's sessions.
type UserSessionsResp = []wire.SessionView

// ListGameSessionsEndpoint lists the caller's sessions of a game they may
// read; another's game they may not read is 404.
type ListGameSessionsEndpoint struct{}

func (ListGameSessionsEndpoint) Method() string             { return "GET" }
func (ListGameSessionsEndpoint) Path() string               { return "/api/games/{id}/sessions" }
func (ListGameSessionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGameSessionsEndpoint) Handle(ctx context.Context, req GameIDReq) (GameSessionsResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return nil, err
	}
	return wire.SessionViewsOf(core.MyGameSessions(ctx, game))
}

// GameSessionsResp is the caller's sessions of a game.
type GameSessionsResp = []wire.SessionView

// DeleteSessionEndpoint deletes one of the caller's sessions; someone
// else's does not exist for them (404).
type DeleteSessionEndpoint struct{}

func (DeleteSessionEndpoint) Method() string             { return "DELETE" }
func (DeleteSessionEndpoint) Path() string               { return "/api/sessions/{id}" }
func (DeleteSessionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteSessionEndpoint) Handle(ctx context.Context, req SessionIDReq) (SessionDeletedResp, error) {
	id, err := parseID(req.ID, "Invalid session ID")
	if err != nil {
		return SessionDeletedResp{}, err
	}
	if err := core.DeleteSession(ctx, id); err != nil {
		return SessionDeletedResp{}, err
	}
	Engine.Forget(id.String())
	return SessionDeletedResp{Status: "deleted"}, nil
}

// SessionIDReq names a session by its id.
type SessionIDReq struct {
	ID string `path:"id"`
}

// SessionDeletedResp confirms a session's deletion with status "deleted".
type SessionDeletedResp struct {
	Status string `json:"status"`
}
