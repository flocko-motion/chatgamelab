package routes

import (
	"log"
	"net/http"
	"strings"

	"cgl/api/auth"
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for users
type UserUpdateRequest struct {
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId,omitempty"`
}

type UsersNewRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

type UsersJwtResponse struct {
	UserID  string `json:"userId"`
	Auth0ID string `json:"auth0Id"`
	Token   string `json:"token"`
}

// GetUsers godoc
//
//	@Summary		List users
//	@Description	Returns all users
//	@Tags			users
//	@Produce		json
//	@Success		200	{array}		obj.User
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users [get]
func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.GetAllUsers(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get users: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, users)
}

// GetCurrentUser godoc
//
//	@Summary		Get current user
//	@Description	Returns the currently authenticated user
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	obj.User
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Security		BearerAuth
//	@Router			/users/me [get]
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	httpx.WriteJSON(w, http.StatusOK, user)
}

// GetUserByID godoc
//
//	@Summary		Get user by ID
//	@Description	Returns a user by ID
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string	true	"User ID (UUID)"
//	@Success		200	{object}	obj.User
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid user ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	httpx.ErrorResponse	"User not found"
//	@Security		BearerAuth
//	@Router			/users/{id} [get]
func GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	log.Printf("GetUserByID: %s", userID)

	user, err := db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

// UpdateUserByID godoc
//
//	@Summary		Update user
//	@Description	Updates a user by ID. Non-admins may only update themselves.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"User ID (UUID)"
//	@Param			request	body		UserUpdateRequest	true	"User update"
//	@Success		200		{object}	obj.User
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	httpx.ErrorResponse	"User not found"
//	@Security		BearerAuth
//	@Router			/users/{id} [post]
func UpdateUserByID(w http.ResponseWriter, r *http.Request) {
	currentUser := httpx.UserFromRequest(r)

	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	log.Printf("UpdateUserByID: %s", userID)

	// Only admins may access other users
	if userID != currentUser.ID {
		if currentUser.Role == nil || currentUser.Role.Role != obj.RoleAdmin {
			httpx.WriteError(w, http.StatusForbidden, "Forbidden: admin access required")
			return
		}
	}

	var req UserUpdateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	user, err := db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if name or email changed
	emailChanged := (user.Email == nil && req.Email != "") ||
		(user.Email != nil && req.Email != *user.Email)
	nameChanged := req.Name != user.Name

	if nameChanged || emailChanged {
		var email *string
		if req.Email != "" {
			email = &req.Email
		}
		if err := db.UpdateUserDetails(r.Context(), userID, req.Name, email); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}
	}

	// Handle default API key share update
	if req.DefaultApiKeyShareID != nil {
		if err := db.SetUserDefaultApiKeyShare(r.Context(), userID, req.DefaultApiKeyShareID); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Failed to set default API key: "+err.Error())
			return
		}
	}

	// Refresh user data
	user, err = db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

// CreateUser godoc
//
//	@Summary		Create user (dev only)
//	@Description	Creates a new user without Auth0. Only available in dev mode.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		UsersNewRequest	true	"Create user request"
//	@Success		200		{object}	obj.User
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden (not dev mode)"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/users/new [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req UsersNewRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Name is required")
		return
	}

	user, err := db.CreateUser(r.Context(), req.Name, req.Email, "")
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

// GetUserJWT godoc
//
//	@Summary		Generate JWT (dev only)
//	@Description	Generates a JWT token for a user. Only available in dev mode.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string	true	"User ID (UUID)"
//	@Success		200	{object}	UsersJwtResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid user ID"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden (not dev mode)"
//	@Failure		404	{object}	httpx.ErrorResponse	"User not found"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/users/{id}/jwt [get]
func GetUserJWT(w http.ResponseWriter, r *http.Request) {
	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := db.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	var auth0ID string
	if user.Auth0Id != nil {
		auth0ID = *user.Auth0Id
	}

	tokenString, _, err := auth.GenerateToken(userID.String())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to sign token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UsersJwtResponse{
		UserID:  userID.String(),
		Auth0ID: auth0ID,
		Token:   tokenString,
	})
}
