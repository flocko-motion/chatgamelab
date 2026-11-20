// package: api / schemaf endpoints
// type:    logic
// job:     lists, reads, updates and removes users, and the caller's own stats, participant code, language and active workshop
// limits:  user storage and its permission checks live in db; roles (-> user_role.go); the caller's own account (-> auth.go)
package api

import (
	"context"
	"fmt"
	"strings"

	"cgl/constants"
	"cgl/db"
	"cgl/events"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// UsersResp is a list of users.
type UsersResp = []obj.User

// UserStatsResp is a user's statistics.
type UserStatsResp = obj.UserStats

// UserIDReq names a user by their id.
type UserIDReq struct {
	ID string `path:"id" json:"-"`
}

// ListUsersEndpoint lists the users the caller manages.
//
// Admins see every user; heads and staff see the members of their
// institution. Everyone else is refused.
type ListUsersEndpoint struct{}

func (ListUsersEndpoint) Method() string             { return "GET" }
func (ListUsersEndpoint) Path() string               { return "/api/users" }
func (ListUsersEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListUsersEndpoint) Handle(ctx context.Context, _ ListUsersReq) (UsersResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		users, err := db.GetAllUsers(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
		return users, nil
	}
	if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) && user.Role.Institution != nil {
		members, err := db.GetInstitutionMembers(ctx, user.Role.Institution.ID, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
		return members, nil
	}
	return nil, appError(obj.ErrForbidden("only admins or institution heads/staff can list users"))
}

// ListUsersReq takes no parameters.
type ListUsersReq struct{}

// GetCurrentUserStatsEndpoint returns the caller's statistics.
type GetCurrentUserStatsEndpoint struct{}

func (GetCurrentUserStatsEndpoint) Method() string             { return "GET" }
func (GetCurrentUserStatsEndpoint) Path() string               { return "/api/users/me/stats" }
func (GetCurrentUserStatsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetCurrentUserStatsEndpoint) Handle(ctx context.Context, _ GetCurrentUserStatsReq) (UserStatsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.UserStats{}, err
	}
	stats, err := db.GetUserStats(ctx, user.ID)
	if err != nil {
		return obj.UserStats{}, fmt.Errorf("failed to get user stats: %w", err)
	}
	return *stats, nil
}

// GetCurrentUserStatsReq takes no parameters.
type GetCurrentUserStatsReq struct{}

// GetCurrentUserParticipantCodeEndpoint returns the caller's own re-login code.
//
// Both fields are null for an account without one.
type GetCurrentUserParticipantCodeEndpoint struct{}

func (GetCurrentUserParticipantCodeEndpoint) Method() string { return "GET" }
func (GetCurrentUserParticipantCodeEndpoint) Path() string {
	return "/api/users/me/participant-code"
}
func (GetCurrentUserParticipantCodeEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (GetCurrentUserParticipantCodeEndpoint) Handle(ctx context.Context, _ GetCurrentUserParticipantCodeReq) (UserParticipantCodeResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return UserParticipantCodeResp{}, err
	}
	token, err := db.GetOwnParticipantToken(ctx, user.ID)
	if err != nil {
		return UserParticipantCodeResp{}, appError(obj.ErrServerError("failed to get participant code"))
	}
	var resp UserParticipantCodeResp
	if token != "" {
		resp.Token = &token
		if words := db.ParticipantTokenWords(token); words != "" {
			resp.Words = &words
		}
	}
	return resp, nil
}

// GetCurrentUserParticipantCodeReq takes no parameters.
type GetCurrentUserParticipantCodeReq struct{}

// UserParticipantCodeResp is the caller's own re-login code; both fields are
// null for an account without one.
type UserParticipantCodeResp struct {
	// Token is the full login token that the /code link and QR code carry.
	Token *string `json:"token"`
	// Words is the token's word code, or null for the long tokens issued before word tokens.
	Words *string `json:"words"`
}

// UpdateUserLanguageEndpoint sets the caller's language and answers with
// their account.
//
// The language is an ISO 639-1 code the platform supports.
type UpdateUserLanguageEndpoint struct{}

func (UpdateUserLanguageEndpoint) Method() string             { return "PATCH" }
func (UpdateUserLanguageEndpoint) Path() string               { return "/api/users/me/language" }
func (UpdateUserLanguageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateUserLanguageEndpoint) Handle(ctx context.Context, req UpdateUserLanguageReq) (UserResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}
	language := strings.ToLower(strings.TrimSpace(req.Language))
	if !lang.IsValidLanguageCode(language) {
		return obj.User{}, errBadRequest("Unsupported language code")
	}
	if err := db.UpdateUserLanguage(ctx, user.ID, user.ID, language); err != nil {
		log.Error("failed to update user language", "user_id", user.ID, "error", err)
		return obj.User{}, fmt.Errorf("failed to update language: %w", err)
	}
	updated, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to get updated user: %w", err)
	}
	return *updated, nil
}

// UpdateUserLanguageReq is the caller's new language.
type UpdateUserLanguageReq struct {
	// Language is an ISO 639-1 code.
	Language string `json:"language"`
}

// SetActiveWorkshopEndpoint enters or leaves workshop mode and answers with
// the caller's account.
//
// Heads, staff and individuals enter a workshop by its id; a null workshopId
// leaves workshop mode.
type SetActiveWorkshopEndpoint struct{}

func (SetActiveWorkshopEndpoint) Method() string             { return "PUT" }
func (SetActiveWorkshopEndpoint) Path() string               { return "/api/users/me/active-workshop" }
func (SetActiveWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetActiveWorkshopEndpoint) Handle(ctx context.Context, req SetActiveWorkshopReq) (UserResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}

	// The workshop being left, for its members' live update.
	var oldWorkshopID *uuid.UUID
	if user.Role != nil && user.Role.Workshop != nil {
		oldWorkshopID = &user.Role.Workshop.ID
	}

	if req.WorkshopID == nil {
		if oldWorkshopID != nil {
			if err := db.ClearActiveWorkshop(ctx, user.ID, *oldWorkshopID); err != nil {
				return obj.User{}, fmt.Errorf("failed to leave workshop: %w", err)
			}
			events.GetBroker().PublishMembersUpdated(*oldWorkshopID)
		} else {
			if err := db.ClearActiveWorkshop(ctx, user.ID); err != nil {
				return obj.User{}, fmt.Errorf("failed to leave workshop: %w", err)
			}
		}
	} else {
		if err := db.SetActiveWorkshop(ctx, user.ID, *req.WorkshopID); err != nil {
			return obj.User{}, appError(err)
		}
		events.GetBroker().PublishMembersUpdated(*req.WorkshopID)
	}

	updated, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to get updated user: %w", err)
	}
	return *updated, nil
}

// SetActiveWorkshopReq is the workshop to enter, or null to leave.
type SetActiveWorkshopReq struct {
	WorkshopID *uuid.UUID `json:"workshopId"`
}

// GetUserEndpoint returns a user.
//
// The caller reads their own account; admins read anyone; heads and staff
// read the members of their institution.
type GetUserEndpoint struct{}

func (GetUserEndpoint) Method() string             { return "GET" }
func (GetUserEndpoint) Path() string               { return "/api/users/{id}" }
func (GetUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetUserEndpoint) Handle(ctx context.Context, req UserIDReq) (UserResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}
	targetID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return obj.User{}, err
	}
	if err := db.CanReadUser(ctx, user.ID, targetID); err != nil {
		return obj.User{}, errForbidden("Not authorized to read this user")
	}
	target, err := db.GetUserByID(ctx, targetID)
	if err != nil {
		return obj.User{}, errNotFound("User not found")
	}
	return *target, nil
}

// UpdateUserEndpoint updates a user and answers with them.
//
// The caller updates themselves; admins update anyone; heads and staff update
// the participants of their institution's workshops. Only admins change an
// email. An empty name or email leaves it unchanged; an empty aiQualityTier
// clears it. The age group is "u13" (13-17, no parental consent), "u13p"
// (13-17, with parental consent) or "u18" (18+).
type UpdateUserEndpoint struct{}

func (UpdateUserEndpoint) Method() string             { return "POST" }
func (UpdateUserEndpoint) Path() string               { return "/api/users/{id}" }
func (UpdateUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateUserEndpoint) Handle(ctx context.Context, req UpdateUserReq) (UserResp, error) {
	currentUser, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}
	userID, err := parseID(req.UserID, "Invalid user ID")
	if err != nil {
		return obj.User{}, err
	}

	isAdmin := currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin
	canUpdate := false
	if userID == currentUser.ID || isAdmin {
		canUpdate = true
	} else if currentUser.Role != nil && (currentUser.Role.Role == obj.RoleHead || currentUser.Role.Role == obj.RoleStaff) {
		if err := db.CanUpdateParticipantName(ctx, currentUser.ID, userID); err == nil {
			canUpdate = true
		}
	}
	if !canUpdate {
		return obj.User{}, errForbidden("Forbidden: cannot update this user")
	}

	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return obj.User{}, errNotFound("User not found")
	}

	// An empty email in the request leaves the email unchanged.
	emailChanged := req.Email != "" && (user.Email == nil || req.Email != *user.Email)
	nameChanged := req.Name != "" && req.Name != user.Name

	if emailChanged && !isAdmin {
		return obj.User{}, errForbidden("Only administrators can change email addresses")
	}
	if nameChanged && constants.IsProfane(req.Name) {
		return obj.User{}, appError(obj.ErrProfaneName("This name is not allowed"))
	}
	if nameChanged {
		nameTaken, err := db.IsNameTakenByOther(ctx, req.Name, userID)
		if err != nil {
			return obj.User{}, fmt.Errorf("failed to check name availability: %w", err)
		}
		if nameTaken {
			return obj.User{}, errConflict("Name is already taken")
		}
	}
	if emailChanged {
		emailTaken, err := db.IsEmailTakenByOther(ctx, req.Email, userID)
		if err != nil {
			return obj.User{}, fmt.Errorf("failed to check email availability: %w", err)
		}
		if emailTaken {
			return obj.User{}, errConflict("Email is already taken")
		}
	}

	if nameChanged || emailChanged {
		name := user.Name
		if req.Name != "" {
			name = req.Name
		}
		email := user.Email
		if req.Email != "" {
			email = &req.Email
		}
		if err := db.UpdateUserDetails(ctx, userID, name, email); err != nil {
			return obj.User{}, fmt.Errorf("failed to update user: %w", err)
		}
	}

	if req.DefaultApiKeyShareID != nil {
		if err := db.SetUserDefaultApiKeyShare(ctx, userID, req.DefaultApiKeyShareID); err != nil {
			return obj.User{}, errBadRequest("Failed to set default API key: " + err.Error())
		}
	}

	if req.AgeGroup != nil {
		if *req.AgeGroup != obj.AgeGroupU13 && *req.AgeGroup != obj.AgeGroupU13p && *req.AgeGroup != obj.AgeGroupU18 {
			return obj.User{}, errBadRequest("Age group must be '" + obj.AgeGroupU13 + "', '" + obj.AgeGroupU13p + "' or '" + obj.AgeGroupU18 + "'")
		}
		if err := db.UpdateUserAgeGroup(ctx, userID, req.AgeGroup); err != nil {
			return obj.User{}, fmt.Errorf("failed to update age group: %w", err)
		}
	}

	if req.AiQualityTier != nil {
		tier := req.AiQualityTier
		if *tier == "" {
			tier = nil
		}
		if err := db.UpdateUserAiQualityTier(ctx, userID, tier); err != nil {
			return obj.User{}, fmt.Errorf("failed to update AI quality tier: %w", err)
		}
	}

	updated, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to get updated user: %w", err)
	}
	return *updated, nil
}

// UpdateUserReq is the user from the path and the properties to change.
type UpdateUserReq struct {
	UserID               string     `path:"id" json:"-"`
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId,omitempty"`
	AiQualityTier        *string    `json:"aiQualityTier,omitempty"`
	// AgeGroup is "u13" (13-17, no parental consent), "u13p" (13-17, with
	// parental consent) or "u18" (18+).
	AgeGroup *string `json:"ageGroup,omitempty"`
}

// DeleteUserEndpoint removes a user, such as a participant, keeping their
// record.
type DeleteUserEndpoint struct{}

func (DeleteUserEndpoint) Method() string             { return "DELETE" }
func (DeleteUserEndpoint) Path() string               { return "/api/users/{id}" }
func (DeleteUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteUserEndpoint) Handle(ctx context.Context, req UserIDReq) (UserDeletedResp, error) {
	currentUser, err := caller(ctx)
	if err != nil {
		return UserDeletedResp{}, err
	}
	userID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserDeletedResp{}, err
	}
	if err := db.RemoveUser(ctx, currentUser.ID, userID); err != nil {
		return UserDeletedResp{}, appError(err)
	}
	return UserDeletedResp{}, nil
}

// UserDeletedResp confirms a removal; it has no fields.
type UserDeletedResp struct{}

// CreateUserReq is a new user's name and optional email.
type CreateUserReq struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

// CreateUserEndpoint creates a user without Auth0; in dev mode only.
//
// The name is required. The account's language follows the request's
// Accept-Language.
type CreateUserEndpoint struct{}

func (CreateUserEndpoint) Method() string             { return "POST" }
func (CreateUserEndpoint) Path() string               { return "/api/users/new" }
func (CreateUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (CreateUserEndpoint) Handle(ctx context.Context, req CreateUserReq) (UserResp, error) {
	if !devMode {
		return obj.User{}, errNotFound("Not found")
	}
	if strings.TrimSpace(req.Name) == "" {
		return obj.User{}, errBadRequest("Name is required")
	}
	user, err := db.CreateUser(ctx, req.Name, req.Email, "", nil, preferredLanguage(ctx))
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return *user, nil
}
