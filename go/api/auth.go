// package: api / schemaf endpoints
// type:    logic
// job:     the platform's ways in beside schemaf's own login routes: completing registration, a participant's sign-in, and the account behind a request
// limits:  who is calling is resolved in api/auth; schemaf serves Auth0 login, /api/user/me, refresh and logout
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"cgl/api/auth"
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"cgl/tokenlock"

	schemafapi "github.com/flocko-motion/schemaf/api"
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
// Takes {"token": "..."}: the participant token whole, or its words from a link,
// QR code or typed code. Starts a session, sets the session cookie and answers
// {"token": "...", "userId": "..."}; the token is for Bearer clients.
type ParticipantLoginEndpoint struct{}

func (ParticipantLoginEndpoint) Method() string             { return "POST" }
func (ParticipantLoginEndpoint) Path() string               { return "/api/auth/participant-login" }
func (ParticipantLoginEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (ParticipantLoginEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	if retry := tokenlock.RetryAfter(); retry > 0 {
		httpx.WriteTokenLocked(w, retry)
		return nil
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		return fmt.Errorf("invalid request body: %w", schemafapi.ErrBadRequest)
	}
	userID, err := auth.RedeemParticipantToken(r.Context(), req.Token)
	if errors.Is(err, auth.ErrInvalidParticipantToken) {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return nil
	}
	if err != nil {
		return err
	}
	token, err := schemafapi.StartSession(r.Context(), w, userID.String())
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token, "userId": userID.String()})
	return nil
}

// appError maps the db layer's AppErrors to schemaf's status errors.
func appError(err error) error {
	var appErr *obj.AppError
	if !errors.As(err, &appErr) {
		return err
	}
	switch appErr.Code {
	case obj.ErrCodeConflict:
		return fmt.Errorf("%s: %w", appErr.Message, schemafapi.ErrConflict)
	case obj.ErrCodeNotFound:
		return fmt.Errorf("%s: %w", appErr.Message, schemafapi.ErrNotFound)
	case obj.ErrCodeForbidden:
		return fmt.Errorf("%s: %w", appErr.Message, schemafapi.ErrForbidden)
	case obj.ErrCodeValidation, obj.ErrCodeInvalidInput:
		return fmt.Errorf("%s: %w", appErr.Message, schemafapi.ErrBadRequest)
	}
	return err
}
