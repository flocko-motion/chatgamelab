// package: api / schemaf endpoints
// type:    logic
// job:     a game's embed link: reads and sets it; an organisation's verified pages
// limits:  decides nothing: the core does (core/embed.go, core/verified_page.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// VerifiedPageResp is one verified page.
type VerifiedPageResp = wire.VerifiedPageView

// GameEmbedResp is a game's embed link; Embed is null while it has none.
type GameEmbedResp struct {
	Embed *wire.GameEmbedView `json:"embed"`
}

// GetGameEmbedEndpoint reads a game's embed link, for whoever may share
// the game.
type GetGameEmbedEndpoint struct{}

func (GetGameEmbedEndpoint) Method() string             { return "GET" }
func (GetGameEmbedEndpoint) Path() string               { return "/api/games/{id}/embed" }
func (GetGameEmbedEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameEmbedEndpoint) Handle(ctx context.Context, req GameIDReq) (GameEmbedResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameEmbedResp{}, err
	}
	embed, err := wire.GameEmbedViewOf(core.ReadGameEmbed(ctx, game))
	return GameEmbedResp{Embed: embed}, err
}

// SetGameEmbedReq switches a game's embedding on or off and sets its mode,
// what pays and its verified page.
type SetGameEmbedReq struct {
	ID      string `path:"id"`
	Enabled bool   `json:"enabled"`
	// Mode is "verified" (one verified page), the only mode.
	Mode string `json:"mode"`
	// SponsorKeyShareID is the share it is paid by; null keeps the one
	// paying now. The first time it is required.
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"`
	// VerifiedPageID is required for the verified mode: one of the game's
	// organisation's pages.
	VerifiedPageID *uuid.UUID `json:"verifiedPageId"`
}

// SetGameEmbedEndpoint sets a game's embed link, making it the first time,
// for whoever may share the game. Its token never changes: off keeps it,
// on brings back the same snippet.
type SetGameEmbedEndpoint struct{}

func (SetGameEmbedEndpoint) Method() string             { return "PUT" }
func (SetGameEmbedEndpoint) Path() string               { return "/api/games/{id}/embed" }
func (SetGameEmbedEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetGameEmbedEndpoint) Handle(ctx context.Context, req SetGameEmbedReq) (GameEmbedResp, error) {
	game, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameEmbedResp{}, err
	}
	embed, err := wire.GameEmbedViewOf(core.SetGameEmbed(ctx, game, core.EmbedSettings{
		Enabled: req.Enabled, Mode: core.EmbedMode(req.Mode), From: req.SponsorKeyShareID, VerifiedPage: req.VerifiedPageID,
	}))
	return GameEmbedResp{Embed: embed}, err
}

// OrganisationIDReq names an organisation.
type OrganisationIDReq struct {
	ID string `path:"id"`
}

// VerifiedPagesResp is an organisation's verified pages.
type VerifiedPagesResp struct {
	Pages []wire.VerifiedPageView `json:"pages"`
}

// ListVerifiedPagesEndpoint lists an organisation's verified pages, for its
// head and staff.
type ListVerifiedPagesEndpoint struct{}

func (ListVerifiedPagesEndpoint) Method() string             { return "GET" }
func (ListVerifiedPagesEndpoint) Path() string               { return "/api/institutions/{id}/verified-pages" }
func (ListVerifiedPagesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListVerifiedPagesEndpoint) Handle(ctx context.Context, req OrganisationIDReq) (VerifiedPagesResp, error) {
	org, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return VerifiedPagesResp{}, err
	}
	pages, err := wire.VerifiedPageViewsOf(core.ListVerifiedPages(ctx, org))
	return VerifiedPagesResp{Pages: pages}, err
}

// CreateVerifiedPageReq is a new verified page: its address and callback,
// both https.
type CreateVerifiedPageReq struct {
	ID          string `path:"id"`
	PageURL     string `json:"pageUrl"`
	CallbackURL string `json:"callbackUrl"`
}

// CreateVerifiedPageEndpoint registers a verified page for an organisation,
// with a new page ID, for its head and staff.
type CreateVerifiedPageEndpoint struct{}

func (CreateVerifiedPageEndpoint) Method() string             { return "POST" }
func (CreateVerifiedPageEndpoint) Path() string               { return "/api/institutions/{id}/verified-pages" }
func (CreateVerifiedPageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateVerifiedPageEndpoint) Handle(ctx context.Context, req CreateVerifiedPageReq) (VerifiedPageResp, error) {
	org, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return VerifiedPageResp{}, err
	}
	return wire.VerifiedPageViewOf(core.CreateVerifiedPage(ctx, org, core.VerifiedPageDraft{
		PageURL: req.PageURL, CallbackURL: req.CallbackURL,
	}))
}

// UpdateVerifiedPageReq is a verified page's new address and callback.
type UpdateVerifiedPageReq struct {
	ID          string `path:"id"`
	PageURL     string `json:"pageUrl"`
	CallbackURL string `json:"callbackUrl"`
}

// UpdateVerifiedPageEndpoint changes a verified page's address and
// callback; its page ID stays.
type UpdateVerifiedPageEndpoint struct{}

func (UpdateVerifiedPageEndpoint) Method() string             { return "PATCH" }
func (UpdateVerifiedPageEndpoint) Path() string               { return "/api/verified-pages/{id}" }
func (UpdateVerifiedPageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateVerifiedPageEndpoint) Handle(ctx context.Context, req UpdateVerifiedPageReq) (VerifiedPageResp, error) {
	id, err := parseID(req.ID, "Invalid verified page ID")
	if err != nil {
		return VerifiedPageResp{}, err
	}
	return wire.VerifiedPageViewOf(core.UpdateVerifiedPage(ctx, id, core.VerifiedPageDraft{
		PageURL: req.PageURL, CallbackURL: req.CallbackURL,
	}))
}

// VerifiedPageIDReq names a verified page.
type VerifiedPageIDReq struct {
	ID string `path:"id"`
}

// DeleteVerifiedPageEndpoint removes a verified page; the games embedded
// for it start nothing until they get another.
type DeleteVerifiedPageEndpoint struct{}

func (DeleteVerifiedPageEndpoint) Method() string             { return "DELETE" }
func (DeleteVerifiedPageEndpoint) Path() string               { return "/api/verified-pages/{id}" }
func (DeleteVerifiedPageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteVerifiedPageEndpoint) Handle(ctx context.Context, req VerifiedPageIDReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid verified page ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.DeleteVerifiedPage(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Verified page removed"}, nil
}

// VerifiedPageFilter narrows the verified page list by approval.
type VerifiedPageFilter string

const (
	VerifiedPageFilterWaiting  VerifiedPageFilter = "waiting"
	VerifiedPageFilterApproved VerifiedPageFilter = "approved"
)

// VerifiedPageSort orders the verified page list; the default is status,
// waiting first.
type VerifiedPageSort string

const (
	VerifiedPageSortStatus       VerifiedPageSort = "status"
	VerifiedPageSortOrganisation VerifiedPageSort = "organisation"
	VerifiedPageSortCreatedAt    VerifiedPageSort = "createdAt"
)

// ListAllVerifiedPagesReq filters, sorts and pages the verified page list.
type ListAllVerifiedPagesReq struct {
	ListParams[VerifiedPageFilter, VerifiedPageSort]
}

// VerifiedPagePageResp is a page of verified pages and the total.
type VerifiedPagePageResp = wire.Page[wire.VerifiedPageView]

// ListAllVerifiedPagesEndpoint lists a page of the verified pages the caller
// may see, waiting first: every one for an admin, the allow list.
type ListAllVerifiedPagesEndpoint struct{}

func (ListAllVerifiedPagesEndpoint) Method() string             { return "GET" }
func (ListAllVerifiedPagesEndpoint) Path() string               { return "/api/verified-pages" }
func (ListAllVerifiedPagesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListAllVerifiedPagesEndpoint) Handle(ctx context.Context, req ListAllVerifiedPagesReq) (VerifiedPagePageResp, error) {
	return wire.VerifiedPageViewPageOf(core.List(ctx, core.VerifiedPages(), req.query()))
}

// ApproveVerifiedPageReq approves a verified page, or withdraws the approval.
type ApproveVerifiedPageReq struct {
	ID       string `path:"id"`
	Approved bool   `json:"approved"`
}

// ApproveVerifiedPageEndpoint approves a verified page or withdraws the
// approval, for an admin.
type ApproveVerifiedPageEndpoint struct{}

func (ApproveVerifiedPageEndpoint) Method() string             { return "PUT" }
func (ApproveVerifiedPageEndpoint) Path() string               { return "/api/verified-pages/{id}/approval" }
func (ApproveVerifiedPageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ApproveVerifiedPageEndpoint) Handle(ctx context.Context, req ApproveVerifiedPageReq) (VerifiedPageResp, error) {
	id, err := parseID(req.ID, "Invalid verified page ID")
	if err != nil {
		return VerifiedPageResp{}, err
	}
	return wire.VerifiedPageViewOf(core.ApproveVerifiedPage(ctx, id, req.Approved))
}
