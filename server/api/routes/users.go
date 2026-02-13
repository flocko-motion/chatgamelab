package routes

import (
	"net/http"
	"strings"

	"cgl/api/auth"
	"cgl/api/httpx"
	"cgl/constants"
	"cgl/db"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for users
type UserUpdateRequest struct {
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId,omitempty"`
	AiQualityTier        *string    `json:"aiQualityTier,omitempty"`
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
	user := httpx.UserFromRequest(r)

	// Admin sees all users
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		users, err := db.GetAllUsers(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to get users: "+err.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, users)
		return
	}

	// Head/staff sees members of their institution
	if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) && user.Role.Institution != nil {
		members, err := db.GetInstitutionMembers(r.Context(), user.Role.Institution.ID, user.ID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to get users: "+err.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, members)
		return
	}

	// Everyone else (individual, participant) gets 403
	httpx.WriteAppError(w, obj.ErrForbidden("only admins or institution heads/staff can list users"))
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

// GetCurrentUserStats godoc
//
//	@Summary		Get current user statistics
//	@Description	Returns statistics for the authenticated user
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	obj.UserStats
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Security		BearerAuth
//	@Router			/users/me/stats [get]
func GetCurrentUserStats(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	ctx := r.Context()

	stats, err := db.GetUserStats(ctx, user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get user stats")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, stats)
}

// UpdateUserLanguage godoc
//
//	@Summary		Update user language preference
//	@Description	Sets the language preference for the authenticated user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{language=string}	true	"Language preference (ISO 639-1 code)"
//	@Success		200		{object}	obj.User
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Security		BearerAuth
//	@Router			/users/me/language [patch]
func UpdateUserLanguage(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	ctx := r.Context()

	var req struct {
		Language string `json:"language"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Validate language code against supported languages
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if !lang.IsValidLanguageCode(req.Language) {
		httpx.WriteError(w, http.StatusBadRequest, "Unsupported language code")
		return
	}

	err := db.UpdateUserLanguage(ctx, user.ID, user.ID, req.Language)
	if err != nil {
		log.Error("failed to update user language", "user_id", user.ID, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update language")
		return
	}

	// Return updated user
	updatedUser, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updatedUser)
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
	currentUser := httpx.UserFromRequest(r)

	targetID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Permission check: own profile, admin, or head/staff for org members
	if err := db.CanReadUser(r.Context(), currentUser.ID, targetID); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "Not authorized to read this user")
		return
	}

	user, err := db.GetUserByID(r.Context(), targetID)
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

	// Permission check: self, admin, or staff/head managing participants in their institution
	canUpdate := false
	if userID == currentUser.ID {
		// Users can always update themselves
		canUpdate = true
	} else if currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin {
		// Admins can update anyone
		canUpdate = true
	} else if currentUser.Role != nil && (currentUser.Role.Role == obj.RoleHead || currentUser.Role.Role == obj.RoleStaff) {
		// Head/Staff can update participants in their institution's workshops
		if err := db.CanUpdateParticipantName(r.Context(), currentUser.ID, userID); err == nil {
			canUpdate = true
		}
	}

	if !canUpdate {
		httpx.WriteError(w, http.StatusForbidden, "Forbidden: cannot update this user")
		return
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
	// Only consider email changed if a non-empty email is provided that differs from current
	// (empty email in request means "don't change email", not "clear email")
	emailChanged := req.Email != "" && (user.Email == nil || req.Email != *user.Email)
	nameChanged := req.Name != "" && req.Name != user.Name

	// Only admins can change email addresses
	isAdmin := currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin
	if emailChanged && !isAdmin {
		httpx.WriteError(w, http.StatusForbidden, "Only administrators can change email addresses")
		return
	}

	// Check for profanity if name changed
	if nameChanged && constants.IsProfane(req.Name) {
		httpx.WriteAppError(w, obj.ErrProfaneName("This name is not allowed"))
		return
	}

	// Validate name uniqueness if changed
	if nameChanged {
		nameTaken, err := db.IsNameTakenByOther(r.Context(), req.Name, userID)
		if err != nil {
			log.Error("failed to check name availability", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to check name availability")
			return
		}
		if nameTaken {
			httpx.WriteError(w, http.StatusConflict, "Name is already taken")
			return
		}
	}

	// Validate email uniqueness if changed
	if emailChanged && req.Email != "" {
		emailTaken, err := db.IsEmailTakenByOther(r.Context(), req.Email, userID)
		if err != nil {
			log.Error("failed to check email availability", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to check email availability")
			return
		}
		if emailTaken {
			httpx.WriteError(w, http.StatusConflict, "Email is already taken")
			return
		}
	}

	if nameChanged || emailChanged {
		name := user.Name
		if req.Name != "" {
			name = req.Name
		}
		var email *string
		if req.Email != "" {
			email = &req.Email
		} else if user.Email != nil {
			email = user.Email
		}
		if err := db.UpdateUserDetails(r.Context(), userID, name, email); err != nil {
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

	// Handle aiQualityTier update
	if req.AiQualityTier != nil {
		tier := req.AiQualityTier
		if *tier == "" {
			tier = nil // empty string means clear
		}
		if err := db.UpdateUserAiQualityTier(r.Context(), userID, tier); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to update AI quality tier")
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

// DeleteUser godoc
//
//	@Summary		Delete user
//	@Description	Soft-deletes a user (for removing participants)
//	@Tags			users
//	@Param			id	path	string	true	"User ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid user ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := httpx.UserFromRequest(r)
	userID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := db.RemoveUser(r.Context(), currentUser.ID, userID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			status := http.StatusForbidden
			switch appErr.Code {
			case obj.ErrCodeNotFound:
				status = http.StatusNotFound
			case obj.ErrCodeForbidden:
				status = http.StatusForbidden
			case obj.ErrCodeUnauthorized:
				status = http.StatusUnauthorized
			case obj.ErrCodeLastHead:
				status = http.StatusConflict
			}
			httpx.WriteError(w, status, appErr.Error())
		} else {
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SetActiveWorkshopRequest is the request body for setting active workshop
type SetActiveWorkshopRequest struct {
	WorkshopID *uuid.UUID `json:"workshopId"`
}

// SetActiveWorkshop godoc
//
//	@Summary		Set active workshop (workshop mode)
//	@Description	Sets the active workshop for head/staff/individual users to enter workshop mode. Pass null workshopId to leave workshop mode.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SetActiveWorkshopRequest	true	"Workshop ID (null to leave)"
//	@Success		200		{object}	obj.User					"Updated user with workshop context"
//	@Failure		400		{object}	httpx.ErrorResponse			"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse			"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse			"Forbidden - not allowed for this role"
//	@Failure		404		{object}	httpx.ErrorResponse			"Workshop not found"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me/active-workshop [put]
func SetActiveWorkshop(w http.ResponseWriter, r *http.Request) {
	currentUser := httpx.UserFromRequest(r)

	var req SetActiveWorkshopRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkshopID == nil {
		// Leave workshop mode
		if err := db.ClearActiveWorkshop(r.Context(), currentUser.ID); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		// Enter workshop mode
		if err := db.SetActiveWorkshop(r.Context(), currentUser.ID, *req.WorkshopID); err != nil {
			if appErr, ok := err.(*obj.AppError); ok {
				status := http.StatusForbidden
				switch appErr.Code {
				case obj.ErrCodeNotFound:
					status = http.StatusNotFound
				case obj.ErrCodeForbidden:
					status = http.StatusForbidden
				}
				httpx.WriteError(w, status, appErr.Error())
			} else {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
	}

	// Return updated user
	user, err := db.GetUserByID(r.Context(), currentUser.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}
