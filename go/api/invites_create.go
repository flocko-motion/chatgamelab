// package: api / schemaf endpoints
// type:    logic
// job:     creates institution, workshop and workshop-email invites
// limits:  listing invites (-> invites.go); accepting, declining and revoking them (-> invites_accept.go)
package api

import (
	"context"
	"time"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// CreateInstitutionInviteEndpoint invites a user to join an institution as
// head, staff or individual.
//
// The role defaults to staff. The invite names the invited user by id or
// email.
type CreateInstitutionInviteEndpoint struct{}

func (CreateInstitutionInviteEndpoint) Method() string             { return "POST" }
func (CreateInstitutionInviteEndpoint) Path() string               { return "/api/invites/institution" }
func (CreateInstitutionInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateInstitutionInviteEndpoint) Handle(ctx context.Context, req CreateInstitutionInviteReq) (UserRoleInviteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}

	log.Debug("creating institution invite",
		"user_id", user.ID,
		"institution_id", req.InstitutionID,
		"role", req.Role,
		"invited_user_id", req.InvitedUserID,
		"invited_email", req.InvitedEmail,
	)

	institutionID, err := uuid.Parse(req.InstitutionID)
	if err != nil {
		log.Warn("invalid institution ID", "user_id", user.ID, "institution_id", req.InstitutionID, "error", err)
		return obj.UserRoleInvite{}, appError(obj.ErrValidation("Invalid institution ID"))
	}

	role := obj.RoleStaff
	if req.Role != "" {
		role = obj.Role(req.Role)
		if role != obj.RoleHead && role != obj.RoleStaff && role != obj.RoleIndividual {
			log.Warn("invalid role for institution invite", "user_id", user.ID, "role", req.Role)
			return obj.UserRoleInvite{}, appError(obj.ErrValidation("Role must be 'head', 'staff', or 'individual'"))
		}
	}

	var invitedUserID *uuid.UUID
	if req.InvitedUserID != nil {
		uid, err := uuid.Parse(*req.InvitedUserID)
		if err != nil {
			log.Warn("invalid invited user ID", "user_id", user.ID, "invited_user_id", *req.InvitedUserID, "error", err)
			return obj.UserRoleInvite{}, appError(obj.ErrValidation("Invalid invited user ID"))
		}
		invitedUserID = &uid
	}

	invite, err := db.CreateInstitutionInvite(ctx, user.ID, institutionID, role, invitedUserID, req.InvitedEmail)
	if err != nil {
		log.Warn("failed to create institution invite",
			"user_id", user.ID,
			"institution_id", institutionID,
			"role", role,
			"error", err,
		)
		return obj.UserRoleInvite{}, appError(err)
	}

	log.Info("institution invite created", "user_id", user.ID, "invite_id", invite.ID, "institution_id", institutionID)
	return invite, nil
}

// CreateInstitutionInviteReq is the institution, the role, and the invited
// user's id or email.
type CreateInstitutionInviteReq struct {
	InstitutionID string `json:"institutionId"`
	// Role is head, staff or individual; empty makes staff.
	Role          string  `json:"role"`
	InvitedUserID *string `json:"invitedUserId,omitempty"`
	InvitedEmail  *string `json:"invitedEmail,omitempty"`
}

// CreateWorkshopInviteEndpoint creates an open invite for users to join a
// workshop as participants.
type CreateWorkshopInviteEndpoint struct{}

func (CreateWorkshopInviteEndpoint) Method() string             { return "POST" }
func (CreateWorkshopInviteEndpoint) Path() string               { return "/api/invites/workshop" }
func (CreateWorkshopInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopInviteEndpoint) Handle(ctx context.Context, req CreateWorkshopInviteReq) (UserRoleInviteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}
	workshopID, err := parseIDValidation(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return obj.UserRoleInvite{}, err
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return obj.UserRoleInvite{}, appError(obj.ErrValidation("Invalid expiresAt format (use RFC3339)"))
		}
		expiresAt = &t
	}

	invite, err := db.CreateWorkshopInvite(ctx, user.ID, workshopID, req.MaxUses, expiresAt)
	if err != nil {
		return obj.UserRoleInvite{}, appError(err)
	}
	return invite, nil
}

// CreateWorkshopInviteReq is the workshop, and optionally how often and until
// when the invite can be used.
type CreateWorkshopInviteReq struct {
	WorkshopID string `json:"workshopId"`
	MaxUses    *int32 `json:"maxUses,omitempty"`
	// ExpiresAt is an RFC 3339 time.
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// CreateWorkshopEmailInviteEndpoint invites a registered user, by email, to
// join a workshop.
//
// A second pending invite for the same user and workshop is refused with 409.
type CreateWorkshopEmailInviteEndpoint struct{}

func (CreateWorkshopEmailInviteEndpoint) Method() string             { return "POST" }
func (CreateWorkshopEmailInviteEndpoint) Path() string               { return "/api/invites/workshop/email" }
func (CreateWorkshopEmailInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopEmailInviteEndpoint) Handle(ctx context.Context, req CreateWorkshopEmailInviteReq) (InviteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return InviteResp{}, err
	}
	if req.WorkshopID == "" || req.Email == "" {
		return InviteResp{}, appError(obj.ErrValidation("workshopId and email are required"))
	}
	workshopID, err := parseIDValidation(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return InviteResp{}, err
	}
	invite, err := db.CreateWorkshopEmailInvite(ctx, user.ID, workshopID, req.Email)
	if err != nil {
		return InviteResp{}, appError(err)
	}
	return toInviteResp(ctx, invite), nil
}

// CreateWorkshopEmailInviteReq is the workshop and the invited user's email.
type CreateWorkshopEmailInviteReq struct {
	WorkshopID string `json:"workshopId"`
	Email      string `json:"email"`
}
