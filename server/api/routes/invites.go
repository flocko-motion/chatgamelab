package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CreateInstitutionInviteRequest represents the request to create an institution invite
type CreateInstitutionInviteRequest struct {
	InstitutionID string  `json:"institutionId"`
	Role          string  `json:"role"`
	InvitedUserID *string `json:"invitedUserId,omitempty"`
	InvitedEmail  *string `json:"invitedEmail,omitempty"`
}

// CreateWorkshopInviteRequest represents the request to create a workshop invite
type CreateWorkshopInviteRequest struct {
	WorkshopID string  `json:"workshopId"`
	MaxUses    *int32  `json:"maxUses,omitempty"`
	ExpiresAt  *string `json:"expiresAt,omitempty"`
}

// InviteResponse represents an invite response
type InviteResponse struct {
	ID            string  `json:"id"`
	InstitutionID string  `json:"institutionId"`
	Role          string  `json:"role"`
	WorkshopID    *string `json:"workshopId,omitempty"`
	InvitedUserID *string `json:"invitedUserId,omitempty"`
	InvitedEmail  *string `json:"invitedEmail,omitempty"`
	InviteToken   *string `json:"inviteToken,omitempty"`
	MaxUses       *int32  `json:"maxUses,omitempty"`
	UsesCount     int32   `json:"usesCount"`
	ExpiresAt     *string `json:"expiresAt,omitempty"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
}

// ListInvites godoc
//
//	@Summary		List invites
//	@Description	Lists invites scoped by user permissions. Admins see all invites, regular users see only their own pending invites.
//	@Tags			invites
//	@Produce		json
//	@Success		200		{array}		InviteResponse
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Security		BearerAuth
//	@Router			/invites [get]
func ListInvites(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	invites, err := db.GetInvites(r.Context(), user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, invites)
}

// GetInvite godoc
//
//	@Summary		Get invite by ID or token
//	@Description	Retrieves a specific invite. Auto-detects whether parameter is a UUID (ID) or string (token). Admins can see any invite, regular users can only see invites targeted to them or created by them.
//	@Tags			invites
//	@Produce		json
//	@Param			idOrToken	path		string	true	"Invite ID (UUID) or token"
//	@Success		200			{object}	obj.UserRoleInvite
//	@Failure		401			{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403			{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404			{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/{idOrToken} [get]
func GetInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	idOrToken := httpx.PathParam(r, "idOrToken")
	if idOrToken == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Missing invite ID or token"))
		return
	}

	// Try to parse as UUID first
	var invite obj.UserRoleInvite
	var err error

	inviteID, parseErr := uuid.Parse(idOrToken)
	if parseErr == nil {
		// It's a UUID - get by ID
		invite, err = db.GetInviteByID(r.Context(), user.ID, inviteID)
	} else {
		// Not a UUID - treat as token
		invite, err = db.GetInviteByToken(r.Context(), user.ID, idOrToken)
	}

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
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	institutionID, err := uuid.Parse(req.InstitutionID)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	role := obj.Role(req.Role)
	if role != obj.RoleHead && role != obj.RoleStaff {
		httpx.WriteAppError(w, obj.ErrValidation("Role must be 'head' or 'staff'"))
		return
	}

	var invitedUserID *uuid.UUID
	if req.InvitedUserID != nil {
		uid, err := uuid.Parse(*req.InvitedUserID)
		if err != nil {
			httpx.WriteAppError(w, obj.ErrValidation("Invalid invited user ID"))
			return
		}
		invitedUserID = &uid
	}

	invite, err := db.CreateInstitutionInvite(r.Context(), user.ID, institutionID, role, invitedUserID, req.InvitedEmail)
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

// AcceptInvite godoc
//
//	@Summary		Accept invite
//	@Description	Accepts an invite by ID. Use GET /invites/{token} to look up the ID first if you have a token.
//	@Tags			invites
//	@Produce		json
//	@Param			id	path		string	true	"Invite ID (UUID)"
//	@Success		200	{object}	obj.UserRoleInvite
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/{id}/accept [post]
func AcceptInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	idStr := httpx.PathParam(r, "id")
	inviteID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid invite ID"))
		return
	}

	// First, get the invite to determine its type
	invite, err := db.GetInviteByID(r.Context(), user.ID, inviteID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine invite type and accept accordingly
	var acceptErr error
	if invite.InviteToken != nil {
		// Open invite (workshop) - accept by token
		_, acceptErr = db.AcceptOpenInvite(r.Context(), *invite.InviteToken, user.ID)
	} else {
		// Targeted invite (institution) - accept by ID
		_, acceptErr = db.AcceptTargetedInvite(r.Context(), inviteID, "", user.ID)
	}

	if acceptErr != nil {
		if appErr, ok := acceptErr.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
		return
	}

	// Success - return accepted status
	httpx.WriteJSON(w, http.StatusOK, obj.UserRoleInvite{
		ID:     inviteID,
		Status: obj.InviteStatusAccepted,
	})
}

// DeclineInvite godoc
//
//	@Summary		Decline invite
//	@Description	Declines an invite by ID
//	@Tags			invites
//	@Produce		json
//	@Param			id	path		string	true	"Invite ID (UUID)"
//	@Success		200	{object}	obj.UserRoleInvite
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/{id}/decline [post]
func DeclineInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	idStr := httpx.PathParam(r, "id")
	inviteID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid invite ID"))
		return
	}

	// First, get the invite to validate permissions
	_, err = db.GetInviteByID(r.Context(), user.ID, inviteID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Decline the invite (works for both targeted and open invites)
	err = db.DeclineTargetedInvite(r.Context(), inviteID, "", user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Success - return declined status
	httpx.WriteJSON(w, http.StatusOK, obj.UserRoleInvite{
		ID:     inviteID,
		Status: obj.InviteStatusDeclined,
	})
}

// RevokeInvite godoc
//
//	@Summary		Revoke invite
//	@Description	Revokes a pending invite (creator or admin only)
//	@Tags			invites
//	@Produce		json
//	@Param			id	path		string	true	"Invite ID (UUID)"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/{id} [delete]
func RevokeInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	inviteID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid invite ID"))
		return
	}

	err = db.RevokeInvite(r.Context(), inviteID, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Invite revoked",
	})
}
