// package: api / schemaf endpoints
// type:    logic
// job:     makes, lists, changes and stops a game's share links
// limits:  decides nothing: the core does (core/share_link.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// GameShareResp is a share link as the caller sees it.
type GameShareResp = wire.ShareLinkView

// GameShareIDReq names a game and one of its share links.
type GameShareIDReq struct {
	ID      string `path:"id"`
	ShareID string `path:"shareId"`
}

// CreateGameShareEndpoint makes a share link for a game the caller may
// share: paid by a key share they may draw from (sponsorKeyShareId), or by
// the workshop's key (workshopId) for the workshop's head and staff, and for
// those in it while it allows sharing. A workshop has one link per game; an
// existing one is answered. maxSessions limits the plays; null: unlimited.
type CreateGameShareEndpoint struct{}

func (CreateGameShareEndpoint) Method() string             { return "POST" }
func (CreateGameShareEndpoint) Path() string               { return "/api/games/{id}/shares" }
func (CreateGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateGameShareEndpoint) Handle(ctx context.Context, req CreateGameShareReq) (GameShareResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameShareResp{}, err
	}
	return wire.ShareLinkViewOf(core.CreateShareLink(ctx, core.ShareLinkDraft{
		Game: game, From: req.SponsorKeyShareID, Workshop: req.WorkshopID,
		MaxPlays: int32Ptr(req.MaxSessions), Tier: req.AiQualityTier,
	}))
}

// CreateGameShareReq is the game, where the money comes from, and the
// link's limit and tier.
type CreateGameShareReq struct {
	ID string `path:"id"`
	// SponsorKeyShareID is the key share it is paid by; workshop links
	// ignore it.
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"`
	// MaxSessions is null for unlimited.
	MaxSessions *int `json:"maxSessions"`
	// WorkshopID makes a workshop's link, paid by the workshop's key.
	WorkshopID *uuid.UUID `json:"workshopId"`
	// AiQualityTier is null for the source's default.
	AiQualityTier *string `json:"aiQualityTier"`
}

// DeleteGameShareEndpoint stops a share link: its guests go with it. For its
// maker, the game's owner, the key's owner, and head and staff when their
// organisation's key pays.
type DeleteGameShareEndpoint struct{}

func (DeleteGameShareEndpoint) Method() string             { return "DELETE" }
func (DeleteGameShareEndpoint) Path() string               { return "/api/games/{id}/shares/{shareId}" }
func (DeleteGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteGameShareEndpoint) Handle(ctx context.Context, req GameShareIDReq) (MessageResp, error) {
	id, err := parseID(req.ShareID, "Invalid share ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.DeleteShareLink(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Share revoked"}, nil
}

// UpdateGameShareEndpoint sets a share link's limit and tier, for those who
// may stop it. Both are set: a null maxSessions makes it unlimited, a null
// aiQualityTier plays at the source's default.
type UpdateGameShareEndpoint struct{}

func (UpdateGameShareEndpoint) Method() string             { return "PATCH" }
func (UpdateGameShareEndpoint) Path() string               { return "/api/games/{id}/shares/{shareId}" }
func (UpdateGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameShareEndpoint) Handle(ctx context.Context, req UpdateGameShareReq) (GameShareResp, error) {
	id, err := parseID(req.ShareID, "Invalid share ID")
	if err != nil {
		return GameShareResp{}, err
	}
	return wire.ShareLinkViewOf(core.UpdateShareLink(ctx, id, core.ShareLinkChanges{
		MaxPlays: int32Ptr(req.MaxSessions), Unlimited: req.MaxSessions == nil,
		Tier: req.AiQualityTier, DefaultTier: req.AiQualityTier == nil,
	}))
}

// UpdateGameShareReq is the link and its new limit and tier.
type UpdateGameShareReq struct {
	ID      string `path:"id"`
	ShareID string `path:"shareId"`
	// MaxSessions is null for unlimited.
	MaxSessions *int `json:"maxSessions"`
	// AiQualityTier is null for the source's default.
	AiQualityTier *string `json:"aiQualityTier"`
}

// PrivateShareStatusResp is a game's share links the caller sees.
type PrivateShareStatusResp struct {
	Shares []wire.ShareLinkView `json:"shares"`
}

// GetPrivateShareStatusEndpoint lists a game's share links the caller may
// see: those they made, of their game, paid by their keys or their
// organisation's, and their workshop's. A game they may not read is 404.
type GetPrivateShareStatusEndpoint struct{}

func (GetPrivateShareStatusEndpoint) Method() string             { return "GET" }
func (GetPrivateShareStatusEndpoint) Path() string               { return "/api/games/{id}/private-share" }
func (GetPrivateShareStatusEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetPrivateShareStatusEndpoint) Handle(ctx context.Context, req GameIDReq) (PrivateShareStatusResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return PrivateShareStatusResp{}, err
	}
	if _, err := core.ReadGameCard(ctx, game); err != nil {
		return PrivateShareStatusResp{}, err
	}
	// the game's embed link has a panel of its own (games_embed.go)
	notEmbed := false
	links, err := wire.ShareLinkViewsOf(core.ListShareLinks(ctx, core.ShareLinkFilter{Game: &game, Embed: &notEmbed}))
	if err != nil {
		return PrivateShareStatusResp{}, err
	}
	return PrivateShareStatusResp{Shares: links}, nil
}

// int32Ptr narrows an optional count.
func int32Ptr(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}
