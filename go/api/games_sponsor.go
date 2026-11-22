// package: api / schemaf endpoints
// type:    logic
// job:     sponsors games, and says which keys pay for the caller's plays of a game
// limits:  decides nothing: the core does (core/share_link.go, core/payer.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SetGameSponsorEndpoint makes a key share pay for a game's public plays,
// replacing what paid before: for those who edit the game, with a share
// they may draw from. A key known to fail is refused.
type SetGameSponsorEndpoint struct{}

func (SetGameSponsorEndpoint) Method() string             { return "PUT" }
func (SetGameSponsorEndpoint) Path() string               { return "/api/games/{id}/sponsor" }
func (SetGameSponsorEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetGameSponsorEndpoint) Handle(ctx context.Context, req GameSponsorReq) (GameViewResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	if req.ShareID == uuid.Nil {
		return GameViewResp{}, errBadRequest("Missing required field: shareId")
	}
	return wire.GameViewOf(ctx)(core.SponsorGame(ctx, game, req.ShareID))
}

// GameSponsorReq is the game and the key share to sponsor it with.
type GameSponsorReq struct {
	ID      string    `path:"id"`
	ShareID uuid.UUID `json:"shareId"`
}

// RemoveGameSponsorEndpoint stops a game's sponsorship: for those who edit
// the game, the key's owner, and head and staff when their organisation's
// key pays.
type RemoveGameSponsorEndpoint struct{}

func (RemoveGameSponsorEndpoint) Method() string             { return "DELETE" }
func (RemoveGameSponsorEndpoint) Path() string               { return "/api/games/{id}/sponsor" }
func (RemoveGameSponsorEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveGameSponsorEndpoint) Handle(ctx context.Context, req GameIDReq) (GameViewResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	if err := core.EndSponsorship(ctx, game); err != nil {
		return GameViewResp{}, err
	}
	return wire.GameViewOf(ctx)(core.ReadGameCard(ctx, game))
}

// GameAvailableKeysResp is the keys that would pay for the caller's plays.
type GameAvailableKeysResp = []obj.AvailableKey

// ListGameAvailableKeysEndpoint lists the keys that would pay for the
// caller's plays of a game they may read, in the order they are tried.
type ListGameAvailableKeysEndpoint struct{}

func (ListGameAvailableKeysEndpoint) Method() string             { return "GET" }
func (ListGameAvailableKeysEndpoint) Path() string               { return "/api/games/{id}/available-keys" }
func (ListGameAvailableKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListGameAvailableKeysEndpoint) Handle(ctx context.Context, req GameIDReq) (GameAvailableKeysResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return nil, err
	}
	if _, err := core.ReadGameCard(ctx, game); err != nil {
		return nil, err
	}
	payers, _, err := core.Payers(ctx, game, "")
	if err != nil {
		return nil, err
	}
	keys := make(GameAvailableKeysResp, 0, len(payers))
	for i, p := range payers {
		k := obj.AvailableKey{Name: p.KeyName, Platform: p.Platform, Source: p.Source, IsDefault: i == 0}
		if p.ShareID != nil {
			k.ShareID = *p.ShareID
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// GameApiKeyStatusResp says whether a key would pay for the caller's plays.
type GameApiKeyStatusResp struct {
	Available bool `json:"available"`
}

// GetGameApiKeyStatusEndpoint answers whether a key would pay for the
// caller's plays of a game they may read.
type GetGameApiKeyStatusEndpoint struct{}

func (GetGameApiKeyStatusEndpoint) Method() string             { return "GET" }
func (GetGameApiKeyStatusEndpoint) Path() string               { return "/api/games/{id}/api-key-status" }
func (GetGameApiKeyStatusEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameApiKeyStatusEndpoint) Handle(ctx context.Context, req GameIDReq) (GameApiKeyStatusResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameApiKeyStatusResp{}, err
	}
	if _, err := core.ReadGameCard(ctx, game); err != nil {
		return GameApiKeyStatusResp{}, err
	}
	payers, _, err := core.Payers(ctx, game, "")
	if err != nil {
		return GameApiKeyStatusResp{}, err
	}
	return GameApiKeyStatusResp{Available: len(payers) > 0}, nil
}
