// package: routes / invite HTTP handlers
// type:    logic
// job:     list invites, fetch a single invite, add/remove workshop members, and shared response types.
// limits:  does not create invites (-> invites_create.go) or accept/decline/revoke them (-> invites_accept.go).
package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/events"
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

	// Convert to response format (use request context so workshop/institution names are included)
	responses := make([]InviteResponse, len(invites))
	for i, inv := range invites {
		responses[i] = toInviteResponseWithContext(r.Context(), inv)
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

// AddMemberToWorkshopRequest represents the request to add an org member to a workshop
type AddMemberToWorkshopRequest struct {
	UserID string `json:"userId"`
}

// AddMemberToWorkshop godoc
//
//	@Summary		Add org member to workshop
//	@Description	Directly adds an organization individual to a workshop by setting their active workshop
//	@Tags			workshops
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Workshop ID"
//	@Param			body	body		AddMemberToWorkshopRequest		true	"User ID"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id}/members [post]
func AddMemberToWorkshop(w http.ResponseWriter, r *http.Request) {
	caller := httpx.UserFromRequest(r)

	workshopIDStr := httpx.PathParam(r, "id")
	workshopID, err := uuid.Parse(workshopIDStr)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	var req AddMemberToWorkshopRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" {
		httpx.WriteAppError(w, obj.ErrValidation("userId is required"))
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid user ID"))
		return
	}

	if err := db.AddMemberToWorkshop(r.Context(), caller.ID, workshopID, targetUserID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	events.GetBroker().PublishMembersUpdated(workshopID)

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Member added to workshop",
	})
}

// RemoveMemberFromWorkshop godoc
//
//	@Summary		Remove member from workshop
//	@Description	Removes a non-permanent member from a workshop by clearing their active workshop
//	@Tags			workshops
//	@Produce		json
//	@Param			id		path		string	true	"Workshop ID"
//	@Param			userId	path		string	true	"User ID to remove"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id}/members/{userId} [delete]
func RemoveMemberFromWorkshop(w http.ResponseWriter, r *http.Request) {
	caller := httpx.UserFromRequest(r)

	workshopIDStr := httpx.PathParam(r, "id")
	workshopID, err := uuid.Parse(workshopIDStr)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	userIDStr := httpx.PathParam(r, "userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid user ID"))
		return
	}

	if err := db.RemoveMemberFromWorkshop(r.Context(), caller.ID, workshopID, targetUserID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	events.GetBroker().PublishMembersUpdated(workshopID)

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Member removed from workshop",
	})
}
