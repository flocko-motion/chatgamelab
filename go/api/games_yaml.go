// package: api / schemaf endpoints
// type:    logic
// job:     exports a game as a game file and updates a game from one
// limits:  the file's shape lives in obj (GameFile); storage in db
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GameFileResp is a game as a file.
type GameFileResp = obj.GameFile

// GetGameYAMLEndpoint exports a game as a game file, YAML by default.
type GetGameYAMLEndpoint struct{}

func (GetGameYAMLEndpoint) Method() string             { return "GET" }
func (GetGameYAMLEndpoint) Path() string               { return "/api/games/{id}/yaml" }
func (GetGameYAMLEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameYAMLEndpoint) Produces() []string {
	return []string{"application/x-yaml", "application/json"}
}
func (GetGameYAMLEndpoint) Handle(ctx context.Context, req GameIDReq) (GameFileResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.GameFile{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.GameFile{}, err
	}
	log.Debug("exporting game as YAML", "game_id", gameID, "user_id", user.ID)
	game, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.GameFile{}, errNotFound("Game not found")
	}
	return obj.NewGameFile(*game), nil
}

// UpdateGameYAMLReq is the game from the path and the game file to update it
// from.
type UpdateGameYAMLReq struct {
	GameID string `path:"id" json:"-"`
	obj.GameFile
}

// UpdateGameYAMLEndpoint updates a game from a game file, YAML or JSON, and
// answers with the game.
//
// The file's content replaces the game's; its id is ignored.
type UpdateGameYAMLEndpoint struct{}

func (UpdateGameYAMLEndpoint) Method() string             { return "PUT" }
func (UpdateGameYAMLEndpoint) Path() string               { return "/api/games/{id}/yaml" }
func (UpdateGameYAMLEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameYAMLEndpoint) Consumes() []string {
	return []string{"application/x-yaml", "application/json"}
}
func (UpdateGameYAMLEndpoint) Handle(ctx context.Context, req UpdateGameYAMLReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.GameID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	log.Debug("updating game from a game file", "game_id", gameID, "user_id", user.ID)
	if err := db.UpdateGameFromFile(ctx, user.ID, gameID, req.GameFile); err != nil {
		return obj.Game{}, fmt.Errorf("failed to update game: %w", appError(err))
	}
	updated, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	return *updated, nil
}
