package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"
	"context"
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

// toInviteResponse converts an obj.UserRoleInvite to InviteResponse
func toInviteResponse(inv obj.UserRoleInvite) InviteResponse {
	return toInviteResponseWithContext(context.Background(), inv)
}

// toInviteResponseWithContext converts an obj.UserRoleInvite to InviteResponse with context for fetching names
func toInviteResponseWithContext(ctx context.Context, inv obj.UserRoleInvite) InviteResponse {
	resp := InviteResponse{
		ID:            inv.ID.String(),
		InstitutionID: inv.InstitutionID.String(),
		Role:          string(inv.Role),
		UsesCount:     inv.UsesCount,
		Status:        string(inv.Status),
	}

	// Fetch institution name (no permission check - just for display)
	if name, err := db.GetInstitutionName(ctx, inv.InstitutionID); err == nil {
		resp.InstitutionName = &name
	}

	if inv.WorkshopID != nil {
		wsID := inv.WorkshopID.String()
		resp.WorkshopID = &wsID
		// Fetch workshop name (no permission check - just for display)
		if name, err := db.GetWorkshopName(ctx, *inv.WorkshopID); err == nil {
			resp.WorkshopName = &name
		}
	}
	if inv.InvitedUserID != nil {
		userID := inv.InvitedUserID.String()
		resp.InvitedUserID = &userID
	}
	if inv.InvitedEmail != nil {
		resp.InvitedEmail = inv.InvitedEmail
	}
	if inv.InviteToken != nil {
		resp.InviteToken = inv.InviteToken
	}
	if inv.MaxUses != nil {
		resp.MaxUses = inv.MaxUses
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

// ListInvites godoc
//
//	@Summary		List incoming invites for the current user
//	@Description	Lists invites scoped by user permissions - shows invites for the current user to join an institution
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

	// Convert to response format
	responses := make([]InviteResponse, len(invites))
	for i, inv := range invites {
		responses[i] = toInviteResponse(inv)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// ListAllInvites godoc
//
//	@Summary		List all invites (admin only)
//	@Description	Lists all invites. Requires admin role.
//	@Tags			invites
//	@Produce		json
//	@Success		200		{array}		InviteResponse
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden"
//	@Security		BearerAuth
//	@Router			/invites/all [get]
func ListAllInvites(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	invites, err := db.GetAllInvites(r.Context(), user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response format
	responses := make([]InviteResponse, len(invites))
	for i, inv := range invites {
		responses[i] = toInviteResponse(inv)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// ListInvitesByInstitution godoc
//
//	@Summary		List outgoing invites from an institution
//	@Description	Lists all invites that invite users to join a specific institution. Requires head/staff role in the institution or admin.
//	@Tags			invites
//	@Produce		json
//	@Param			institutionId	path		string	true	"Institution ID"
//	@Success		200				{array}		InviteResponse
//	@Failure		401				{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403				{object}	httpx.ErrorResponse	"Forbidden"
//	@Security		BearerAuth
//	@Router			/invites/institution/{institutionId} [get]
func ListInvitesByInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "institutionId")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	invites, err := db.GetInvitesByInstitutionID(r.Context(), user.ID, institutionID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response format
	responses := make([]InviteResponse, len(invites))
	for i, inv := range invites {
		responses[i] = toInviteResponse(inv)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
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
	user := httpx.MaybeUserFromRequest(r) // Optional - anonymous users can view invite details by token

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
		// It's a UUID - get by ID (requires auth)
		if user == nil {
			httpx.WriteAppError(w, obj.ErrUnauthorized("Authentication required to view invite by ID"))
			return
		}
		invite, err = db.GetInviteByID(r.Context(), user.ID, inviteID)
	} else {
		// Not a UUID - treat as token (can be anonymous for open invites)
		var userID uuid.UUID
		if user != nil {
			userID = user.ID
		}
		invite, err = db.GetInviteByToken(r.Context(), userID, idOrToken)
	}

	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toInviteResponseWithContext(r.Context(), invite))
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

	// Role is required - must be head or staff (individuals join workshops via open token links)
	var role obj.Role
	if req.Role != "" {
		role = obj.Role(req.Role)
		if role != obj.RoleHead && role != obj.RoleStaff {
			log.Warn("invalid role for institution invite", "user_id", user.ID, "role", req.Role)
			httpx.WriteAppError(w, obj.ErrValidation("Role must be 'head' or 'staff'"))
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

// AcceptInvite godoc
//
//	@Summary		Accept invite
//	@Description	Accepts an invite by ID or token. For workshop invites, can be used anonymously (creates ad-hoc user). For institution invites, requires authentication.
//	@Tags			invites
//	@Produce		json
//	@Param			idOrToken	path		string	true	"Invite ID (UUID) or token"
//	@Success		200			{object}	AcceptInviteResponse
//	@Failure		400			{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403			{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404			{object}	httpx.ErrorResponse	"Not found"
//	@Security		BearerAuth
//	@Router			/invites/{idOrToken}/accept [post]
func AcceptInvite(w http.ResponseWriter, r *http.Request) {
	user := httpx.MaybeUserFromRequest(r) // Optional auth - may be nil for anonymous workshop invites

	idOrToken := httpx.PathParam(r, "idOrToken")

	// Try to parse as UUID first
	inviteID, err := uuid.Parse(idOrToken)
	isToken := err != nil

	if isToken {
		// It's a token - check if it's a workshop invite that can be accepted anonymously
		if user == nil {
			// Anonymous user accepting workshop invite
			createdUser, authToken, err := db.AcceptWorkshopInviteAnonymously(r.Context(), idOrToken)
			if err != nil {
				if appErr, ok := err.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}

			// Set HttpOnly cookie for participant session
			// The authToken is the participant token (starts with "participant-")
			httpx.SetSessionCookie(w, r, authToken)

			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				User:      createdUser,
				AuthToken: &authToken,
				Message:   "Workshop invite accepted, user created",
			})
			return
		}

		// Authenticated user accepting open invite by token
		// Participants switch workshops (new role assignment)
		if user.Role != nil && user.Role.Role == obj.RoleParticipant {
			_, acceptErr := db.AcceptOpenInvite(r.Context(), idOrToken, user.ID)
			if acceptErr != nil {
				if appErr, ok := acceptErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
				return
			}
			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				Message: "Invite accepted",
			})
			return
		}

		// Head, staff, individual enter workshop mode (keep their role, set active workshop)
		if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff || user.Role.Role == obj.RoleIndividual) {
			invite, getErr := db.GetInviteByToken(r.Context(), uuid.Nil, idOrToken)
			if getErr != nil {
				if appErr, ok := getErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, getErr.Error())
				return
			}
			if invite.WorkshopID == nil {
				httpx.WriteAppError(w, obj.ErrValidation("this is not a workshop invite"))
				return
			}
			setErr := db.SetActiveWorkshop(r.Context(), user.ID, *invite.WorkshopID)
			if setErr != nil {
				if appErr, ok := setErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, setErr.Error())
				return
			}
			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				Message: "Workshop entered",
			})
			return
		}

		// Users without a recognized role (e.g., no role at all) join as participant
		_, acceptErr := db.AcceptOpenInvite(r.Context(), idOrToken, user.ID)
		if acceptErr != nil {
			if appErr, ok := acceptErr.(*obj.AppError); ok {
				httpx.WriteAppError(w, appErr)
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
			Message: "Invite accepted",
		})
		return
	}

	// It's a UUID - requires authentication
	if user == nil {
		httpx.WriteAppError(w, obj.ErrUnauthorized("Authentication required for targeted invites"))
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
	if invite.InviteToken != nil {
		// Open invite (workshop) — head/staff/individual enter workshop mode, others join as participant
		if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff || user.Role.Role == obj.RoleIndividual) {
			if invite.WorkshopID == nil {
				httpx.WriteAppError(w, obj.ErrValidation("this is not a workshop invite"))
				return
			}
			setErr := db.SetActiveWorkshop(r.Context(), user.ID, *invite.WorkshopID)
			if setErr != nil {
				if appErr, ok := setErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, setErr.Error())
				return
			}
			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				Message: "Workshop entered",
			})
			return
		}
		_, acceptErr := db.AcceptOpenInvite(r.Context(), *invite.InviteToken, user.ID)
		if acceptErr != nil {
			if appErr, ok := acceptErr.(*obj.AppError); ok {
				httpx.WriteAppError(w, appErr)
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
			return
		}
	} else {
		// Targeted invite (institution) - accept by ID
		_, acceptErr := db.AcceptTargetedInvite(r.Context(), inviteID, "", user.ID)
		if acceptErr != nil {
			if appErr, ok := acceptErr.(*obj.AppError); ok {
				httpx.WriteAppError(w, appErr)
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
			return
		}
	}

	// Success - return a simple response (don't refetch invite as permissions may have changed)
	httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
		Message: "Invite accepted",
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
//	@Description	Revokes a pending invite (creator, admin, or institution staff for workshop invites)
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

// AcceptInviteResponse represents the response when accepting an invite
type AcceptInviteResponse struct {
	User      *obj.User `json:"user,omitempty"`
	AuthToken *string   `json:"authToken,omitempty"`
	Message   string    `json:"message"`
}
