// package: api / schemaf endpoints
// type:    logic
// job:     creates, lists, updates and deletes institutions, and manages their members, API keys and prompt constraints
// limits:  institution storage and its permission checks live in db
package api

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"cgl/db"
	"cgl/legal"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// InstitutionResp is an institution.
type InstitutionResp = obj.Institution

// InstitutionsResp is a list of institutions.
type InstitutionsResp = []obj.Institution

// InstitutionMembersResp is the members of an institution.
type InstitutionMembersResp = []obj.User

// InstitutionApiKeysResp is the API key shares of an institution.
type InstitutionApiKeysResp = []obj.ApiKeyShare

// InstitutionIDReq names an institution by its id.
type InstitutionIDReq struct {
	ID string `path:"id"`
}

// CreateInstitutionEndpoint creates an institution.
//
// The name is required; operator and address are optional.
type CreateInstitutionEndpoint struct{}

func (CreateInstitutionEndpoint) Method() string             { return "POST" }
func (CreateInstitutionEndpoint) Path() string               { return "/api/institutions" }
func (CreateInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateInstitutionEndpoint) Handle(ctx context.Context, req CreateInstitutionReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Institution{}, err
	}
	if req.Name == "" {
		return obj.Institution{}, appError(obj.ErrValidation("Name is required"))
	}
	institution, err := db.CreateInstitution(ctx, user.ID, req.Name, req.details())
	if err != nil {
		return obj.Institution{}, appError(err)
	}
	return *institution, nil
}

// CreateInstitutionReq is a new institution's name, and optionally its
// operator (Träger) and address.
type CreateInstitutionReq struct {
	Name       string `json:"name"`
	Operator   string `json:"operator,omitempty"`
	Street     string `json:"street,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
}

func (r CreateInstitutionReq) details() db.InstitutionDetails {
	return db.InstitutionDetails{Operator: r.Operator, Street: r.Street, PostalCode: r.PostalCode, City: r.City}
}

// ListInstitutionsEndpoint lists the institutions the caller can see.
//
// Admins see each one's suspension reason, consent status and, for one that
// registered itself, how. The filter "recentlyRegistered" narrows the list
// to institutions that registered themselves in the last 14 days, which
// admins check; an unknown filter is refused (400).
type ListInstitutionsEndpoint struct{}

func (ListInstitutionsEndpoint) Method() string             { return "GET" }
func (ListInstitutionsEndpoint) Path() string               { return "/api/institutions" }
func (ListInstitutionsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListInstitutionsEndpoint) Handle(ctx context.Context, req ListInstitutionsReq) (InstitutionsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var registeredSince *time.Time
	for _, f := range req.Filter {
		if f != filterRecentlyRegistered {
			return nil, appError(obj.ErrValidationf("unknown filter %q; known: %s", f, filterRecentlyRegistered))
		}
		since := time.Now().AddDate(0, 0, -recentlyRegisteredDays)
		registeredSince = &since
	}
	institutions, err := db.ListInstitutions(ctx, user.ID, registeredSince)
	if err != nil {
		return nil, appError(err)
	}
	return institutions, nil
}

// filterRecentlyRegistered narrows the institutions to those that registered
// themselves in the last recentlyRegisteredDays days.
const (
	filterRecentlyRegistered = "recentlyRegistered"
	recentlyRegisteredDays   = 14
)

// ListInstitutionsReq is the filters the list is narrowed by.
type ListInstitutionsReq struct {
	Filter []string `query:"filter"`
}

// GetInstitutionEndpoint returns an institution.
type GetInstitutionEndpoint struct{}

func (GetInstitutionEndpoint) Method() string             { return "GET" }
func (GetInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (GetInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Institution{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return obj.Institution{}, err
	}
	institution, err := db.GetInstitutionByID(ctx, user.ID, id)
	if err != nil {
		return obj.Institution{}, appError(err)
	}
	return *institution, nil
}

// UpdateInstitutionEndpoint renames an institution.
//
// The name is required.
type UpdateInstitutionEndpoint struct{}

func (UpdateInstitutionEndpoint) Method() string             { return "PATCH" }
func (UpdateInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (UpdateInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateInstitutionEndpoint) Handle(ctx context.Context, req UpdateInstitutionReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Institution{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return obj.Institution{}, err
	}
	if req.Name == "" {
		return obj.Institution{}, appError(obj.ErrValidation("Name is required"))
	}
	institution, err := db.UpdateInstitution(ctx, id, user.ID, req.Name)
	if err != nil {
		return obj.Institution{}, appError(err)
	}
	return *institution, nil
}

// UpdateInstitutionReq is the institution from the path and its new name.
type UpdateInstitutionReq struct {
	ID   string `path:"id"`
	Name string `json:"name"`
}

// DeleteInstitutionEndpoint deletes an institution for good, with its
// workshops and the guests in them; heads and staff keep their accounts as
// individuals. Admins only. To stop an institution without losing anything,
// suspend it (POST /api/institutions/{id}/suspension).
type DeleteInstitutionEndpoint struct{}

func (DeleteInstitutionEndpoint) Method() string             { return "DELETE" }
func (DeleteInstitutionEndpoint) Path() string               { return "/api/institutions/{id}" }
func (DeleteInstitutionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteInstitutionEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.DeleteInstitution(ctx, id, user.ID); err != nil {
		return MessageResp{}, appError(err)
	}
	return MessageResp{Message: "Institution deleted"}, nil
}

// GetInstitutionMembersEndpoint lists the members of an institution.
type GetInstitutionMembersEndpoint struct{}

func (GetInstitutionMembersEndpoint) Method() string             { return "GET" }
func (GetInstitutionMembersEndpoint) Path() string               { return "/api/institutions/{id}/members" }
func (GetInstitutionMembersEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionMembersEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (InstitutionMembersResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	institutionID, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return nil, err
	}
	members, err := db.GetInstitutionMembers(ctx, institutionID, user.ID)
	if err != nil {
		return nil, appError(err)
	}
	return members, nil
}

// RemoveInstitutionMemberEndpoint removes a member from an institution; heads
// and admins only.
type RemoveInstitutionMemberEndpoint struct{}

func (RemoveInstitutionMemberEndpoint) Method() string { return "DELETE" }
func (RemoveInstitutionMemberEndpoint) Path() string {
	return "/api/institutions/{id}/members/{userID}"
}
func (RemoveInstitutionMemberEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveInstitutionMemberEndpoint) Handle(ctx context.Context, req RemoveInstitutionMemberReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	institutionID, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return MessageResp{}, err
	}
	userID, err := parseIDValidation(req.UserID, "Invalid user ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.RemoveInstitutionMember(ctx, institutionID, userID, user.ID); err != nil {
		return MessageResp{}, appError(err)
	}
	return MessageResp{Message: "Member removed"}, nil
}

// RemoveInstitutionMemberReq names an institution and one of its members.
type RemoveInstitutionMemberReq struct {
	ID     string `path:"id"`
	UserID string `path:"userID"`
}

// GetInstitutionApiKeysEndpoint lists the API keys shared with an institution;
// heads and staff only.
type GetInstitutionApiKeysEndpoint struct{}

func (GetInstitutionApiKeysEndpoint) Method() string             { return "GET" }
func (GetInstitutionApiKeysEndpoint) Path() string               { return "/api/institutions/{id}/apikeys" }
func (GetInstitutionApiKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionApiKeysEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (InstitutionApiKeysResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	institutionID, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return nil, err
	}
	shares, err := db.GetApiKeySharesByInstitution(ctx, user.ID, institutionID)
	if err != nil {
		return nil, appError(err)
	}
	return shares, nil
}

// SetInstitutionFreeUseApiKeyShareEndpoint sets or clears an institution's
// free-use API key share and answers with the institution.
//
// Any member of the institution plays for free with this key. A null shareId
// clears it.
type SetInstitutionFreeUseApiKeyShareEndpoint struct{}

func (SetInstitutionFreeUseApiKeyShareEndpoint) Method() string { return "PATCH" }
func (SetInstitutionFreeUseApiKeyShareEndpoint) Path() string {
	return "/api/institutions/{id}/free-use-key"
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Handle(ctx context.Context, req SetInstitutionFreeUseKeyReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Institution{}, err
	}
	institutionID, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return obj.Institution{}, err
	}
	if err := db.SetInstitutionFreeUseApiKeyShare(ctx, user.ID, institutionID, req.ShareID); err != nil {
		return obj.Institution{}, appError(err)
	}
	institution, err := db.GetInstitutionByID(ctx, user.ID, institutionID)
	if err != nil {
		return obj.Institution{}, fmt.Errorf("failed to get updated institution: %w", err)
	}
	return *institution, nil
}

// SetInstitutionFreeUseKeyReq is the institution from the path and the share
// to set, or null to clear it.
type SetInstitutionFreeUseKeyReq struct {
	ID      string     `path:"id"`
	ShareID *uuid.UUID `json:"shareId"`
}

// UpdateInstitutionPromptConstraintsEndpoint sets or clears an institution's
// prompt constraints and answers with the institution; heads and admins only.
//
// An empty or null promptConstraints clears them; they have at most 200
// characters.
type UpdateInstitutionPromptConstraintsEndpoint struct{}

func (UpdateInstitutionPromptConstraintsEndpoint) Method() string { return "PATCH" }
func (UpdateInstitutionPromptConstraintsEndpoint) Path() string {
	return "/api/institutions/{id}/prompt-constraints"
}
func (UpdateInstitutionPromptConstraintsEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (UpdateInstitutionPromptConstraintsEndpoint) Handle(ctx context.Context, req UpdateInstitutionPromptConstraintsReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Institution{}, err
	}
	institutionID, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return obj.Institution{}, err
	}
	constraint := req.PromptConstraints
	if constraint != nil && *constraint == "" {
		constraint = nil
	}
	if constraint != nil && utf8.RuneCountInString(*constraint) > legal.MaxPromptConstraintLength {
		return obj.Institution{}, appError(obj.ErrValidationf("prompt constraints have at most %d characters", legal.MaxPromptConstraintLength))
	}
	if err := db.UpdateInstitutionPromptConstraints(ctx, user.ID, institutionID, constraint); err != nil {
		return obj.Institution{}, appError(err)
	}
	institution, err := db.GetInstitutionByID(ctx, user.ID, institutionID)
	if err != nil {
		return obj.Institution{}, fmt.Errorf("failed to get updated institution: %w", err)
	}
	return *institution, nil
}

// UpdateInstitutionPromptConstraintsReq is the institution from the path and
// its new prompt constraints.
type UpdateInstitutionPromptConstraintsReq struct {
	ID                string  `path:"id"`
	PromptConstraints *string `json:"promptConstraints"`
}
