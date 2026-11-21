// package: api / schemaf endpoints
// type:    logic
// job:     lists, reads, updates and deletes users, their stats and login code; /me is a shortcut to the caller's own user
// limits:  decides nothing: the core does (core.ReadUser and its siblings); roles (-> user_role.go); workshop mode is still v1 here (SetActiveWorkshopEndpoint)
package api

import (
	"context"
	"fmt"
	"strings"

	"cgl/api/wire"
	"cgl/core"
	"cgl/db"
	"cgl/events"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// UserViewResp is a user as the caller sees them.
type UserViewResp = wire.UserView

// UserViewsResp is a list of users.
type UserViewsResp = []wire.UserView

// UserStatsResp is a user's statistics.
type UserStatsResp = wire.UserStatsView

// UserSecretsResp is a user's login code.
type UserSecretsResp = wire.UserSecretsView

// UserIDReq names a user by their id.
type UserIDReq struct {
	ID string `path:"id"`
}

// MeReq names the caller: a shortcut to their own user.
type MeReq struct{}

// me is the caller's own id, for the /me shortcuts; refused for a caller
// nobody signed in.
func me(ctx context.Context) (uuid.UUID, error) {
	_, user, err := core.ReadMe(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if user == nil {
		return uuid.Nil, errForbidden("Not signed in or registration not complete")
	}
	return user.ID(), nil
}

// ListUsersEndpoint lists the users the caller may list: admins every user,
// heads and staff the members of their organisation.
type ListUsersEndpoint struct{}

func (ListUsersEndpoint) Method() string             { return "GET" }
func (ListUsersEndpoint) Path() string               { return "/api/users" }
func (ListUsersEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListUsersEndpoint) Handle(ctx context.Context, _ ListUsersReq) (UserViewsResp, error) {
	return wire.UserViewsOf(core.ListUsers(ctx))
}

// ListUsersReq takes no parameters.
type ListUsersReq struct{}

// GetUserEndpoint returns a user the caller may see: themselves, and for
// heads and staff their organisation's members and the guests in its
// workshops; admins anyone. 404 otherwise.
type GetUserEndpoint struct{}

func (GetUserEndpoint) Method() string             { return "GET" }
func (GetUserEndpoint) Path() string               { return "/api/users/{id}" }
func (GetUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetUserEndpoint) Handle(ctx context.Context, req UserIDReq) (UserViewResp, error) {
	userID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserViewResp{}, err
	}
	return wire.UserViewOf(core.ReadUser(ctx, userID))
}

// UpdateUserEndpoint changes a user and answers with them as stored.
//
// Send what changes; a field left out stays as it is. Each changed section
// must be one the caller may edit (mayEditCard for the name, mayEditAccount
// for language, age group, AI quality tier and default key, mayEditContact
// for the email); otherwise the whole request is refused (403). The age group
// is "u13" (13-17, no parental consent), "u13p" (13-17, with parental
// consent) or "u18" (18+). An empty aiQualityTier or email removes it.
type UpdateUserEndpoint struct{}

func (UpdateUserEndpoint) Method() string             { return "POST" }
func (UpdateUserEndpoint) Path() string               { return "/api/users/{id}" }
func (UpdateUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateUserEndpoint) Handle(ctx context.Context, req UpdateUserReq) (UserViewResp, error) {
	userID, err := parseID(req.UserID, "Invalid user ID")
	if err != nil {
		return UserViewResp{}, err
	}
	return wire.UserViewOf(core.UpdateUser(ctx, userID, req.changes()))
}

// UpdateUserReq is the user from the path and what changes; left out (or
// null): unchanged.
type UpdateUserReq struct {
	UserID string `path:"id"`

	Name                 *string    `json:"name"`
	Email                *string    `json:"email"`
	Language             *string    `json:"language"`
	AgeGroup             *string    `json:"ageGroup"`
	AiQualityTier        *string    `json:"aiQualityTier"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId"`
}

func (r UpdateUserReq) changes() core.UserChanges {
	return core.UserChanges{
		Name:                 r.Name,
		Email:                r.Email,
		Language:             r.Language,
		AgeGroup:             r.AgeGroup,
		AiQualityTier:        r.AiQualityTier,
		DefaultApiKeyShareID: r.DefaultApiKeyShareID,
	}
}

// DeleteUserEndpoint deletes a user with their games, keys, shares and
// roles. It takes mayDelete (403 otherwise); an organisation's last head is
// refused (409).
type DeleteUserEndpoint struct{}

func (DeleteUserEndpoint) Method() string             { return "DELETE" }
func (DeleteUserEndpoint) Path() string               { return "/api/users/{id}" }
func (DeleteUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteUserEndpoint) Handle(ctx context.Context, req UserIDReq) (UserDeletedResp, error) {
	userID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserDeletedResp{}, err
	}
	return UserDeletedResp{}, core.DeleteUser(ctx, userID)
}

// UserDeletedResp confirms a removal; it has no fields.
type UserDeletedResp struct{}

// GetUserStatsEndpoint returns what a user did on the platform, counted. It
// takes seeing their account (403 otherwise).
type GetUserStatsEndpoint struct{}

func (GetUserStatsEndpoint) Method() string             { return "GET" }
func (GetUserStatsEndpoint) Path() string               { return "/api/users/{id}/stats" }
func (GetUserStatsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetUserStatsEndpoint) Handle(ctx context.Context, req UserIDReq) (UserStatsResp, error) {
	userID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserStatsResp{}, err
	}
	return wire.UserStatsViewOf(core.ReadUserStats(ctx, userID))
}

// GetMyStatsEndpoint is the caller's own statistics: GetUserStatsEndpoint
// with their id.
type GetMyStatsEndpoint struct{}

func (GetMyStatsEndpoint) Method() string             { return "GET" }
func (GetMyStatsEndpoint) Path() string               { return "/api/users/me/stats" }
func (GetMyStatsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetMyStatsEndpoint) Handle(ctx context.Context, _ MeReq) (UserStatsResp, error) {
	id, err := me(ctx)
	if err != nil {
		return UserStatsResp{}, err
	}
	return wire.UserStatsViewOf(core.ReadUserStats(ctx, id))
}

// GetUserSecretsEndpoint returns a user's login code: the guest's own, and
// for heads and staff the guests' in their organisation's workshops, to help
// a guest who lost theirs. Both fields are null for an account without one.
type GetUserSecretsEndpoint struct{}

func (GetUserSecretsEndpoint) Method() string             { return "GET" }
func (GetUserSecretsEndpoint) Path() string               { return "/api/users/{id}/secrets" }
func (GetUserSecretsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetUserSecretsEndpoint) Handle(ctx context.Context, req UserIDReq) (UserSecretsResp, error) {
	userID, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserSecretsResp{}, err
	}
	return wire.UserSecretsViewOf(core.ReadUser(ctx, userID))
}

// GetMySecretsEndpoint is the caller's own login code: GetUserSecretsEndpoint
// with their id.
type GetMySecretsEndpoint struct{}

func (GetMySecretsEndpoint) Method() string             { return "GET" }
func (GetMySecretsEndpoint) Path() string               { return "/api/users/me/secrets" }
func (GetMySecretsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetMySecretsEndpoint) Handle(ctx context.Context, _ MeReq) (UserSecretsResp, error) {
	id, err := me(ctx)
	if err != nil {
		return UserSecretsResp{}, err
	}
	return wire.UserSecretsViewOf(core.ReadUser(ctx, id))
}

// UpdateMyLanguageEndpoint sets the caller's language and answers with their
// user: UpdateUserEndpoint with their id and only the language. The
// language is an ISO 639-1 code the platform supports.
type UpdateMyLanguageEndpoint struct{}

func (UpdateMyLanguageEndpoint) Method() string             { return "PATCH" }
func (UpdateMyLanguageEndpoint) Path() string               { return "/api/users/me/language" }
func (UpdateMyLanguageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateMyLanguageEndpoint) Handle(ctx context.Context, req UpdateMyLanguageReq) (UserViewResp, error) {
	id, err := me(ctx)
	if err != nil {
		return UserViewResp{}, err
	}
	language := strings.ToLower(strings.TrimSpace(req.Language))
	return wire.UserViewOf(core.UpdateUser(ctx, id, core.UserChanges{Language: &language}))
}

// UpdateMyLanguageReq is the caller's new language, an ISO 639-1 code.
type UpdateMyLanguageReq struct {
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

// CreateUserReq is a new user's name and optional email.
type CreateUserReq struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

// CreateUserEndpoint creates a user without Auth0; in dev mode only.
//
// The account's language follows the request's Accept-Language. Without a
// name it creates an account still registering, as a first Auth0 sign-in
// does (for testing registration).
type CreateUserEndpoint struct{}

func (CreateUserEndpoint) Method() string             { return "POST" }
func (CreateUserEndpoint) Path() string               { return "/api/users/new" }
func (CreateUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (CreateUserEndpoint) Handle(ctx context.Context, req CreateUserReq) (UserResp, error) {
	if !schemafapi.DevMode() {
		return obj.User{}, errNotFound("Not found")
	}
	if strings.TrimSpace(req.Name) == "" {
		id, err := db.AccountForAuth0(ctx, "dev|"+uuid.NewString())
		if err != nil {
			return obj.User{}, fmt.Errorf("failed to create account: %w", err)
		}
		return obj.User{ID: id}, nil
	}
	user, err := db.CreateUser(ctx, req.Name, req.Email, "", nil, preferredLanguage(ctx))
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return *user, nil
}
