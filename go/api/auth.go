// package: api / schemaf endpoints
// type:    logic
// job:     the platform's ways in beside schemaf's own login routes: completing registration (an organisation's included), a participant's sign-in, the account behind a request, and development's logins and sent mails
// limits:  who is calling is resolved by the secure core (core.Resolve); schemaf serves Auth0 login, /api/user/me, refresh and logout
package api

import (
	"context"
	"errors"

	"cgl/api/auth"
	"cgl/api/wire"
	"cgl/core"
	"cgl/db"
	"cgl/mail"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// CompleteRegistrationEndpoint completes the caller's registration and
// answers with their user.
//
// An account exists from its first Auth0 sign-in and is unusable until it has
// a name (at most 24 characters) and an age group; the email is optional. An
// account registers once (409 afterwards). An email another account has is
// refused (409 email_belongs_to_other_account). The rights in the answer are
// those of the registering caller; GET /api/users/me shows the registered
// account's.
//
// With an organisation, the account registers it too and becomes its head
// instead of an individual. That takes the age group u18 (403
// institution_age_not_allowed otherwise), an email and acceptTerms; the
// organisation is as in POST /api/users/me/organisation. Account and
// organisation are stored together or not at all.
type CompleteRegistrationEndpoint struct{}

func (CompleteRegistrationEndpoint) Method() string             { return "POST" }
func (CompleteRegistrationEndpoint) Path() string               { return "/api/auth/register" }
func (CompleteRegistrationEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CompleteRegistrationEndpoint) Handle(ctx context.Context, req CompleteRegistrationReq) (UserViewResp, error) {
	return wire.UserViewOf(core.Register(ctx, req.registration()))
}

// CompleteRegistrationReq is the name, age group, optional email and
// interface language that complete an account.
type CompleteRegistrationReq struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	// AgeGroup is "u13" (13–17 without parental consent), "u13p" (13–17 with
	// parental consent) or "u18" (18 and over).
	AgeGroup string `json:"ageGroup"`
	// Language is the ISO 639-1 code of the interface the person registered
	// in; the account keeps its language when it is empty.
	Language string `json:"language,omitempty"`

	// Organisation, when set, is registered with the account.
	Organisation *OrganisationSignUpReq `json:"organisation,omitempty"`
	// AcceptTerms consents to the terms of use and the privacy policy;
	// required with an organisation.
	AcceptTerms bool `json:"acceptTerms,omitempty"`
}

func (r CompleteRegistrationReq) registration() core.Registration {
	reg := core.Registration{
		Name:        r.Name,
		Email:       r.Email,
		AgeGroup:    r.AgeGroup,
		Language:    r.Language,
		AcceptTerms: r.AcceptTerms,
	}
	if r.Organisation != nil {
		s := r.Organisation.signUp()
		reg.Organisation = &s
	}
	return reg
}

// GetCurrentUserEndpoint reports who is calling.
//
// The state is "anonymous" without a sign-in, "registering" for an account
// whose registration is incomplete, and "registered" with the account and its
// role. A guest whose workshop is inactive, or an account that is gone, is
// refused with 403. Every other endpoint refuses an account that is still
// registering, and a member of a suspended organisation; this one reports
// both.
type GetCurrentUserEndpoint struct{}

func (GetCurrentUserEndpoint) Method() string             { return "GET" }
func (GetCurrentUserEndpoint) Path() string               { return "/api/users/me" }
func (GetCurrentUserEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthOptional }
func (GetCurrentUserEndpoint) Handle(ctx context.Context, _ GetCurrentUserReq) (GetCurrentUserResp, error) {
	state, user, err := core.ReadMe(ctx)
	if err != nil || user == nil {
		return GetCurrentUserResp{State: state}, err
	}
	view, err := wire.UserViewOf(user, nil)
	if err != nil {
		return GetCurrentUserResp{}, err
	}
	my := core.ReadMyOrganisation(ctx)
	return GetCurrentUserResp{
		State:                    state,
		User:                     &view,
		InstitutionSuspension:    my.Suspension,
		MayConsentForInstitution: my.MayConsent,
		MaySuspendInstitutions:   my.MaySuspend,
	}, nil
}

// GetCurrentUserResp is the caller's state (anonymous, registering,
// registered), and their user once registered: the same view as
// GET /api/users/{id} with their id.
type GetCurrentUserResp struct {
	State string        `json:"state"`
	User  *UserViewResp `json:"user,omitempty"`

	// InstitutionSuspension says why the caller's organisation is suspended:
	// "admin" (an admin suspended it) or "consent" (it has no consent to the
	// current declaration; a head lifts it by consenting); empty while it is
	// not. Every other endpoint refuses a suspended organisation's members,
	// but a head may consent (/api/institutions/{id}/declaration-consent).
	InstitutionSuspension string `json:"institutionSuspension,omitempty"`
	// MayConsentForInstitution: the caller may consent to the declaration
	// in their organisation's name or withdraw that, editing its details.
	MayConsentForInstitution bool `json:"mayConsentForInstitution"`
	// MaySuspendInstitutions: the caller may suspend organisations, and sees
	// those that registered themselves lately
	// (GET /api/institutions?filter=recentlyRegistered).
	MaySuspendInstitutions bool `json:"maySuspendInstitutions"`
}

// UserResp is an account with its role.
type UserResp = obj.User

// GetCurrentUserReq takes no parameters.
type GetCurrentUserReq struct{}

// GuestLoginEndpoint signs a workshop participant in with their token.
//
// Takes the participant token whole, or its words from a link, QR code or
// typed code. Starts a session, sets the session cookie and answers with the
// session token, for Bearer clients, and the participant's id.
type GuestLoginEndpoint struct{}

func (GuestLoginEndpoint) Method() string             { return "POST" }
func (GuestLoginEndpoint) Path() string               { return "/api/auth/guest-login" }
func (GuestLoginEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GuestLoginEndpoint) Handle(ctx context.Context, req GuestLoginReq) (GuestLoginResp, error) {
	if err := tokenLocked(); err != nil {
		return GuestLoginResp{}, err
	}
	userID, err := auth.RedeemGuestToken(ctx, req.Token)
	if errors.Is(err, auth.ErrInvalidGuestToken) {
		return GuestLoginResp{}, errUnauthorized(err.Error())
	}
	if err != nil {
		return GuestLoginResp{}, err
	}
	token, err := schemafapi.StartSessionCtx(ctx, userID.String())
	if err != nil {
		return GuestLoginResp{}, err
	}
	return GuestLoginResp{Token: token, UserID: userID}, nil
}

// GuestLoginReq is a participant's token.
type GuestLoginReq struct {
	Token string `json:"token"`
}

// GuestLoginResp is the session token and the signed-in participant.
type GuestLoginResp struct {
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"userId"`
}

// DevLoginEndpoint signs in as one of the seeded development accounts.
//
// Development only: outside development mode it answers 404. Takes a preset role
// ("admin-1", "admin-2", "head-1", "head-2", "staff-1", "staff-2",
// "individual-1", "individual-2", "guest"), starts a session for its
// account and sets the session cookie. "new-person" signs in a fresh
// account still registering, as a first Auth0 sign-in does, to try
// registration without Auth0.
type DevLoginEndpoint struct{}

func (DevLoginEndpoint) Method() string             { return "POST" }
func (DevLoginEndpoint) Path() string               { return "/api/auth/dev-login" }
func (DevLoginEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (DevLoginEndpoint) Handle(ctx context.Context, req DevLoginReq) (DevLoginResp, error) {
	if !schemafapi.DevMode() {
		return DevLoginResp{}, errNotFound("Not found")
	}
	userID, err := devLoginAccount(ctx, req.Role)
	if err != nil {
		return DevLoginResp{}, err
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
	"guest":        db.DevGuestUserID,
}

// devNewPerson is the dev login's preset for a fresh account still
// registering.
const devNewPerson = "new-person"

// devLoginAccount is the account a dev login's preset signs in.
func devLoginAccount(ctx context.Context, role string) (uuid.UUID, error) {
	if role == devNewPerson {
		return db.AccountForAuth0(ctx, "dev|"+uuid.NewString())
	}
	userID, ok := devLoginAccounts[role]
	if !ok {
		return uuid.Nil, errNotFound("Unknown dev role")
	}
	return userID, nil
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

// SentMailsResp is the mails the server sent, newest first.
type SentMailsResp = []mail.Message

// GetSentMailsEndpoint lists the mails the server sent since it started,
// newest first, at most 50.
//
// Development only: outside development mode it answers 404. For tests and
// for trying mails without a mail server.
type GetSentMailsEndpoint struct{}

func (GetSentMailsEndpoint) Method() string             { return "GET" }
func (GetSentMailsEndpoint) Path() string               { return "/api/dev/mails" }
func (GetSentMailsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetSentMailsEndpoint) Handle(_ context.Context, _ GetSentMailsReq) (SentMailsResp, error) {
	if !schemafapi.DevMode() {
		return nil, errNotFound("Not found")
	}
	return mail.Sent(), nil
}

// GetSentMailsReq takes no parameters.
type GetSentMailsReq struct{}
