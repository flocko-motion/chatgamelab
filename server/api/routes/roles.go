package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/obj"
)

// RolesResponse represents the response for the roles endpoint
type RolesResponse struct {
	Roles []obj.Role `json:"roles"`
}

// GetRoles godoc
//
//	@Summary		Get available roles
//	@Description	Returns all available user roles
//	@Tags			roles
//	@Produce		json
//	@Success		200	{object}	RolesResponse
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/roles [get]
func GetRoles(w http.ResponseWriter, r *http.Request) {
	// Return all available roles from the obj package
	roles := []obj.Role{
		obj.RoleAdmin,
		obj.RoleHead,
		obj.RoleStaff,
	}

	response := RolesResponse{
		Roles: roles,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
