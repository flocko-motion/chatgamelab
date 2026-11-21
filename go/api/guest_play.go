// package: api / schemaf endpoints
// type:    logic
// job:     answers what a share token offers: the game and how many plays are left
// limits:  token validation lives in game; share management (-> games_private_share.go)
package api

import (
	"context"

	"cgl/game"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// GuestGameInfoReq names a private share by its token.
type GuestGameInfoReq struct {
	Token string `path:"token"`
}

// GuestGameInfoResp is the public game info for the welcome screen.
type GuestGameInfoResp struct {
	// GameID and Public let the welcome screen offer a copy. Only a public game
	// may be copied: a share link alone does not release its AI instructions.
	GameID      uuid.UUID `json:"gameId"`
	Public      bool      `json:"public"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	// Remaining is null for unlimited and 0 when exhausted.
	Remaining *int `json:"remaining"`
}

// GetGuestGameInfoEndpoint returns the game a share token plays, for the
// welcome screen: its name, description and how many plays are left.
type GetGuestGameInfoEndpoint struct{}

func (GetGuestGameInfoEndpoint) Method() string             { return "GET" }
func (GetGuestGameInfoEndpoint) Path() string               { return "/api/play/{token}/info" }
func (GetGuestGameInfoEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetGuestGameInfoEndpoint) Handle(ctx context.Context, req GuestGameInfoReq) (GuestGameInfoResp, error) {
	if err := tokenLocked(); err != nil {
		return GuestGameInfoResp{}, err
	}
	if req.Token == "" {
		return GuestGameInfoResp{}, errBadRequest("Missing share token")
	}
	gameShare, gameObj, httpErr := game.ValidatePrivateShareToken(ctx, req.Token)
	if httpErr != nil {
		return GuestGameInfoResp{}, httpError(httpErr)
	}
	return GuestGameInfoResp{
		GameID:      gameObj.ID,
		Public:      gameObj.Public,
		Name:        gameObj.Name,
		Description: gameObj.Description,
		Remaining:   gameShare.Remaining,
	}, nil
}
