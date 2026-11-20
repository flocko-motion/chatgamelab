// package: api / schemaf endpoints
// type:    logic
// job:     the platform's ways in beside schemaf's own login routes: completing registration, a participant's sign-in, and the account behind a request
// limits:  who is calling is resolved in api/auth; schemaf serves Auth0 login, /api/user/me, refresh and logout
package api

import (
	"context"
	"errors"

	"cgl/api/auth"
	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// CompleteRegistrationEndpoint completes the caller's registration.
//
// An account exists from its first Auth0 sign-in and is unusable until it has a
// name and an age group; the email is optional. An account registers once.
type CompleteRegistrationEndpoint struct{}

func (CompleteRegistrationEndpoint) Method() string             { return "POST" }
func (CompleteRegistrationEndpoint) Path() string               { return "/api/auth/register" }
func (CompleteRegistrationEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CompleteRegistrationEndpoint) Handle(ctx context.Context, req CompleteRegistrationReq) (UserResp, error) {
	caller, err := auth.Registering(ctx)
	if err != nil {
		return obj.User{}, err
	}
	reg, err := req.Validate()
	if err != nil {
		return obj.User{}, err
	}
	user, err := db.CompleteRegistration(ctx, caller.ID, reg.Name, reg.Email, reg.AgeGroup, reg.Language)
	if err != nil {
		return obj.User{}, appError(err)
	}
	return *user, nil
}

// GetCurrentUserEndpoint reports who is calling.
//
// The state is "anonymous" without a sign-in, "registering" for an account
// whose registration is incomplete, and "registered" with the account and its
// role. A participant whose workshop is inactive, or an account that is gone,
// is refused with 403. Every other endpoint refuses an account that is still
// registering; this one reports it.
type GetCurrentUserEndpoint struct{}

func (GetCurrentUserEndpoint) Method() string             { return "GET" }
func (GetCurrentUserEndpoint) Path() string               { return "/api/users/me" }
func (GetCurrentUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (GetCurrentUserEndpoint) Handle(ctx context.Context, _ GetCurrentUserReq) (GetCurrentUserResp, error) {
	if _, ok := schemafapi.Subject(ctx); !ok {
		return GetCurrentUserResp{State: CallerAnonymous}, nil
	}
	user, err := auth.Registering(ctx)
	if err != nil {
		return GetCurrentUserResp{}, err
	}
	if user.Name == "" {
		return GetCurrentUserResp{State: CallerRegistering}, nil
	}
	return GetCurrentUserResp{State: CallerRegistered, User: user}, nil
}

// The states GetCurrentUserEndpoint reports.
const (
	CallerAnonymous   = "anonymous"
	CallerRegistering = "registering"
	CallerRegistered  = "registered"
)

// GetCurrentUserResp is the caller's state, and their account once registered.
type GetCurrentUserResp struct {
	State string    `json:"state"`
	User  *UserResp `json:"user,omitempty"`
}

// CompleteRegistrationReq is the name, age group, optional email and
// interface language that complete an account.
type CompleteRegistrationReq = auth.Registration

// UserResp is an account with its role.
type UserResp = obj.User

// GetCurrentUserReq takes no parameters.
type GetCurrentUserReq struct{}

// ParticipantLoginEndpoint signs a workshop participant in with their token.
//
// Takes the participant token whole, or its words from a link, QR code or
// typed code. Starts a session, sets the session cookie and answers with the
// session token, for Bearer clients, and the participant's id.
type ParticipantLoginEndpoint struct{}

func (ParticipantLoginEndpoint) Method() string             { return "POST" }
func (ParticipantLoginEndpoint) Path() string               { return "/api/auth/participant-login" }
func (ParticipantLoginEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (ParticipantLoginEndpoint) Handle(ctx context.Context, req ParticipantLoginReq) (ParticipantLoginResp, error) {
	if err := tokenLocked(); err != nil {
		return ParticipantLoginResp{}, err
	}
	userID, err := auth.RedeemParticipantToken(ctx, req.Token)
	if errors.Is(err, auth.ErrInvalidParticipantToken) {
		return ParticipantLoginResp{}, errUnauthorized(err.Error())
	}
	if err != nil {
		return ParticipantLoginResp{}, err
	}
	token, err := schemafapi.StartSessionCtx(ctx, userID.String())
	if err != nil {
		return ParticipantLoginResp{}, err
	}
	return ParticipantLoginResp{Token: token, UserID: userID}, nil
}

// ParticipantLoginReq is a participant's token.
type ParticipantLoginReq struct {
	Token string `json:"token"`
}

// ParticipantLoginResp is the session token and the signed-in participant.
type ParticipantLoginResp struct {
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"userId"`
}

// DevLoginEndpoint signs in as one of the seeded development accounts.
//
// Development only: without DEV_MODE it answers 404. Takes a preset role
// ("admin-1", "admin-2", "head-1", "head-2", "staff-1", "staff-2",
// "individual-1", "individual-2", "participant"), starts a session for its
// account and sets the session cookie.
type DevLoginEndpoint struct{}

func (DevLoginEndpoint) Method() string             { return "POST" }
func (DevLoginEndpoint) Path() string               { return "/api/auth/dev-login" }
func (DevLoginEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (DevLoginEndpoint) Handle(ctx context.Context, req DevLoginReq) (DevLoginResp, error) {
	if !devMode {
		return DevLoginResp{}, errNotFound("Not found")
	}
	userID, ok := devLoginAccounts[req.Role]
	if !ok {
		return DevLoginResp{}, errNotFound("Unknown dev role")
	}
	token, err := schemafapi.StartSessionCtx(ctx, userID.String())
	if err != nil {
		return DevLoginResp{}, err
	}
	return DevLoginResp{Token: token, UserID: userID}, nil
}

// devLoginAccounts are the seeded development accounts by preset role.
var devLoginAccounts = map[string]uuid.UUID{
	"admin-1":      db.DevAdmin1UserID,
	"admin-2":      db.DevAdmin2UserID,
	"head-1":       db.DevHead1UserID,
	"head-2":       db.DevHead2UserID,
	"staff-1":      db.DevStaff1UserID,
	"staff-2":      db.DevStaff2UserID,
	"individual-1": db.DevIndividual1UserID,
	"individual-2": db.DevIndividual2UserID,
	"participant":  db.DevParticipantUserID,
}

// DevLoginReq is a preset role of a seeded development account.
type DevLoginReq struct {
	Role string `json:"role"`
}

// DevLoginResp is the session token and the signed-in account.
type DevLoginResp struct {
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"userId"`
}
