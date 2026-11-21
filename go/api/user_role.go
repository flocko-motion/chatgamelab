// package: api / schemaf endpoints
// type:    logic
// job:     changes and takes away a user's role
// limits:  decides nothing: the core does (core/role.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SetUserRoleEndpoint gives a user a new role and answers with them.
//
// The admin gives any role, scoped to an organisation or a workshop by
// their ids; a head makes a member of their organisation head or staff
// there. Only an individual becomes admin, and an organisation keeps its
// last head (409 last_head).
type SetUserRoleEndpoint struct{}

func (SetUserRoleEndpoint) Method() string             { return "POST" }
func (SetUserRoleEndpoint) Path() string               { return "/api/users/{id}/role" }
func (SetUserRoleEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetUserRoleEndpoint) Handle(ctx context.Context, req SetUserRoleReq) (UserViewResp, error) {
	id, err := parseID(req.UserID, "Invalid user ID")
	if err != nil {
		return UserViewResp{}, err
	}
	if _, err := core.SetRole(ctx, id, core.RoleChange{
		Role: req.Role, Organisation: req.InstitutionID, Workshop: req.WorkshopID,
	}); err != nil {
		return UserViewResp{}, err
	}
	return wire.UserViewOf(core.ReadUser(ctx, id))
}

// SetUserRoleReq is the user from the path and their new role and its scope.
type SetUserRoleReq struct {
	UserID        string     `path:"id"`
	Role          string     `json:"role"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
}

// RemoveUserRoleEndpoint takes a user's role away, the admin's: they become
// an individual of no organisation. A head removes a member from the
// organisation instead (DELETE /api/institutions/{id}/members/{userID}).
type RemoveUserRoleEndpoint struct{}

func (RemoveUserRoleEndpoint) Method() string             { return "DELETE" }
func (RemoveUserRoleEndpoint) Path() string               { return "/api/users/{id}/role" }
func (RemoveUserRoleEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveUserRoleEndpoint) Handle(ctx context.Context, req UserIDReq) (UserViewResp, error) {
	id, err := parseID(req.ID, "Invalid user ID")
	if err != nil {
		return UserViewResp{}, err
	}
	if _, err := core.RemoveRole(ctx, id); err != nil {
		return UserViewResp{}, err
	}
	return wire.UserViewOf(core.ReadUser(ctx, id))
}
