// package: api / schemaf endpoints
// type:    logic
// job:     lists invites, fetches a single invite, and the invite shape the invite endpoints answer with
// limits:  creating invites (-> invites_create.go); accepting, declining and revoking them (-> invites_accept.go)
package api

import (
	"context"
	"time"

	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// InviteResp is an invite with the names of its institution and workshop.
type InviteResp struct {
	ID              string  `json:"id"`
	InstitutionID   string  `json:"institutionId"`
	InstitutionName *string `json:"institutionName,omitempty"`
	Role            string  `json:"role"`
	WorkshopID      *string `json:"workshopId,omitempty"`
	WorkshopName    *string `json:"workshopName,omitempty"`
	InvitedUserID   *string `json:"invitedUserId,omitempty"`
	InvitedEmail    *string `json:"invitedEmail,omitempty"`
	InviteToken     *string `json:"inviteToken,omitempty"`
	MaxUses         *int32  `json:"maxUses,omitempty"`
	UsesCount       int32   `json:"usesCount"`
	ExpiresAt       *string `json:"expiresAt,omitempty"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"createdAt"`
	ModifiedAt      *string `json:"modifiedAt,omitempty"`
}

// InvitesResp is a list of invites.
type InvitesResp = []InviteResp

// UserRoleInviteResp is an invite as stored.
type UserRoleInviteResp = obj.UserRoleInvite

// toInviteResp turns an invite into its response, with the names of its
// institution and workshop for display (no permission check).
func toInviteResp(ctx context.Context, inv obj.UserRoleInvite) InviteResp {
	resp := InviteResp{
		ID:            inv.ID.String(),
		InstitutionID: inv.InstitutionID.String(),
		Role:          string(inv.Role),
		UsesCount:     inv.UsesCount,
		Status:        string(inv.Status),
		InvitedEmail:  inv.InvitedEmail,
		InviteToken:   inv.InviteToken,
		MaxUses:       inv.MaxUses,
	}
	if name, err := db.GetInstitutionName(ctx, inv.InstitutionID); err == nil {
		resp.InstitutionName = &name
	}
	if inv.WorkshopID != nil {
		wsID := inv.WorkshopID.String()
		resp.WorkshopID = &wsID
		if name, err := db.GetWorkshopName(ctx, *inv.WorkshopID); err == nil {
			resp.WorkshopName = &name
		}
	}
	if inv.InvitedUserID != nil {
		userID := inv.InvitedUserID.String()
		resp.InvitedUserID = &userID
	}
	if inv.ExpiresAt != nil {
		expiresAt := inv.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &expiresAt
	}
	if inv.Meta.CreatedAt != nil {
		resp.CreatedAt = inv.Meta.CreatedAt.Format(time.RFC3339)
	}
	if inv.Meta.ModifiedAt != nil {
		modifiedAt := inv.Meta.ModifiedAt.Format(time.RFC3339)
		resp.ModifiedAt = &modifiedAt
	}
	return resp
}

// toInvitesResp turns a list of invites into their responses.
func toInvitesResp(ctx context.Context, invites []obj.UserRoleInvite) InvitesResp {
	resp := make(InvitesResp, len(invites))
	for i, inv := range invites {
		resp[i] = toInviteResp(ctx, inv)
	}
	return resp
}

// ListInvitesReq takes no parameters.
type ListInvitesReq struct{}

// ListInvitesEndpoint lists the invites addressed to the caller.
type ListInvitesEndpoint struct{}

func (ListInvitesEndpoint) Method() string             { return "GET" }
func (ListInvitesEndpoint) Path() string               { return "/api/invites" }
func (ListInvitesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInvitesEndpoint) Handle(ctx context.Context, _ ListInvitesReq) (InvitesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	invites, err := db.GetInvites(ctx, user.ID)
	if err != nil {
		return nil, appError(err)
	}
	return toInvitesResp(ctx, invites), nil
}

// ListAllInvitesEndpoint lists every invite; admins only.
type ListAllInvitesEndpoint struct{}

func (ListAllInvitesEndpoint) Method() string             { return "GET" }
func (ListAllInvitesEndpoint) Path() string               { return "/api/invites/all" }
func (ListAllInvitesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListAllInvitesEndpoint) Handle(ctx context.Context, _ ListInvitesReq) (InvitesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	invites, err := db.GetAllInvites(ctx, user.ID)
	if err != nil {
		return nil, appError(err)
	}
	return toInvitesResp(ctx, invites), nil
}

// InstitutionInvitesReq names an institution by its id.
type InstitutionInvitesReq struct {
	InstitutionID string `path:"institutionId" json:"-"`
}

// ListInvitesByInstitutionEndpoint lists the invites to join an institution.
//
// The caller is head or staff of the institution, or an admin.
type ListInvitesByInstitutionEndpoint struct{}

func (ListInvitesByInstitutionEndpoint) Method() string { return "GET" }
func (ListInvitesByInstitutionEndpoint) Path() string {
	return "/api/invites/institution/{institutionId}"
}
func (ListInvitesByInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInvitesByInstitutionEndpoint) Handle(ctx context.Context, req InstitutionInvitesReq) (InvitesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	institutionID, err := parseIDValidation(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return nil, err
	}
	invites, err := db.GetInvitesByInstitutionID(ctx, user.ID, institutionID)
	if err != nil {
		return nil, appError(err)
	}
	return toInvitesResp(ctx, invites), nil
}

// InviteIDOrTokenReq names an invite by its id or its token.
type InviteIDOrTokenReq struct {
	IDOrToken string `path:"idOrToken" json:"-"`
}

// GetInviteEndpoint returns an invite by its id or its token.
//
// An id needs a signed-in caller: admins see any invite, everyone else the
// invites addressed to them or created by them. A token also answers an
// anonymous caller, for open invites.
type GetInviteEndpoint struct{}

func (GetInviteEndpoint) Method() string             { return "GET" }
func (GetInviteEndpoint) Path() string               { return "/api/invites/{idOrToken}" }
func (GetInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (GetInviteEndpoint) Handle(ctx context.Context, req InviteIDOrTokenReq) (InviteResp, error) {
	if err := tokenLocked(); err != nil {
		return InviteResp{}, err
	}
	user, err := maybeCaller(ctx)
	if err != nil {
		return InviteResp{}, err
	}
	if req.IDOrToken == "" {
		return InviteResp{}, appError(obj.ErrValidation("Missing invite ID or token"))
	}

	var invite obj.UserRoleInvite
	if inviteID, parseErr := uuid.Parse(req.IDOrToken); parseErr == nil {
		if user == nil {
			return InviteResp{}, appError(obj.ErrUnauthorized("Authentication required to view invite by ID"))
		}
		invite, err = db.GetInviteByID(ctx, user.ID, inviteID)
	} else {
		var userID uuid.UUID
		if user != nil {
			userID = user.ID
		}
		token, _ := db.ResolveInviteToken(ctx, req.IDOrToken)
		invite, err = db.GetInviteByToken(ctx, userID, token)
	}
	if err != nil {
		return InviteResp{}, appError(err)
	}
	return toInviteResp(ctx, invite), nil
}
