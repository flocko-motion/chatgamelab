// package: api / schemaf endpoints
// type:    logic
// job:     sets and removes the API key share that sponsors a game, and answers which keys can play it
// limits:  sponsorship and key resolution live in db and game
package api

import (
	"context"

	"cgl/db"
	"cgl/game"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SetGameSponsorEndpoint sponsors a game publicly with one of the caller's
// API key shares and answers with the game.
type SetGameSponsorEndpoint struct{}

func (SetGameSponsorEndpoint) Method() string             { return "PUT" }
func (SetGameSponsorEndpoint) Path() string               { return "/api/games/{id}/sponsor" }
func (SetGameSponsorEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetGameSponsorEndpoint) Handle(ctx context.Context, req GameSponsorReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	if req.ShareID == uuid.Nil {
		return obj.Game{}, errBadRequest("Missing required field: shareId")
	}
	if err := db.SetGamePublicSponsorship(ctx, user.ID, gameID, req.ShareID); err != nil {
		return obj.Game{}, appError(err)
	}
	g, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	return *g, nil
}

// GameSponsorReq is the game from the path and the API key share that sponsors it.
type GameSponsorReq struct {
	ID      string    `path:"id" json:"-"`
	ShareID uuid.UUID `json:"shareId"`
}

// RemoveGameSponsorEndpoint removes the public sponsorship from a game and
// answers with the game.
type RemoveGameSponsorEndpoint struct{}

func (RemoveGameSponsorEndpoint) Method() string             { return "DELETE" }
func (RemoveGameSponsorEndpoint) Path() string               { return "/api/games/{id}/sponsor" }
func (RemoveGameSponsorEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveGameSponsorEndpoint) Handle(ctx context.Context, req GameIDReq) (GameResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Game{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.Game{}, err
	}
	if err := db.ClearGamePublicSponsorship(ctx, user.ID, gameID); err != nil {
		return obj.Game{}, appError(err)
	}
	g, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return obj.Game{}, errNotFound("Game not found")
	}
	return *g, nil
}

// GameAvailableKeysResp is the API keys the caller can play a game with, best first.
type GameAvailableKeysResp = []obj.AvailableKey

// ListGameAvailableKeysEndpoint lists the API keys the caller can play a game
// with, in order of priority.
type ListGameAvailableKeysEndpoint struct{}

func (ListGameAvailableKeysEndpoint) Method() string             { return "GET" }
func (ListGameAvailableKeysEndpoint) Path() string               { return "/api/games/{id}/available-keys" }
func (ListGameAvailableKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGameAvailableKeysEndpoint) Handle(ctx context.Context, req GameIDReq) (GameAvailableKeysResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return nil, err
	}
	keys, err := db.GetAvailableKeysForGame(ctx, user.ID, gameID)
	if err != nil {
		return nil, appError(err)
	}
	return keys, nil
}

// GameApiKeyStatusResp says whether an API key resolves for the caller and the game.
type GameApiKeyStatusResp struct {
	Available bool `json:"available"`
}

// GetGameApiKeyStatusEndpoint answers whether an API key resolves for the
// caller playing a game.
type GetGameApiKeyStatusEndpoint struct{}

func (GetGameApiKeyStatusEndpoint) Method() string             { return "GET" }
func (GetGameApiKeyStatusEndpoint) Path() string               { return "/api/games/{id}/api-key-status" }
func (GetGameApiKeyStatusEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameApiKeyStatusEndpoint) Handle(ctx context.Context, req GameIDReq) (GameApiKeyStatusResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return GameApiKeyStatusResp{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameApiKeyStatusResp{}, err
	}
	return GameApiKeyStatusResp{Available: game.IsApiKeyAvailable(ctx, user.ID, gameID)}, nil
}
