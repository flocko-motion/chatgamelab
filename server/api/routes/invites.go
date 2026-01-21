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

// AcceptInstitutionInvite godoc
//
//	@Summary		Accept institution invite
//	@Description	Accepts a targeted institution invite by ID or token
//	@Tags			invites
//	@Produce		json
//	@Param			idOrToken	path		string	true	"Invite ID (UUID) or token"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403			{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404			{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/institution/{idOrToken}/accept [post]
func AcceptInstitutionInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	idOrToken := httpx.PathParam(r, "idOrToken")
	if idOrToken == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Missing idOrToken parameter"))
		return
	}

	// Try to parse as UUID first
	inviteID, err := uuid.Parse(idOrToken)
	var token string
	if err != nil {
		// Not a UUID, treat as token
		token = idOrToken
		inviteID = uuid.Nil
	}

	roleID, err := db.AcceptTargetedInvite(r.Context(), inviteID, token, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"roleId":  roleID.String(),
		"message": "Invite accepted successfully",
	})
}

// DeclineInstitutionInvite godoc
//
//	@Summary		Decline institution invite
//	@Description	Declines a targeted institution invite by ID or token
//	@Tags			invites
//	@Produce		json
//	@Param			idOrToken	path		string	true	"Invite ID (UUID) or token"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403			{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404			{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/institution/{idOrToken}/decline [post]
func DeclineInstitutionInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	idOrToken := httpx.PathParam(r, "idOrToken")
	if idOrToken == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Missing idOrToken parameter"))
		return
	}

	// Try to parse as UUID first
	inviteID, err := uuid.Parse(idOrToken)
	var token string
	if err != nil {
		// Not a UUID, treat as token
		token = idOrToken
		inviteID = uuid.Nil
	}

	err = db.DeclineTargetedInvite(r.Context(), inviteID, token, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Invite declined",
	})
}

// AcceptWorkshopInvite godoc
//
//	@Summary		Accept workshop invite
//	@Description	Accepts an open workshop invite by token
//	@Tags			invites
//	@Produce		json
//	@Param			token	path		string	true	"Invite token"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		404		{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/workshop/{token}/accept [post]
func AcceptWorkshopInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	token := httpx.PathParam(r, "token")
	if token == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Missing token parameter"))
		return
	}

	roleID, err := db.AcceptOpenInvite(r.Context(), token, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"roleId":  roleID.String(),
		"message": "Joined workshop successfully",
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
