// package: api / schemaf endpoints
// type:    logic
// job:     lists, creates, clones, updates and deletes games
// limits:  game storage and its permission checks live in db
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GameResp is a game.
type GameResp = obj.Game

// GamesResp is a list of games.
type GamesResp = []obj.Game

// GameViewResp is a game as the caller sees it.
type GameViewResp = wire.GameView

// GameViewPageResp is a page of games and the total.
type GameViewPageResp = wire.GameViewPage

// GameIDReq names a game by its id.
type GameIDReq struct {
	ID string `path:"id"`
}

// ListGamesEndpoint lists a page of the games the caller can see, and the total.
//
// Query, all optional: filter (own, others, public, favorites, sponsored;
// repeat to combine), search, sortBy (createdAt, modifiedAt, name, playCount,
// visibility, creator), sortDir (asc, desc), limit, offset.
type ListGamesEndpoint struct{}

func (ListGamesEndpoint) Method() string             { return "GET" }
func (ListGamesEndpoint) Path() string               { return "/api/games" }
func (ListGamesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGamesEndpoint) Handle(ctx context.Context, req ListGamesReq) (GameViewPageResp, error) {
	return wire.GameViewPageOf(core.ListGameCards(ctx, req.query()))
}

// ListGamesReq filters, sorts and pages the game list.
type ListGamesReq struct {
	Filter  []string `query:"filter"`
	Search  *string  `query:"search"`
	SortBy  *string  `query:"sortBy"`
	SortDir *string  `query:"sortDir"`
	Limit   *int     `query:"limit"`
	Offset  *int     `query:"offset"`
}

func (r ListGamesReq) query() core.GameCardQuery {
	q := core.GameCardQuery{Filter: r.Filter, Search: opt(r.Search), SortBy: opt(r.SortBy), SortDir: opt(r.SortDir)}
	if r.Limit != nil {
		q.Limit = *r.Limit
	}
	if r.Offset != nil {
		q.Offset = *r.Offset
	}
	return q
}

// NewGameEndpoint returns the template for a game the caller creates: what a
// new game starts with and what the caller may do with it. Nothing is stored;
// the client fills it in and creates the game with CreateGameEndpoint.
type NewGameEndpoint struct{}

func (NewGameEndpoint) Method() string             { return "GET" }
func (NewGameEndpoint) Path() string               { return "/api/games/new" }
func (NewGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (NewGameEndpoint) Handle(ctx context.Context, _ NewGameReq) (GameTemplateResp, error) {
	return wire.GameTemplateOf(core.NewGame(ctx))
}

// NewGameReq asks for nothing: the template depends only on the caller.
type NewGameReq struct{}

// GameTemplateResp is a game not yet created.
type GameTemplateResp = wire.GameTemplate

// CreateGameEndpoint creates a game owned by the caller from the filled-in
// template, and answers with it as stored. Only the name is required, at
// most 70 characters; making it public takes the template's mayPublish. In
// a workshop, the game belongs to it.
type CreateGameEndpoint struct{}

func (CreateGameEndpoint) Method() string             { return "POST" }
func (CreateGameEndpoint) Path() string               { return "/api/games/new" }
func (CreateGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateGameEndpoint) Handle(ctx context.Context, req CreateGameReq) (GameViewResp, error) {
	return wire.GameViewOf(core.CreateGame(ctx, req.changes()))
}

// CreateGameReq is a new game's fields; left out, a field keeps the
// template's value.
type CreateGameReq struct {
	GameFields
}

// GetGameEndpoint returns a game the caller can see.
type GetGameEndpoint struct{}

func (GetGameEndpoint) Method() string             { return "GET" }
func (GetGameEndpoint) Path() string               { return "/api/games/{id}" }
func (GetGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameViewResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(core.ReadGameCard(ctx, gameID))
}

// UpdateGameEndpoint changes a game's card and content and answers with the
// game as stored.
//
// Send what changes; a field left out stays as it is, and sending the whole
// game back is fine: only what differs counts. Changing a section the caller
// may not edit (mayEditCard, mayEditContent), or making the game public
// without mayPublish, refuses the whole request (403). The name is at most 70
// characters. Workshop, sponsor and creator are not changed here.
type UpdateGameEndpoint struct{}

func (UpdateGameEndpoint) Method() string             { return "POST" }
func (UpdateGameEndpoint) Path() string               { return "/api/games/{id}" }
func (UpdateGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameEndpoint) Handle(ctx context.Context, req UpdateGameReq) (GameViewResp, error) {
	gameID, err := parseID(req.GameID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(core.UpdateGame(ctx, gameID, req.changes()))
}

// UpdateGameReq is the game from the path and what changes from the body.
type UpdateGameReq struct {
	GameID string `path:"id"`
	GameFields
}

// GameFields are a game's fields as a client sends them, under the names a
// game view has. Left out (or null): unchanged. Icon and firstImage: ""
// removes. Theme, firstMessage, firstStatus: "" removes. Genre: "" leaves it.
type GameFields struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *[]byte `json:"icon"`
	Genre       *string `json:"genre"`
	Theme       *string `json:"theme"`
	Public      *bool   `json:"public"`

	SystemMessageScenario  *string `json:"systemMessageScenario"`
	SystemMessageGameStart *string `json:"systemMessageGameStart"`
	ImageStyle             *string `json:"imageStyle"`
	CSS                    *string `json:"css"`
	StatusFields           *string `json:"statusFields"`
	FirstMessage           *string `json:"firstMessage"`
	FirstStatus            *string `json:"firstStatus"`
	FirstImage             *[]byte `json:"firstImage"`
}

// deref is a given byte field, "" (removed) included, or nil when left out.
func deref(b *[]byte) []byte {
	if b == nil {
		return nil
	}
	if *b == nil {
		return []byte{}
	}
	return *b
}

func (r GameFields) changes() core.GameChanges {
	return core.GameChanges{
		Name:         r.Name,
		Description:  r.Description,
		Icon:         deref(r.Icon),
		Genre:        r.Genre,
		Theme:        r.Theme,
		Public:       r.Public,
		Scenario:     r.SystemMessageScenario,
		GameStart:    r.SystemMessageGameStart,
		ImageStyle:   r.ImageStyle,
		CSS:          r.CSS,
		StatusFields: r.StatusFields,
		FirstMessage: r.FirstMessage,
		FirstStatus:  r.FirstStatus,
		FirstImage:   deref(r.FirstImage),
	}
}

// DeleteGameEndpoint deletes a game and answers with it as it was. Deleting
// takes mayEditSettings (403 otherwise).
type DeleteGameEndpoint struct{}

func (DeleteGameEndpoint) Method() string             { return "DELETE" }
func (DeleteGameEndpoint) Path() string               { return "/api/games/{id}" }
func (DeleteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameViewResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(core.DeleteGame(ctx, gameID))
}

// CloneGameEndpoint copies a game into a new private game of the caller's
// and answers with the copy. It takes mayCopy (403 otherwise); the copy has
// the game's card and content, not its images.
type CloneGameEndpoint struct{}

func (CloneGameEndpoint) Method() string             { return "POST" }
func (CloneGameEndpoint) Path() string               { return "/api/games/{id}/clone" }
func (CloneGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CloneGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameViewResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(core.CopyGame(ctx, gameID))
}
