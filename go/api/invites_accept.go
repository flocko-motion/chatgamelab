// package: api / schemaf endpoints
// type:    logic
// job:     accepts, declines and revokes invites
// limits:  decides nothing: the core does (core/invite.go)
package api

import (
	"context"

	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// AcceptInviteResp is what accepting did. For someone who joined without an
// account, user is who they are now and authToken their login code.
type AcceptInviteResp struct {
	User      *NewGuestView `json:"user,omitempty"`
	AuthToken *string       `json:"authToken,omitempty"`
	Message   string        `json:"message"`
}

// NewGuestView is a guest account just made by following a link.
type NewGuestView struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Language string    `json:"language"`
}

// AcceptInviteEndpoint accepts an invite, by its id or its link's token.
//
// Following a link without an account joins its workshop as a new guest,
// signed in at once. A member of an organisation keeps their role and works
// in the workshop; anyone else takes the invite's role. Only a pending
// invite is accepted (409). Repeated guesses at tokens are slowed down
// (429).
type AcceptInviteEndpoint struct{}

func (AcceptInviteEndpoint) Method() string             { return "POST" }
func (AcceptInviteEndpoint) Path() string               { return "/api/invites/{idOrToken}/accept" }
func (AcceptInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (AcceptInviteEndpoint) Handle(ctx context.Context, req AcceptInviteReq) (AcceptInviteResp, error) {
	if err := tokenLocked(); err != nil {
		return AcceptInviteResp{}, err
	}
	accepted, err := core.AcceptInvite(ctx, req.IDOrToken, preferredLanguage(ctx))
	if err != nil {
		return AcceptInviteResp{}, err
	}
	resp := AcceptInviteResp{AuthToken: accepted.GuestCode, Message: accepted.Message}
	if g := accepted.Guest; g != nil {
		resp.User = &NewGuestView{ID: g.ID, Name: g.Name, Language: g.Language}
	}
	return resp, nil
}

// AcceptInviteReq names an invite by its id or its link's token.
type AcceptInviteReq struct {
	IDOrToken string `path:"idOrToken"`
}

// InviteIDReq names an invite by its id.
type InviteIDReq struct {
	ID string `path:"id"`
}

// DeclineInviteEndpoint declines a pending invite addressed to the caller.
type DeclineInviteEndpoint struct{}

func (DeclineInviteEndpoint) Method() string             { return "POST" }
func (DeclineInviteEndpoint) Path() string               { return "/api/invites/{id}/decline" }
func (DeclineInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeclineInviteEndpoint) Handle(ctx context.Context, req InviteIDReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid invite ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.DeclineInvite(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Invite declined"}, nil
}

// RevokeInviteEndpoint deletes a pending invite: for its organisation's head
// and staff (staff not the invites to become head), and the admin. An
// answered invite stays (409).
type RevokeInviteEndpoint struct{}

func (RevokeInviteEndpoint) Method() string             { return "DELETE" }
func (RevokeInviteEndpoint) Path() string               { return "/api/invites/{id}" }
func (RevokeInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RevokeInviteEndpoint) Handle(ctx context.Context, req InviteIDReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid invite ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.RevokeInvite(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Invite revoked"}, nil
}
