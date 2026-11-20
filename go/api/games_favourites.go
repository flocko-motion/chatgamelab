// package: api / schemaf endpoints
// type:    logic
// job:     lists, adds and removes the caller's favourite games
// limits:  persistence lives in db
package api

import (
	"context"
	"fmt"

	"cgl/db"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// FavouriteResp says whether the game is now a favourite.
type FavouriteResp struct {
	Favourite bool `json:"favourite"`
}

// FavouriteGamesReq takes no parameters.
type FavouriteGamesReq struct{}

// ListFavouriteGamesEndpoint lists the games the caller has marked as favourites.
type ListFavouriteGamesEndpoint struct{}

func (ListFavouriteGamesEndpoint) Method() string             { return "GET" }
func (ListFavouriteGamesEndpoint) Path() string               { return "/api/games/favourites" }
func (ListFavouriteGamesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListFavouriteGamesEndpoint) Handle(ctx context.Context, _ FavouriteGamesReq) (GamesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	games, err := db.GetFavouriteGames(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favourite games: %w", err)
	}
	return games, nil
}

// AddFavouriteGameEndpoint adds a game to the caller's favourites.
type AddFavouriteGameEndpoint struct{}

func (AddFavouriteGameEndpoint) Method() string             { return "POST" }
func (AddFavouriteGameEndpoint) Path() string               { return "/api/games/{id}/favourite" }
func (AddFavouriteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (AddFavouriteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (FavouriteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return FavouriteResp{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return FavouriteResp{}, err
	}
	if err := db.AddFavouriteGame(ctx, user.ID, gameID); err != nil {
		return FavouriteResp{}, fmt.Errorf("failed to add favourite: %w", err)
	}
	return FavouriteResp{Favourite: true}, nil
}

// RemoveFavouriteGameEndpoint removes a game from the caller's favourites.
type RemoveFavouriteGameEndpoint struct{}

func (RemoveFavouriteGameEndpoint) Method() string             { return "DELETE" }
func (RemoveFavouriteGameEndpoint) Path() string               { return "/api/games/{id}/favourite" }
func (RemoveFavouriteGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveFavouriteGameEndpoint) Handle(ctx context.Context, req GameIDReq) (FavouriteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return FavouriteResp{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return FavouriteResp{}, err
	}
	if err := db.RemoveFavouriteGame(ctx, user.ID, gameID); err != nil {
		return FavouriteResp{}, fmt.Errorf("failed to remove favourite: %w", err)
	}
	return FavouriteResp{Favourite: false}, nil
}
