package routes

import (
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
)

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// RegisterUser godoc
//
//	@Summary		Register new user
//	@Description	Registers a new user after Auth0 authentication. Requires valid Auth0 token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration details"
//	@Success		200		{object}	obj.User
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		409		{object}	httpx.ErrorResponse	"Name already taken"
//	@Security		BearerAuth
//	@Router			/auth/register [post]
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	// Get Auth0 ID from context (set by RequireAuth0Token middleware)
	auth0ID := httpx.Auth0IDFromContext(r.Context())
	if auth0ID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "Auth0 ID not found in token")
		return
	}

	var req RegisterRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if utf8.RuneCountInString(name) > 24 {
		httpx.WriteError(w, http.StatusBadRequest, "Name must be at most 24 characters")
		return
	}

	// Validate email
	email := strings.TrimSpace(req.Email)
	if email == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Email is required")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid email address")
		return
	}

	// Check if user already exists with this Auth0 ID
	existingUser, _ := db.GetUserByAuth0ID(r.Context(), auth0ID)
	if existingUser != nil {
		httpx.WriteError(w, http.StatusBadRequest, "User already registered")
		return
	}

	// Check if name is already taken
	nameTaken, err := db.IsNameTaken(r.Context(), name)
	if err != nil {
		log.Error("failed to check name availability", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to check name availability")
		return
	}
	if nameTaken {
		httpx.WriteError(w, http.StatusConflict, "Name is already taken")
		return
	}

	// Create the user
	user, err := db.CreateUser(r.Context(), name, &email, auth0ID)
	if err != nil {
		log.Error("failed to create user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	log.Info("user registered", "user_id", user.ID, "name", name, "auth0_id", auth0ID)
	httpx.WriteJSON(w, http.StatusOK, user)
}

// CheckNameAvailability godoc
//
//	@Summary		Check if name is available
//	@Description	Checks if a username is available for registration
//	@Tags			auth
//	@Produce		json
//	@Param			name	query		string	true	"Name to check"
//	@Success		200		{object}	map[string]bool
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Router			/auth/check-name [get]
func CheckNameAvailability(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Name parameter is required")
		return
	}

	if utf8.RuneCountInString(name) > 24 {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": false})
		return
	}

	taken, err := db.IsNameTaken(r.Context(), name)
	if err != nil {
		log.Error("failed to check name availability", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to check name availability")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}
