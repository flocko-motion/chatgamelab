package routes

import (
	"io"
	"log"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request types for users
type UserUpdateRequest struct {
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId,omitempty"`
}

type CreateGameResponse struct {
	ID uuid.UUID `json:"id"`
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
	user := httpx.UserFrom(r)
	if user == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

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
	user := httpx.UserFrom(r)
	if user == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

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
	currentUser := httpx.UserFrom(r)
	if currentUser == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

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
	currentUser := httpx.UserFrom(r)
	if currentUser == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

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

// CreateGame godoc
//
//	@Summary		Create game
//	@Description	Creates a new game. Supports JSON body {"name":"..."}. If Content-Type is application/x-yaml, the raw body is interpreted as YAML and applied after creation.
//	@Tags			games
//	@Accept			json
//	@Accept			application/x-yaml
//	@Produce		json
//	@Param			request	body		object	false	"Create game request (JSON: {name}) or YAML string"
//	@Success		200		{object}	CreateGameResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/new [post]
func CreateGame(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFrom(r)
	if user == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	contentType := r.Header.Get("Content-Type")

	// Create a minimal game first
	newGame := obj.Game{
		Name:         "New Game",
		StatusFields: `[{"name":"Gold","value":"100"}]`,
	}

	if err := db.CreateGame(r.Context(), user.ID, &newGame); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create game: "+err.Error())
		return
	}

	if contentType == "application/x-yaml" || contentType == "text/yaml" {
		// Update with YAML content from "import game" form
		body, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Failed to read body: "+err.Error())
			return
		}

		if err := db.UpdateGameYaml(r.Context(), user.ID, newGame.ID, string(body)); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Failed to apply YAML: "+err.Error())
			return
		}
	} else {
		// Update with JSON content from "new game" form
		var req struct {
			Name string `json:"name"`
		}
		if err := httpx.ReadJSON(r, &req); err != nil {
			// Ignore JSON parse error for empty body
		}

		if req.Name != "" {
			newGame.Name = req.Name
			if err := db.UpdateGame(r.Context(), user.ID, &newGame); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "Failed to update game: "+err.Error())
				return
			}
		}
	}

	httpx.WriteJSON(w, http.StatusOK, CreateGameResponse{ID: newGame.ID})
}
