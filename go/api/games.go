// package: api / schemaf endpoints
// type:    logic
// job:     lists, creates, clones, updates and deletes games
// limits:  game storage and its permission checks live in db
package api

import (
	"context"
	"fmt"
	"strings"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// GameResp is a game.
type GameResp = obj.Game

// GamesResp is a list of games.
type GamesResp = []obj.Game

// GameIDReq names a game by its id.
type GameIDReq struct {
	ID string `path:"id" json:"-"`
}

// ListGamesEndpoint lists the games the caller can see.
//
// Query: search (name, case-insensitive), sortBy (name, createdAt, modifiedAt,
// playCount), sortDir (asc, desc), filter (all, own, public, organization,
// favorites).
type ListGamesEndpoint struct{}

func (ListGamesEndpoint) Method() string             { return "GET" }
func (ListGamesEndpoint) Path() string               { return "/api/games" }
func (ListGamesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGamesEndpoint) Handle(ctx context.Context, req ListGamesReq) (GamesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	games, err := db.GetGames(ctx, &user.ID, &db.GetGamesFilters{
		Search:    req.Search,
		SortField: req.SortBy,
		SortDir:   req.SortDir,
		Filter:    req.Filter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}
	return games, nil
}

// ListGamesReq filters and sorts the game list.
type ListGamesReq struct {
	Search  string `query:"search" json:"-"`
	SortBy  string `query:"sortBy" json:"-"`
	SortDir string `query:"sortDir" json:"-"`
	Filter  string `query:"filter" json:"-"`
}

// CreateGameEndpoint creates a game owned by the caller.
//
// Only the name is required, at most 70 characters; the genre is one of the
// engine's genres and makes an adventure when empty.
type CreateGameEndpoint struct{}

func (CreateGameEndpoint) Method() string             { return "POST" }
func (CreateGameEndpoint) Path() string               { return "/api/games/new" }
func (CreateGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateGameEndpoint) Handle(ctx context.Context, req CreateGameReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return obj.Game{}, errBadRequest("Missing required field: name")
	}
	if len(name) > 70 {
		return obj.Game{}, appError(obj.ErrNameTooLong("Game name must be 70 characters or less"))
	}
	game := obj.Game{
		Name:        req.Name,
		Description: req.Description,
		Genre:       req.Genre,
	}
	if req.Public != nil {
		game.Public = *req.Public
	}
	if err := db.CreateGame(ctx, user.ID, &game); err != nil {
		return obj.Game{}, appError(err)
	}
	created, err := db.GetGameByID(ctx, &user.ID, game.ID)
	if err != nil {
		return obj.Game{}, fmt.Errorf("failed to load created game: %w", err)
	}
	return *created, nil
}

// CreateGameReq is a new game's name, description, visibility and genre.
type CreateGameReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Public      *bool  `json:"public,omitempty"`
	// Genre is one of the engine's genres; empty makes an adventure.
	Genre string `json:"genre,omitempty"`
}

// GetGameEndpoint returns a game the caller can see.
type GetGameEndpoint struct{}

func (GetGameEndpoint) Method() string             { return "GET" }
func (GetGameEndpoint) Path() string               { return "/api/games/{id}" }
func (GetGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	game, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	return *game, nil
}

// UpdateGameEndpoint updates a game's properties.
//
// Takes the whole game; its name is at most 70 characters. Answers with the
// game as stored.
type UpdateGameEndpoint struct{}

func (UpdateGameEndpoint) Method() string             { return "POST" }
func (UpdateGameEndpoint) Path() string               { return "/api/games/{id}" }
func (UpdateGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameEndpoint) Handle(ctx context.Context, req UpdateGameReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.GameID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	updated := req.Game
	updated.ID = gameID
	if len(strings.TrimSpace(updated.Name)) > 70 {
		return obj.Game{}, appError(obj.ErrNameTooLong("Game name must be 70 characters or less"))
	}
	if err := db.UpdateGame(ctx, user.ID, &updated); err != nil {
		return obj.Game{}, appError(err)
	}
	game, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	return *game, nil
}

// UpdateGameReq is the game from the path and its new properties from the body.
type UpdateGameReq struct {
	GameID string `path:"id" json:"-"`
	obj.Game
}

// DeleteGameEndpoint deletes a game and answers with it as it was.
type DeleteGameEndpoint struct{}

func (DeleteGameEndpoint) Method() string             { return "DELETE" }
func (DeleteGameEndpoint) Path() string               { return "/api/games/{id}" }
func (DeleteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	deleted, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	if err := db.DeleteGame(ctx, user.ID, gameID); err != nil {
		return obj.Game{}, fmt.Errorf("failed to delete game: %w", err)
	}
	return *deleted, nil
}

// CloneGameEndpoint copies a public game, or one of the caller's own, into a
// new private game of the caller's.
type CloneGameEndpoint struct{}

func (CloneGameEndpoint) Method() string             { return "POST" }
func (CloneGameEndpoint) Path() string               { return "/api/games/{id}/clone" }
func (CloneGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CloneGameEndpoint) Handle(ctx context.Context, req GameIDReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	source, err := db.GetGameByID(ctx, nil, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	isOwner := source.Meta.CreatedBy.Valid && source.Meta.CreatedBy.UUID == user.ID
	if !source.Public && !isOwner {
		return obj.Game{}, errNotFound("Game not found")
	}

	// The original creator carries over from copy to copy.
	var originalCreator *uuid.UUID
	if source.OriginallyCreatedBy != nil {
		originalCreator = source.OriginallyCreatedBy
	} else if source.Meta.CreatedBy.Valid {
		originalCreator = &source.Meta.CreatedBy.UUID
	}

	clone := obj.Game{
		Name:                   source.Name + " (Copy)",
		Description:            source.Description,
		Public:                 false,
		ImageStyle:             source.ImageStyle,
		SystemMessageScenario:  source.SystemMessageScenario,
		SystemMessageGameStart: source.SystemMessageGameStart,
		StatusFields:           source.StatusFields,
		Genre:                  source.Genre,
		Theme:                  source.Theme,
		FirstMessage:           source.FirstMessage,
		FirstStatus:            source.FirstStatus,
		CSS:                    source.CSS,
		OriginallyCreatedBy:    originalCreator,
	}
	if err := db.CreateGame(ctx, user.ID, &clone); err != nil {
		return obj.Game{}, fmt.Errorf("failed to clone game: %w", err)
	}
	if err := db.IncrementGameCloneCount(ctx, gameID); err != nil {
		log.Warn("failed to increment clone count", "error", err)
	}
	created, err := db.GetGameByID(ctx, &user.ID, clone.ID)
	if err != nil {
		return obj.Game{}, fmt.Errorf("failed to load cloned game: %w", err)
	}
	return *created, nil
}
