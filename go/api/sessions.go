// package: api / schemaf endpoints
// type:    logic
// job:     lists and deletes a player's sessions
// limits:  playing a session is the engine's (-> engine.go); storage is db/game_sessions.go
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// ListUserSessionsEndpoint lists the caller's recent sessions with their games' names.
//
// Query: search (game name), sortBy (game, model for the AI platform,
// lastPlayed by default).
type ListUserSessionsEndpoint struct{}

func (ListUserSessionsEndpoint) Method() string             { return "GET" }
func (ListUserSessionsEndpoint) Path() string               { return "/api/sessions" }
func (ListUserSessionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListUserSessionsEndpoint) Handle(ctx context.Context, req ListUserSessionsReq) (UserSessionsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := db.GetGameSessionsByUserID(ctx, user.ID, db.GetUserSessionsFilters{
		Search:    req.Search,
		SortField: req.SortBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	return sessions, nil
}

// ListUserSessionsReq filters and sorts the session list.
type ListUserSessionsReq struct {
	Search string `query:"search" json:"-"`
	SortBy string `query:"sortBy" json:"-"`
}

// UserSessionsResp is a list of the caller's sessions with their games' names.
type UserSessionsResp = []db.UserSessionWithGame

// ListGameSessionsEndpoint lists the caller's own sessions of a game.
type ListGameSessionsEndpoint struct{}

func (ListGameSessionsEndpoint) Method() string             { return "GET" }
func (ListGameSessionsEndpoint) Path() string               { return "/api/games/{id}/sessions" }
func (ListGameSessionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGameSessionsEndpoint) Handle(ctx context.Context, req GameIDReq) (GameSessionsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return nil, err
	}
	sessions, err := db.GetGameSessionsByGameID(ctx, user.ID, gameID)
	if err != nil {
		return nil, appError(err)
	}
	return sessions, nil
}

// GameSessionsResp is a list of sessions.
type GameSessionsResp = []obj.GameSession

// DeleteSessionEndpoint deletes a session the caller owns and everything it stored.
type DeleteSessionEndpoint struct{}

func (DeleteSessionEndpoint) Method() string             { return "DELETE" }
func (DeleteSessionEndpoint) Path() string               { return "/api/sessions/{id}" }
func (DeleteSessionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteSessionEndpoint) Handle(ctx context.Context, req SessionIDReq) (SessionDeletedResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return SessionDeletedResp{}, err
	}
	sessionID, err := parseID(req.ID, "Invalid session ID")
	if err != nil {
		return SessionDeletedResp{}, err
	}
	if err := db.DeleteGameSession(ctx, user.ID, sessionID); err != nil {
		return SessionDeletedResp{}, appError(err)
	}
	Engine.Forget(sessionID.String())
	return SessionDeletedResp{Status: "deleted"}, nil
}

// SessionIDReq names a session by its id.
type SessionIDReq struct {
	ID string `path:"id" json:"-"`
}

// SessionDeletedResp confirms a session's deletion with status "deleted".
type SessionDeletedResp struct {
	Status string `json:"status"`
}
