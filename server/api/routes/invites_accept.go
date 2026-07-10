// package: routes / invite HTTP handlers
// type:    logic
// job:     HTTP handlers for accepting, declining, and revoking invites.
// limits:  does not create or list invites (-> invites_create.go, invites.go).
package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/events"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

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

			// Notify workshop members about the new participant
			if createdUser.Role != nil && createdUser.Role.Workshop != nil {
				events.GetBroker().PublishMembersUpdated(createdUser.Role.Workshop.ID)
			}

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
			// Look up invite to get workshop ID for SSE notification
			inviteForSSE, _ := db.GetInviteByToken(r.Context(), uuid.Nil, idOrToken)

			_, acceptErr := db.AcceptOpenInvite(r.Context(), idOrToken, user.ID)
			if acceptErr != nil {
				if appErr, ok := acceptErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
				return
			}

			// Notify old and new workshop about member change
			if user.Role.Workshop != nil {
				events.GetBroker().PublishMembersUpdated(user.Role.Workshop.ID)
			}
			if inviteForSSE.WorkshopID != nil {
				events.GetBroker().PublishMembersUpdated(*inviteForSSE.WorkshopID)
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
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				Message: "Workshop entered",
			})
			return
		}

		// Users without a recognized role (e.g., no role at all) join as participant
		inviteForJoin, _ := db.GetInviteByToken(r.Context(), uuid.Nil, idOrToken)

		_, acceptErr := db.AcceptOpenInvite(r.Context(), idOrToken, user.ID)
		if acceptErr != nil {
			if appErr, ok := acceptErr.(*obj.AppError); ok {
				httpx.WriteAppError(w, appErr)
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, acceptErr.Error())
			return
		}

		if inviteForJoin.WorkshopID != nil {
			events.GetBroker().PublishMembersUpdated(*inviteForJoin.WorkshopID)
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
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
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
		if invite.WorkshopID != nil {
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
		}
	} else {
		// Targeted invite - check if it's a workshop-scoped invite for an existing head/staff/individual
		if invite.WorkshopID != nil && user.Role != nil &&
			(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff || user.Role.Role == obj.RoleIndividual) {
			// Enter workshop mode instead of changing roles
			setErr := db.SetActiveWorkshop(r.Context(), user.ID, *invite.WorkshopID)
			if setErr != nil {
				if appErr, ok := setErr.(*obj.AppError); ok {
					httpx.WriteAppError(w, appErr)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, setErr.Error())
				return
			}
			// Mark invite as accepted
			_ = db.UpdateInviteStatus(r.Context(), user.ID, inviteID, obj.InviteStatusAccepted)
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
			httpx.WriteJSON(w, http.StatusOK, AcceptInviteResponse{
				Message: "Workshop entered",
			})
			return
		}

		// Standard targeted invite (institution) - accept by ID
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
