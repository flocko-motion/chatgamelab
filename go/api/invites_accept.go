// package: api / schemaf endpoints
// type:    logic
// job:     accepts, declines and revokes invites
// limits:  creating and listing invites (-> invites_create.go, invites.go)
package api

import (
	"context"

	"cgl/db"
	"cgl/events"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// AcceptInviteResp confirms an accepted invite; an anonymous caller also gets
// the Guest's account created for them and its sign-in code.
type AcceptInviteResp struct {
	User      *obj.User `json:"user,omitempty"`
	AuthToken *string   `json:"authToken,omitempty"`
	Message   string    `json:"message"`
}

// AcceptInviteEndpoint accepts an invite by its id or its token.
//
// An anonymous caller accepts a workshop invite by token: this creates a
// Guest in the browser's language, starts its session, sets the
// session cookie and answers with the account and its token. A signed-in
// Guest switches to the invite's workshop; a head, staff member or
// individual enters it and keeps their role; anyone else joins as a
// Guest. An invite by id needs a signed-in caller: a workshop invite
// lets them enter the workshop with their own role, an institution invite
// gives them its role.
type AcceptInviteEndpoint struct{}

func (AcceptInviteEndpoint) Method() string             { return "POST" }
func (AcceptInviteEndpoint) Path() string               { return "/api/invites/{idOrToken}/accept" }
func (AcceptInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (AcceptInviteEndpoint) Handle(ctx context.Context, req AcceptInviteReq) (AcceptInviteResp, error) {
	if err := tokenLocked(); err != nil {
		return AcceptInviteResp{}, err
	}
	user, err := maybeCaller(ctx)
	if err != nil {
		return AcceptInviteResp{}, err
	}
	inviteID, parseErr := uuid.Parse(req.IDOrToken)
	if parseErr != nil {
		token, _ := db.ResolveInviteToken(ctx, req.IDOrToken)
		if user == nil {
			return acceptInviteAnonymously(ctx, token)
		}
		return acceptInviteByToken(ctx, user, token)
	}
	if user == nil {
		return AcceptInviteResp{}, appError(obj.ErrUnauthorized("Authentication required for targeted invites"))
	}
	return acceptInviteByID(ctx, user, inviteID)
}

// AcceptInviteReq names an invite by its id or its token.
type AcceptInviteReq struct {
	IDOrToken string `path:"idOrToken"`
}

// acceptInviteAnonymously creates a Guest for a workshop invite token
// and signs them in. The browser's Accept-Language header is the only language
// signal a link-joining Guest gives.
func acceptInviteAnonymously(ctx context.Context, token string) (AcceptInviteResp, error) {
	createdUser, authToken, err := db.AcceptWorkshopInviteAnonymously(ctx, token, preferredLanguage(ctx))
	if err != nil {
		return AcceptInviteResp{}, appError(err)
	}
	// The Guest's token stays their code to sign in again.
	if _, err := schemafapi.StartSessionCtx(ctx, createdUser.ID.String()); err != nil {
		return AcceptInviteResp{}, err
	}
	if createdUser.Role != nil && createdUser.Role.Workshop != nil {
		events.GetBroker().PublishMembersUpdated(createdUser.Role.Workshop.ID)
	}
	return AcceptInviteResp{
		User:      createdUser,
		AuthToken: &authToken,
		Message:   "Workshop invite accepted, user created",
	}, nil
}

// acceptInviteByToken accepts an open invite for a signed-in caller.
func acceptInviteByToken(ctx context.Context, user *obj.User, token string) (AcceptInviteResp, error) {
	// A Guest switches workshops with a new role assignment.
	if user.Role != nil && user.Role.Role == obj.RoleGuest {
		invite, _ := db.GetInviteByToken(ctx, uuid.Nil, token)
		if _, err := db.AcceptOpenInvite(ctx, token, user.ID); err != nil {
			return AcceptInviteResp{}, appError(err)
		}
		if user.Role.Workshop != nil {
			events.GetBroker().PublishMembersUpdated(user.Role.Workshop.ID)
		}
		if invite.WorkshopID != nil {
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
		}
		return AcceptInviteResp{Message: "Invite accepted"}, nil
	}

	// Head, staff and individual enter the workshop and keep their role.
	if keepsRoleInWorkshop(user) {
		invite, err := db.GetInviteByToken(ctx, uuid.Nil, token)
		if err != nil {
			return AcceptInviteResp{}, appError(err)
		}
		return enterInviteWorkshop(ctx, user, invite)
	}

	// Anyone else joins as a Guest.
	invite, _ := db.GetInviteByToken(ctx, uuid.Nil, token)
	if _, err := db.AcceptOpenInvite(ctx, token, user.ID); err != nil {
		return AcceptInviteResp{}, appError(err)
	}
	if invite.WorkshopID != nil {
		events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
	}
	return AcceptInviteResp{Message: "Invite accepted"}, nil
}

// acceptInviteByID accepts an invite by its id for a signed-in caller.
func acceptInviteByID(ctx context.Context, user *obj.User, inviteID uuid.UUID) (AcceptInviteResp, error) {
	invite, err := db.GetInviteByID(ctx, user.ID, inviteID)
	if err != nil {
		return AcceptInviteResp{}, appError(err)
	}

	if invite.InviteToken != nil {
		// Open invite (workshop): head, staff and individual enter the
		// workshop, anyone else joins as a Guest.
		if keepsRoleInWorkshop(user) {
			return enterInviteWorkshop(ctx, user, invite)
		}
		if _, err := db.AcceptOpenInvite(ctx, *invite.InviteToken, user.ID); err != nil {
			return AcceptInviteResp{}, appError(err)
		}
		if invite.WorkshopID != nil {
			events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
		}
		return AcceptInviteResp{Message: "Invite accepted"}, nil
	}

	// A targeted workshop invite lets the caller enter the workshop with their
	// own role; a targeted institution invite gives the caller its role.
	if _, err := db.AcceptTargetedInvite(ctx, inviteID, "", user.ID); err != nil {
		return AcceptInviteResp{}, appError(err)
	}
	if invite.WorkshopID != nil {
		events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
		return AcceptInviteResp{Message: "Workshop entered"}, nil
	}
	return AcceptInviteResp{Message: "Invite accepted"}, nil
}

// keepsRoleInWorkshop is true for head, staff and individual, who enter a
// workshop with their own role.
func keepsRoleInWorkshop(user *obj.User) bool {
	return user.Role != nil &&
		(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff || user.Role.Role == obj.RoleIndividual)
}

// enterInviteWorkshop makes the invite's workshop the caller's active one.
func enterInviteWorkshop(ctx context.Context, user *obj.User, invite obj.UserRoleInvite) (AcceptInviteResp, error) {
	if invite.WorkshopID == nil {
		return AcceptInviteResp{}, appError(obj.ErrValidation("this is not a workshop invite"))
	}
	if err := db.SetActiveWorkshop(ctx, user.ID, *invite.WorkshopID); err != nil {
		return AcceptInviteResp{}, appError(err)
	}
	events.GetBroker().PublishMembersUpdated(*invite.WorkshopID)
	return AcceptInviteResp{Message: "Workshop entered"}, nil
}

// InviteIDReq names an invite by its id.
type InviteIDReq struct {
	ID string `path:"id"`
}

// DeclineInviteEndpoint declines an invite and answers with its id and its
// status "declined".
type DeclineInviteEndpoint struct{}

func (DeclineInviteEndpoint) Method() string             { return "POST" }
func (DeclineInviteEndpoint) Path() string               { return "/api/invites/{id}/decline" }
func (DeclineInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeclineInviteEndpoint) Handle(ctx context.Context, req InviteIDReq) (UserRoleInviteResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}
	inviteID, err := parseIDValidation(req.ID, "Invalid invite ID")
	if err != nil {
		return obj.UserRoleInvite{}, err
	}
	// Fetching the invite checks the caller may see it.
	if _, err := db.GetInviteByID(ctx, user.ID, inviteID); err != nil {
		return obj.UserRoleInvite{}, appError(err)
	}
	if err := db.DeclineTargetedInvite(ctx, inviteID, "", user.ID); err != nil {
		return obj.UserRoleInvite{}, appError(err)
	}
	return obj.UserRoleInvite{ID: inviteID, Status: obj.InviteStatusDeclined}, nil
}

// RevokeInviteEndpoint revokes a pending invite.
//
// Its creator and admins can revoke it, and institution staff a workshop
// invite.
type RevokeInviteEndpoint struct{}

func (RevokeInviteEndpoint) Method() string             { return "DELETE" }
func (RevokeInviteEndpoint) Path() string               { return "/api/invites/{id}" }
func (RevokeInviteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RevokeInviteEndpoint) Handle(ctx context.Context, req InviteIDReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	inviteID, err := parseIDValidation(req.ID, "Invalid invite ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.RevokeInvite(ctx, inviteID, user.ID); err != nil {
		return MessageResp{}, appError(err)
	}
	return MessageResp{Message: "Invite revoked"}, nil
}
