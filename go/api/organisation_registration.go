// package: api / schemaf endpoints
// type:    logic
// job:     an organisation's consent to the declaration, which its heads give and withdraw, and admins suspending organisations
// limits:  decides nothing: the core does (core.ConsentToDeclaration and its siblings); registering an organisation is POST /api/auth/register (-> auth.go); the declaration's text is GET /api/legal/institution-declaration (-> legal.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// OrganisationSignUpReq is what registering an organisation takes beyond an
// account: who signs for it, the organisation, its youth-protection
// restriction (at most 200 characters) and the consent to the declaration
// in the version shown (GET /api/legal/institution-declaration). A taken
// organisation name is refused (409 institution_name_taken); the
// declaration must be accepted in its current version (400
// institution_consents_required, declaration_version_outdated).
type OrganisationSignUpReq struct {
	PersonName         string `json:"personName"`
	PersonFunction     string `json:"personFunction"`
	InstitutionName    string `json:"institutionName"`
	Operator           string `json:"operator"`
	Street             string `json:"street"`
	PostalCode         string `json:"postalCode"`
	City               string `json:"city"`
	PromptConstraint   string `json:"promptConstraint"`
	AcceptDeclaration  bool   `json:"acceptDeclaration"`
	DeclarationVersion string `json:"declarationVersion"`
}

func (r OrganisationSignUpReq) signUp() core.OrganisationSignUp {
	return core.OrganisationSignUp{
		PersonName:         r.PersonName,
		PersonFunction:     r.PersonFunction,
		Name:               r.InstitutionName,
		Operator:           r.Operator,
		Street:             r.Street,
		PostalCode:         r.PostalCode,
		City:               r.City,
		PromptConstraint:   r.PromptConstraint,
		AcceptDeclaration:  r.AcceptDeclaration,
		DeclarationVersion: r.DeclarationVersion,
	}
}

// OrganisationConsentResp is an organisation as its heads consent for it:
// its consent status, and the details and restriction a consent confirms.
type OrganisationConsentResp = wire.OrganisationConsentView

// GetOrganisationConsentEndpoint returns an organisation as its heads
// consent for it.
//
// For a head of that organisation (403 otherwise), also while it is
// suspended: its consent status (pending while it has no consent to the
// current declaration, which suspends it) and the operator, address and
// youth-protection restriction a consent confirms.
type GetOrganisationConsentEndpoint struct{}

func (GetOrganisationConsentEndpoint) Method() string { return "GET" }
func (GetOrganisationConsentEndpoint) Path() string {
	return "/api/institutions/{id}/declaration-consent"
}
func (GetOrganisationConsentEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetOrganisationConsentEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (OrganisationConsentResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return OrganisationConsentResp{}, err
	}
	return wire.OrganisationConsentViewOf(core.ReadOrganisationConsent(ctx, id))
}

// ConsentToDeclarationEndpoint records a head's consent to the current
// declaration for their organisation and answers with the organisation as
// its heads consent for it.
//
// For a head of that organisation (403 otherwise), also while it is
// suspended; one head's consent covers the organisation and lifts its
// suspension for want of a consent (an admin's stays). It stores the
// operator, the address and the youth-protection restriction (at most 200
// characters) given with it, and confirms the consent by mail, with a copy
// to the operator.
type ConsentToDeclarationEndpoint struct{}

func (ConsentToDeclarationEndpoint) Method() string { return "POST" }
func (ConsentToDeclarationEndpoint) Path() string {
	return "/api/institutions/{id}/declaration-consent"
}
func (ConsentToDeclarationEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ConsentToDeclarationEndpoint) Handle(ctx context.Context, req ConsentToDeclarationReq) (OrganisationConsentResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return OrganisationConsentResp{}, err
	}
	return wire.OrganisationConsentViewOf(core.ConsentToDeclaration(ctx, id, req.consent()))
}

// ConsentToDeclarationReq is the organisation from the path and the head's
// consent, with the organisation's details they confirm.
type ConsentToDeclarationReq struct {
	ID string `path:"id"`

	PersonName         string `json:"personName"`
	PersonFunction     string `json:"personFunction"`
	Email              string `json:"email"`
	Operator           string `json:"operator"`
	Street             string `json:"street"`
	PostalCode         string `json:"postalCode"`
	City               string `json:"city"`
	PromptConstraint   string `json:"promptConstraint"`
	AcceptDeclaration  bool   `json:"acceptDeclaration"`
	DeclarationVersion string `json:"declarationVersion"`
}

func (r ConsentToDeclarationReq) consent() core.DeclarationConsent {
	return core.DeclarationConsent{
		PersonName:         r.PersonName,
		PersonFunction:     r.PersonFunction,
		Email:              r.Email,
		Operator:           r.Operator,
		Street:             r.Street,
		PostalCode:         r.PostalCode,
		City:               r.City,
		PromptConstraint:   r.PromptConstraint,
		AcceptDeclaration:  r.AcceptDeclaration,
		DeclarationVersion: r.DeclarationVersion,
	}
}

// WithdrawConsentEndpoint withdraws a head's organisation's consent to the
// declaration and answers with the organisation as its heads consent for it.
//
// For a head of that organisation (403 otherwise). The organisation is
// suspended until a head consents again; one without a standing consent has
// none to withdraw (409). The consent log keeps the consent and its
// withdrawal.
type WithdrawConsentEndpoint struct{}

func (WithdrawConsentEndpoint) Method() string { return "DELETE" }
func (WithdrawConsentEndpoint) Path() string {
	return "/api/institutions/{id}/declaration-consent"
}
func (WithdrawConsentEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (WithdrawConsentEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (OrganisationConsentResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return OrganisationConsentResp{}, err
	}
	return wire.OrganisationConsentViewOf(core.WithdrawConsent(ctx, id))
}

// OrganisationSuspensionResp is an admin's suspension of an organisation.
type OrganisationSuspensionResp = wire.OrganisationSuspensionView

// SuspendOrganisationEndpoint suspends an organisation and answers with the
// suspension.
//
// Admins only (403 otherwise). Its heads, staff and the guests in its
// workshops are refused everywhere but GET /api/users/me (which reports the
// suspension) and logout; its share links and public workshop pages stop.
// Nothing is deleted; lifting the suspension reverses it. The reason is an
// optional note for admins.
type SuspendOrganisationEndpoint struct{}

func (SuspendOrganisationEndpoint) Method() string             { return "POST" }
func (SuspendOrganisationEndpoint) Path() string               { return "/api/institutions/{id}/suspension" }
func (SuspendOrganisationEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SuspendOrganisationEndpoint) Handle(ctx context.Context, req SuspendOrganisationReq) (OrganisationSuspensionResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return OrganisationSuspensionResp{}, err
	}
	return wire.OrganisationSuspensionViewOf(core.SuspendOrganisation(ctx, id, req.Reason))
}

// SuspendOrganisationReq is the organisation from the path and an optional
// note on why it is suspended.
type SuspendOrganisationReq struct {
	ID     string `path:"id"`
	Reason string `json:"reason,omitempty"`
}

// LiftSuspensionEndpoint lifts an admin's suspension of an organisation and
// answers with the suspension, now empty. Admins only (403 otherwise); an
// organisation without a consent to the current declaration stays suspended
// until a head consents.
type LiftSuspensionEndpoint struct{}

func (LiftSuspensionEndpoint) Method() string             { return "DELETE" }
func (LiftSuspensionEndpoint) Path() string               { return "/api/institutions/{id}/suspension" }
func (LiftSuspensionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (LiftSuspensionEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (OrganisationSuspensionResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return OrganisationSuspensionResp{}, err
	}
	return wire.OrganisationSuspensionViewOf(core.LiftSuspension(ctx, id))
}
