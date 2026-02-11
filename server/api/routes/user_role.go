package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

// SetUserRoleRequest represents the request to set a user's role
type SetUserRoleRequest struct {
	Role          string     `json:"role"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
}

// SetUserRole godoc
//
//	@Summary		Set user role
//	@Description	Sets a user's role in the system
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"User ID"
//	@Param			request	body		SetUserRoleRequest	true	"Role details"
//	@Success		200		{object}	obj.User
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id}/role [post]
func SetUserRole(w http.ResponseWriter, r *http.Request) {
	currentUser := httpx.UserFromRequest(r)

	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid user ID"))
		return
	}

	var req SetUserRoleRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	// Convert role string to pointer
	roleStr := req.Role
	err = db.UpdateUserRole(r.Context(), currentUser.ID, userID, &roleStr, req.InstitutionID, req.WorkshopID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated user
	user, err := db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch updated user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

// RemoveUserRole godoc
//
//	@Summary		Remove user role
//	@Description	Removes a user's role
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	obj.User
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id}/role [delete]
func RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	currentUser := httpx.UserFromRequest(r)

	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid user ID"))
		return
	}

	// Remove role by passing nil
	err = db.UpdateUserRole(r.Context(), currentUser.ID, userID, nil, nil, nil)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated user
	user, err := db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch updated user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}
