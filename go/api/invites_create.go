// package: api / schemaf endpoints
// type:    logic
// job:     invites into organisations and workshops
// limits:  decides nothing: the core does (core/invite.go)
package api

import (
	"context"
	"time"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// CreateInstitutionInviteEndpoint invites an account, by id or email, into
// an organisation as head, staff or individual (staff by default). Head and
// staff invite into their organisation, only heads as head; the admin into
// any.
type CreateInstitutionInviteEndpoint struct{}

func (CreateInstitutionInviteEndpoint) Method() string             { return "POST" }
func (CreateInstitutionInviteEndpoint) Path() string               { return "/api/invites/institution" }
func (CreateInstitutionInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateInstitutionInviteEndpoint) Handle(ctx context.Context, req CreateInstitutionInviteReq) (InviteResp, error) {
	org, err := parseID(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return InviteResp{}, err
	}
	user, err := optionalID(req.InvitedUserID, "Invalid invited user ID")
	if err != nil {
		return InviteResp{}, err
	}
	role := req.Role
	if role == "" {
		role = "staff"
	}
	return wire.InviteCardViewOf(core.InviteToOrganisation(ctx, core.OrganisationInvite{
		Organisation: org, Role: role, User: user, Email: req.InvitedEmail,
	}))
}

// CreateInstitutionInviteReq is the organisation, the role, and the account
// invited, by id or email.
type CreateInstitutionInviteReq struct {
	InstitutionID string  `json:"institutionId"`
	Role          string  `json:"role"`
	InvitedUserID *string `json:"invitedUserId,omitempty"`
	InvitedEmail  *string `json:"invitedEmail,omitempty"`
}

// CreateWorkshopInviteEndpoint returns a workshop's link: its usable one, or
// a new one. Whoever follows it becomes a guest there, or, a member of an
// organisation, works in it.
type CreateWorkshopInviteEndpoint struct{}

func (CreateWorkshopInviteEndpoint) Method() string             { return "POST" }
func (CreateWorkshopInviteEndpoint) Path() string               { return "/api/invites/workshop" }
func (CreateWorkshopInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopInviteEndpoint) Handle(ctx context.Context, req CreateWorkshopInviteReq) (InviteResp, error) {
	workshop, err := parseID(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return InviteResp{}, err
	}
	var expires *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return InviteResp{}, errBadRequest("Invalid expiresAt format (use RFC3339)")
		}
		expires = &t
	}
	return wire.InviteCardViewOf(core.InviteToWorkshop(ctx, core.WorkshopLink{
		Workshop: workshop, MaxUses: req.MaxUses, Expires: expires,
	}))
}

// CreateWorkshopInviteReq is the workshop, and optionally how often and
// until when (RFC3339) its link may be followed.
type CreateWorkshopInviteReq struct {
	WorkshopID string  `json:"workshopId"`
	MaxUses    *int32  `json:"maxUses,omitempty"`
	ExpiresAt  *string `json:"expiresAt,omitempty"`
}

// CreateWorkshopEmailInviteEndpoint invites the account with this email
// into a workshop: accepting, they keep their role and work in it.
type CreateWorkshopEmailInviteEndpoint struct{}

func (CreateWorkshopEmailInviteEndpoint) Method() string             { return "POST" }
func (CreateWorkshopEmailInviteEndpoint) Path() string               { return "/api/invites/workshop/email" }
func (CreateWorkshopEmailInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopEmailInviteEndpoint) Handle(ctx context.Context, req CreateWorkshopEmailInviteReq) (InviteResp, error) {
	workshop, err := parseID(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return InviteResp{}, err
	}
	return wire.InviteCardViewOf(core.InviteToWorkshopByEmail(ctx, workshop, req.Email))
}

// CreateWorkshopEmailInviteReq is the workshop and the invited account's
// email.
type CreateWorkshopEmailInviteReq struct {
	WorkshopID string `json:"workshopId"`
	Email      string `json:"email"`
}
