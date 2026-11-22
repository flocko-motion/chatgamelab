// package: api / schemaf endpoints
// type:    logic
// job:     answers what a share token offers: the game and how many plays are left
// limits:  the share link is read by the core (core.ShareLinkPlay); share management (-> games_private_share.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

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
	// Playability is whether a new session can start through the link now:
	// "playable", or the refusal a start would answer (no_plays, paused,
	// genre, key_rejected, …), as a game view says it. Continuing a session
	// the link started takes no play.
	Playability string `json:"playability"`
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
	play, err := core.ShareLinkPlay(ctx, req.Token)
	if err != nil {
		return GuestGameInfoResp{}, err
	}
	refusal := core.SharePlayability(ctx, play)
	var remaining *int
	if play.Remaining != nil {
		r := int(*play.Remaining)
		remaining = &r
	}
	return GuestGameInfoResp{
		GameID:      play.Game.ID(),
		Public:      play.Game.Public(),
		Name:        play.Game.Name(),
		Description: play.Game.Description(),
		Remaining:   remaining,
		Playability: wire.Playability(refusal),
	}, nil
}
