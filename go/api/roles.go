// package: api / schemaf endpoints
// type:    logic
// job:     lists the roles that can be assigned to users
// limits:  assigning roles lives in user_role.go
package api

import (
	"context"

	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// ListRolesEndpoint lists the roles a user can have.
type ListRolesEndpoint struct{}

func (ListRolesEndpoint) Method() string             { return "GET" }
func (ListRolesEndpoint) Path() string               { return "/api/roles" }
func (ListRolesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListRolesEndpoint) Handle(ctx context.Context, _ ListRolesReq) (RolesResp, error) {
	if err := core.RequireSignedIn(ctx); err != nil {
		return RolesResp{}, err
	}
	return RolesResp{Roles: []obj.Role{obj.RoleAdmin, obj.RoleHead, obj.RoleStaff}}, nil
}

// ListRolesReq takes no parameters.
type ListRolesReq struct{}

// RolesResp is the roles a user can have.
type RolesResp struct {
	Roles []obj.Role `json:"roles"`
}
