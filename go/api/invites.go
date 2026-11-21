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

// ListAllInvitesEndpoint lists every invite the caller may list: for the
// admin, all of them.
type ListAllInvitesEndpoint struct{}

func (ListAllInvitesEndpoint) Method() string             { return "GET" }
func (ListAllInvitesEndpoint) Path() string               { return "/api/invites/all" }
func (ListAllInvitesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListAllInvitesEndpoint) Handle(ctx context.Context, _ ListInvitesReq) (InvitesResp, error) {
	return wire.InviteCardViewsOf(core.ListInvites(ctx, core.InviteFilter{}))
}

// InstitutionInvitesReq names an organisation.
type InstitutionInvitesReq struct {
	InstitutionID string `path:"institutionId"`
}

// ListInvitesByInstitutionEndpoint lists an organisation's invites the
// caller may list: its head and staff (staff without the invites to become
// head), and the admin.
type ListInvitesByInstitutionEndpoint struct{}

func (ListInvitesByInstitutionEndpoint) Method() string { return "GET" }
func (ListInvitesByInstitutionEndpoint) Path() string {
	return "/api/invites/institution/{institutionId}"
}
func (ListInvitesByInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInvitesByInstitutionEndpoint) Handle(ctx context.Context, req InstitutionInvitesReq) (InvitesResp, error) {
	org, err := parseID(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return nil, err
	}
	return wire.InviteCardViewsOf(core.ListInvites(ctx, core.InviteFilter{Organisation: &org}))
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
