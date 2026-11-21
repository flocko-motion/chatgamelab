// package: api / schemaf endpoints
// type:    logic
// job:     marks and unmarks the caller's favorite games
// limits:  decides nothing: the core does (core.MarkFavorite); a game view says whether it is a favorite, the list filters by it
package api

import (
	"context"

	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// FavouriteResp says whether the game is now a favorite.
type FavouriteResp struct {
	Favorite bool `json:"favourite"`
}

// AddFavoriteGameEndpoint marks a game as one of the caller's favorites.
// It takes reading the game (404 otherwise); marking twice changes nothing.
type AddFavoriteGameEndpoint struct{}

func (AddFavoriteGameEndpoint) Method() string             { return "POST" }
func (AddFavoriteGameEndpoint) Path() string               { return "/api/games/{id}/favourite" }
func (AddFavoriteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (AddFavoriteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (FavouriteResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return FavouriteResp{}, err
	}
	if err := core.MarkFavorite(ctx, gameID); err != nil {
		return FavouriteResp{}, err
	}
	return FavouriteResp{Favorite: true}, nil
}

// RemoveFavoriteGameEndpoint removes a game from the caller's favorites. It
// answers the same whatever the game: it gives nothing away.
type RemoveFavoriteGameEndpoint struct{}

func (RemoveFavoriteGameEndpoint) Method() string             { return "DELETE" }
func (RemoveFavoriteGameEndpoint) Path() string               { return "/api/games/{id}/favourite" }
func (RemoveFavoriteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveFavoriteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (FavouriteResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return FavouriteResp{}, err
	}
	if err := core.UnmarkFavorite(ctx, gameID); err != nil {
		return FavouriteResp{}, err
	}
	return FavouriteResp{Favorite: false}, nil
}
