// package: api / schemaf endpoints
// type:    logic
// job:     creates, lists, reads, updates and deletes organisations, and lists and removes their members
// limits:  decides nothing: the core does (core/organisation*.go, core/membership.go); key shares and the free-use key are apikeys.go's
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// InstitutionResp is an organisation as the caller sees it.
type InstitutionResp = wire.OrganisationView

// InstitutionsResp is a list of organisations.
// OrganisationViewPageResp is a page of organisations and the total.
type OrganisationViewPageResp = wire.Page[wire.OrganisationView]

// InstitutionMembersResp is the members of an organisation.
// MemberPageResp is a page of an organisation's members and the total.
type MemberPageResp = wire.Page[wire.UserView]

// InstitutionIDReq names an organisation by its id.
type InstitutionIDReq struct {
	ID string `path:"id"`
}

// CreateInstitutionEndpoint creates an organisation for others: the admin's.
//
// The name is required; operator and address are optional. An organisation
// registers itself through POST /api/auth/register instead.
type CreateInstitutionEndpoint struct{}

func (CreateInstitutionEndpoint) Method() string             { return "POST" }
func (CreateInstitutionEndpoint) Path() string               { return "/api/institutions" }
func (CreateInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateInstitutionEndpoint) Handle(ctx context.Context, req CreateInstitutionReq) (InstitutionResp, error) {
	return wire.OrganisationViewOf(core.CreateOrganisation(ctx, core.OrganisationDraft(req)))
}

// CreateInstitutionReq is a new organisation's name, and optionally its
// operator (Träger) and address.
type CreateInstitutionReq struct {
	Name       string `json:"name"`
	Operator   string `json:"operator,omitempty"`
	Street     string `json:"street,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
}

// ListInstitutionsEndpoint lists a page of the organisations the caller may
// list, and the total: an admin all of them, a head or staff their own.
type ListInstitutionsEndpoint struct{}

func (ListInstitutionsEndpoint) Method() string             { return "GET" }
func (ListInstitutionsEndpoint) Path() string               { return "/api/institutions" }
func (ListInstitutionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInstitutionsEndpoint) Handle(ctx context.Context, req ListInstitutionsReq) (OrganisationViewPageResp, error) {
	return wire.OrganisationViewPageOf(core.List(ctx, core.Organisations, req.query()))
}

// OrganisationFilter narrows the organisation list.
type OrganisationFilter string

const (
	// OrganisationFilterRecentlyRegistered keeps those that registered
	// themselves in the last 14 days, among those whose record the caller sees.
	OrganisationFilterRecentlyRegistered OrganisationFilter = "recentlyRegistered"
)

// OrganisationSort orders the organisation list; the default is name.
type OrganisationSort string

const (
	OrganisationSortName         OrganisationSort = "name"
	OrganisationSortCreatedAt    OrganisationSort = "createdAt"
	OrganisationSortRegisteredAt OrganisationSort = "registeredAt"
)

// ListInstitutionsReq filters, sorts and pages the organisation list.
type ListInstitutionsReq struct {
	ListParams[OrganisationFilter, OrganisationSort]
}

// GetInstitutionEndpoint returns an organisation the caller may read: their
// own, as a member or a guest in one of its workshops; any, for an admin.
// Others get 404.
type GetInstitutionEndpoint struct{}

func (GetInstitutionEndpoint) Method() string             { return "GET" }
func (GetInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (GetInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (InstitutionResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return InstitutionResp{}, err
	}
	return wire.OrganisationViewOf(core.ReadOrganisation(ctx, id))
}

// UpdateInstitutionEndpoint changes an organisation: its name, its
// youth-protection restriction, or both, and answers with it as stored.
//
// Only what is given changes; an empty promptConstraints removes the
// restriction, so the site's rule applies. Renaming takes mayEditCard, the
// restriction mayEditSettings; a request with a part the caller may not
// change is refused as a whole.
type UpdateInstitutionEndpoint struct{}

func (UpdateInstitutionEndpoint) Method() string             { return "PATCH" }
func (UpdateInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (UpdateInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateInstitutionEndpoint) Handle(ctx context.Context, req UpdateInstitutionReq) (InstitutionResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return InstitutionResp{}, err
	}
	return wire.OrganisationViewOf(core.UpdateOrganisation(ctx, id, core.OrganisationChanges{
		Name:             req.Name,
		PromptConstraint: req.PromptConstraints,
	}))
}

// UpdateInstitutionReq is the organisation from the path and what changes.
type UpdateInstitutionReq struct {
	ID                string  `path:"id"`
	Name              *string `json:"name,omitempty"`
	PromptConstraints *string `json:"promptConstraints,omitempty"`
}

// DeleteInstitutionEndpoint deletes an organisation for good, with its
// workshops and the guests in them; its members stay, as individuals.
// Suspending keeps its data instead (POST /api/institutions/{id}/suspension).
type DeleteInstitutionEndpoint struct{}

func (DeleteInstitutionEndpoint) Method() string             { return "DELETE" }
func (DeleteInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (DeleteInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteInstitutionEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.DeleteOrganisation(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Institution deleted"}, nil
}

// GetInstitutionMembersEndpoint lists a page of an organisation's members
// (individual, staff, head), and the total: for its members and the admin.
// Anyone else gets an empty list; an organisation the caller may not read,
// 404.
type GetInstitutionMembersEndpoint struct{}

func (GetInstitutionMembersEndpoint) Method() string             { return "GET" }
func (GetInstitutionMembersEndpoint) Path() string               { return "/api/institutions/{id}/members" }
func (GetInstitutionMembersEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionMembersEndpoint) Handle(ctx context.Context, req ListMembersReq) (MemberPageResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return MemberPageResp{}, err
	}
	return wire.UserViewPageOf(core.List(ctx, core.Members(id), req.query()))
}

// MemberSort orders an organisation's member list; the default is name.
type MemberSort string

const (
	MemberSortName  MemberSort = "name"
	MemberSortEmail MemberSort = "email"
	MemberSortRole  MemberSort = "role"
)

// ListMembersReq names the organisation, and sorts and pages its members.
type ListMembersReq struct {
	ID string `path:"id"`
	ListParams[NoFilter, MemberSort]
}

// RemoveInstitutionMemberEndpoint removes a user from an organisation; they
// keep their account, as an individual. A head removes members and guests,
// staff remove guests, a member leaves, an admin removes anyone; the last
// head stays (409 last_head).
type RemoveInstitutionMemberEndpoint struct{}

func (RemoveInstitutionMemberEndpoint) Method() string { return "DELETE" }
func (RemoveInstitutionMemberEndpoint) Path() string {
	return "/api/institutions/{id}/members/{userID}"
}
func (RemoveInstitutionMemberEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveInstitutionMemberEndpoint) Handle(ctx context.Context, req RemoveInstitutionMemberReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return MessageResp{}, err
	}
	user, err := parseID(req.UserID, "Invalid user ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.RemoveMember(ctx, id, user); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Member removed"}, nil
}

// RemoveInstitutionMemberReq names an organisation and one of its members.
type RemoveInstitutionMemberReq struct {
	ID     string `path:"id"`
	UserID string `path:"userID"`
}
