// package: api / schemaf endpoints
// type:    logic
// job:     lists and reads invites
// limits:  decides nothing: the core does (core/invite.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// InviteResp is an invite as the caller sees it.
type InviteResp = wire.InviteCardView

// InvitesResp is a list of invites.
type InvitesResp = []wire.InviteCardView

// ListInvitesReq takes no parameters.
type ListInvitesReq struct{}

// ListInvitesEndpoint lists the pending invites addressed to the caller, by
// account or email: what they may accept or decline.
type ListInvitesEndpoint struct{}

func (ListInvitesEndpoint) Method() string             { return "GET" }
func (ListInvitesEndpoint) Path() string               { return "/api/invites" }
func (ListInvitesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInvitesEndpoint) Handle(ctx context.Context, _ ListInvitesReq) (InvitesResp, error) {
	return wire.InviteCardViewsOf(core.ListInvites(ctx, core.InviteFilter{Mine: true}))
}

// ListAllInvitesEndpoint lists a page of every invite the caller may list,
// and the total: for the admin, all of them.
type ListAllInvitesEndpoint struct{}

func (ListAllInvitesEndpoint) Method() string             { return "GET" }
func (ListAllInvitesEndpoint) Path() string               { return "/api/invites/all" }
func (ListAllInvitesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListAllInvitesEndpoint) Handle(ctx context.Context, req ListAllInvitesReq) (InvitePageResp, error) {
	return wire.InviteCardViewPageOf(core.List(ctx, core.Invites(core.InviteFilter{}), req.query()))
}

// InvitePageResp is a page of invites and the total.
type InvitePageResp = wire.Page[wire.InviteCardView]

// InviteSort orders an invite list; the default is modifiedAt, newest first.
type InviteSort string

const (
	InviteSortRole         InviteSort = "role"
	InviteSortOrganisation InviteSort = "organisation"
	InviteSortStatus       InviteSort = "status"
	InviteSortModifiedAt   InviteSort = "modifiedAt"
)

// ListAllInvitesReq sorts and pages the invite list.
type ListAllInvitesReq struct {
	ListParams[NoFilter, InviteSort]
}

// InstitutionInvitesReq names an organisation, and sorts and pages its invites.
type InstitutionInvitesReq struct {
	InstitutionID string `path:"institutionId"`
	ListParams[NoFilter, InviteSort]
}

// ListInvitesByInstitutionEndpoint lists a page of an organisation's invites
// the caller may list, and the total: its head and staff (staff without the invites to become
// head), and the admin.
type ListInvitesByInstitutionEndpoint struct{}

func (ListInvitesByInstitutionEndpoint) Method() string { return "GET" }
func (ListInvitesByInstitutionEndpoint) Path() string {
	return "/api/invites/institution/{institutionId}"
}
func (ListInvitesByInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInvitesByInstitutionEndpoint) Handle(ctx context.Context, req InstitutionInvitesReq) (InvitePageResp, error) {
	org, err := parseID(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return InvitePageResp{}, err
	}
	return wire.InviteCardViewPageOf(core.List(ctx, core.Invites(core.InviteFilter{Organisation: &org}), req.query()))
}

// InviteIDOrTokenReq names an invite by its id, or by its link's token.
type InviteIDOrTokenReq struct {
	IDOrToken string `path:"idOrToken"`
}

// GetInviteEndpoint returns an invite: by id, for those who may read it; by
// its link's token, for anyone who has it, signed in or not. Repeated
// guesses at tokens are slowed down (429).
type GetInviteEndpoint struct{}

func (GetInviteEndpoint) Method() string             { return "GET" }
func (GetInviteEndpoint) Path() string               { return "/api/invites/{idOrToken}" }
func (GetInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (GetInviteEndpoint) Handle(ctx context.Context, req InviteIDOrTokenReq) (InviteResp, error) {
	if err := tokenLocked(); err != nil {
		return InviteResp{}, err
	}
	return wire.InviteCardViewOf(core.ReadInviteRef(ctx, req.IDOrToken))
}
