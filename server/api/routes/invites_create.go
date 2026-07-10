// package: routes / invite HTTP handlers
// type:    logic
// job:     HTTP handlers for creating institution, workshop, and workshop-email invites.
// limits:  does not list, accept, or revoke invites (-> invites.go, invites_accept.go).
package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CreateInstitutionInvite godoc
//
//	@Summary		Create institution invite
//	@Description	Creates a targeted invite for a user to join an institution as head or staff
//	@Tags			invites
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateInstitutionInviteRequest	true	"Invite details"
//	@Success		200		{object}	InviteResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/institution [post]
func CreateInstitutionInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateInstitutionInviteRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		log.Warn("failed to parse institution invite request", "user_id", user.ID, "error", err)
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
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
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	// Role is required - must be head, staff, or individual
	var role obj.Role
	if req.Role != "" {
		role = obj.Role(req.Role)
		if role != obj.RoleHead && role != obj.RoleStaff && role != obj.RoleIndividual {
			log.Warn("invalid role for institution invite", "user_id", user.ID, "role", req.Role)
			httpx.WriteAppError(w, obj.ErrValidation("Role must be 'head', 'staff', or 'individual'"))
			return
		}
	} else {
		// Default to staff if no role specified
		role = obj.RoleStaff
	}

	var invitedUserID *uuid.UUID
	if req.InvitedUserID != nil {
		uid, err := uuid.Parse(*req.InvitedUserID)
		if err != nil {
			log.Warn("invalid invited user ID", "user_id", user.ID, "invited_user_id", *req.InvitedUserID, "error", err)
			httpx.WriteAppError(w, obj.ErrValidation("Invalid invited user ID"))
			return
		}
		invitedUserID = &uid
	}

	invite, err := db.CreateInstitutionInvite(r.Context(), user.ID, institutionID, role, invitedUserID, req.InvitedEmail)
	if err != nil {
		log.Warn("failed to create institution invite",
			"user_id", user.ID,
			"institution_id", institutionID,
			"role", role,
			"error", err,
		)
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("institution invite created", "user_id", user.ID, "invite_id", invite.ID, "institution_id", institutionID)
	httpx.WriteJSON(w, http.StatusOK, invite)
}

// CreateWorkshopInvite godoc
//
//	@Summary		Create workshop invite
//	@Description	Creates an open invite for users to join a workshop as participants
//	@Tags			invites
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateWorkshopInviteRequest	true	"Invite details"
//	@Success		200		{object}	InviteResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/workshop [post]
func CreateWorkshopInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateWorkshopInviteRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	workshopID, err := uuid.Parse(req.WorkshopID)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httpx.WriteAppError(w, obj.ErrValidation("Invalid expiresAt format (use RFC3339)"))
			return
		}
		expiresAt = &t
	}

	invite, err := db.CreateWorkshopInvite(r.Context(), user.ID, workshopID, req.MaxUses, expiresAt)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, invite)
}

// CreateWorkshopEmailInviteRequest represents the request to invite a user to a workshop by email
type CreateWorkshopEmailInviteRequest struct {
	WorkshopID string `json:"workshopId"`
	Email      string `json:"email"`
}

// CreateWorkshopEmailInvite godoc
//
//	@Summary		Create workshop email invite
//	@Description	Creates a targeted invite for a registered user (by email) to join a workshop
//	@Tags			invites
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateWorkshopEmailInviteRequest	true	"Workshop ID and email"
//	@Success		201		{object}	InviteResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse	"Duplicate invite"
//	@Security		BearerAuth
//	@Router			/invites/workshop/email [post]
func CreateWorkshopEmailInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateWorkshopEmailInviteRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkshopID == "" || req.Email == "" {
		httpx.WriteAppError(w, obj.ErrValidation("workshopId and email are required"))
		return
	}

	workshopID, err := uuid.Parse(req.WorkshopID)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	invite, err := db.CreateWorkshopEmailInvite(r.Context(), user.ID, workshopID, req.Email)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toInviteResponseWithContext(r.Context(), invite))
}
