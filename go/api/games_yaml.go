// package: api / schemaf endpoints
// type:    logic
// job:     exports a game as a game file and updates a game from one
// limits:  the file's shape lives in obj (GameFile); rights and storage in the core
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GameFileResp is a game as a file.
type GameFileResp = obj.GameFile

// GetGameYAMLEndpoint exports a game as a game file, YAML by default. It
// takes seeing the game's content (hasContent; 403 otherwise).
type GetGameYAMLEndpoint struct{}

func (GetGameYAMLEndpoint) Method() string             { return "GET" }
func (GetGameYAMLEndpoint) Path() string               { return "/api/games/{id}/yaml" }
func (GetGameYAMLEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameYAMLEndpoint) Produces() []string {
	return []string{"application/x-yaml", "application/json"}
}
func (GetGameYAMLEndpoint) Handle(ctx context.Context, req GameIDReq) (GameFileResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.GameFile{}, err
	}
	return wire.GameFileOf(core.ReadGameCard(ctx, gameID))
}

// UpdateGameYAMLReq is the game from the path and the game file to update it
// from.
type UpdateGameYAMLReq struct {
	GameID string `path:"id"`
	obj.GameFile
}

// UpdateGameYAMLEndpoint updates a game from a game file, YAML or JSON, and
// answers with the game as stored.
//
// The file replaces the game's card and content; its id is ignored. It takes
// the same rights as an update (mayEditCard, mayEditContent).
type UpdateGameYAMLEndpoint struct{}

func (UpdateGameYAMLEndpoint) Method() string             { return "PUT" }
func (UpdateGameYAMLEndpoint) Path() string               { return "/api/games/{id}/yaml" }
func (UpdateGameYAMLEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameYAMLEndpoint) Consumes() []string {
	return []string{"application/x-yaml", "application/json"}
}
func (UpdateGameYAMLEndpoint) Handle(ctx context.Context, req UpdateGameYAMLReq) (GameViewResp, error) {
	gameID, err := parseID(req.GameID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(core.ImportGameFile(ctx, gameID, req.GameFile))
}
