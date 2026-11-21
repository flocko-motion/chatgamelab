// package: api / schemaf endpoints
// type:    logic
// job:     assigns and removes a user's role, scoped to an institution or workshop
// limits:  who may assign which role is checked in db; listing roles lives elsewhere
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SetUserRoleEndpoint sets a user's role and answers with the user.
//
// The role is scoped to an institution or a workshop by their ids.
type SetUserRoleEndpoint struct{}

func (SetUserRoleEndpoint) Method() string             { return "POST" }
func (SetUserRoleEndpoint) Path() string               { return "/api/users/{id}/role" }
func (SetUserRoleEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetUserRoleEndpoint) Handle(ctx context.Context, req SetUserRoleReq) (UserResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}
	userID, err := parseIDValidation(req.UserID, "Invalid user ID")
	if err != nil {
		return obj.User{}, err
	}
	role := req.Role
	if err := db.UpdateUserRole(ctx, user.ID, userID, &role, req.InstitutionID, req.WorkshopID); err != nil {
		return obj.User{}, appError(err)
	}
	updated, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to fetch updated user: %w", err)
	}
	return *updated, nil
}

// SetUserRoleReq is the user from the path and their new role and its scope.
type SetUserRoleReq struct {
	UserID        string     `path:"id"`
	Role          string     `json:"role"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
}

// RemoveUserRoleEndpoint removes a user's role and answers with the user.
type RemoveUserRoleEndpoint struct{}

func (RemoveUserRoleEndpoint) Method() string             { return "DELETE" }
func (RemoveUserRoleEndpoint) Path() string               { return "/api/users/{id}/role" }
func (RemoveUserRoleEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveUserRoleEndpoint) Handle(ctx context.Context, req UserIDReq) (UserResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.User{}, err
	}
	userID, err := parseIDValidation(req.ID, "Invalid user ID")
	if err != nil {
		return obj.User{}, err
	}
	if err := db.UpdateUserRole(ctx, user.ID, userID, nil, nil, nil); err != nil {
		return obj.User{}, appError(err)
	}
	updated, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return obj.User{}, fmt.Errorf("failed to fetch updated user: %w", err)
	}
	return *updated, nil
}
