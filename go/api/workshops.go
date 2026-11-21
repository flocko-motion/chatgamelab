// package: api / schemaf endpoints
// type:    logic
// job:     creates, lists, updates and deletes workshops, assigns their API key, manages their members and their Guests' sign-in codes
// limits:  workshop storage and its permission checks live in db; the event stream (-> workshop_events.go)
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/events"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// WorkshopResp is a workshop.
type WorkshopResp = obj.Workshop

// WorkshopsResp is a list of workshops.
type WorkshopsResp = []obj.Workshop

// WorkshopIDReq names a workshop by its id.
type WorkshopIDReq struct {
	ID string `path:"id"`
}

// CreateWorkshopEndpoint creates a workshop.
type CreateWorkshopEndpoint struct{}

func (CreateWorkshopEndpoint) Method() string             { return "POST" }
func (CreateWorkshopEndpoint) Path() string               { return "/api/workshops" }
func (CreateWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopEndpoint) Handle(ctx context.Context, req CreateWorkshopReq) (WorkshopResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Workshop{}, err
	}
	if req.Name == "" {
		return obj.Workshop{}, appError(obj.ErrValidation("Name is required"))
	}
	var institutionID *uuid.UUID
	if req.InstitutionID != nil {
		id, err := parseIDValidation(*req.InstitutionID, "Invalid institution ID")
		if err != nil {
			return obj.Workshop{}, err
		}
		institutionID = &id
	}
	workshop, err := db.CreateWorkshop(ctx, user.ID, institutionID, req.Name, req.Active, req.Public)
	if err != nil {
		return obj.Workshop{}, appError(err)
	}
	return *workshop, nil
}

// CreateWorkshopReq is a new workshop's institution, name and flags.
type CreateWorkshopReq struct {
	InstitutionID *string `json:"institutionId,omitempty"`
	Name          string  `json:"name"`
	Active        bool    `json:"active"`
	Public        bool    `json:"public"`
}

// ListWorkshopsEndpoint lists the workshops the caller can see, or those of
// one institution.
//
// Query: institutionId, search (name), sortBy (name, createdAt,
// participantCount), sortDir (asc, desc), activeOnly ("true" lists active
// workshops only).
type ListWorkshopsEndpoint struct{}

func (ListWorkshopsEndpoint) Method() string             { return "GET" }
func (ListWorkshopsEndpoint) Path() string               { return "/api/workshops" }
func (ListWorkshopsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListWorkshopsEndpoint) Handle(ctx context.Context, req ListWorkshopsReq) (WorkshopsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var institutionID *uuid.UUID
	if opt(req.InstitutionID) != "" {
		id, err := parseIDValidation(opt(req.InstitutionID), "Invalid institution ID")
		if err != nil {
			return nil, err
		}
		institutionID = &id
	}
	var activeOnly *bool
	if opt(req.ActiveOnly) == "true" {
		t := true
		activeOnly = &t
	}
	workshops, err := db.ListWorkshopsWithOptions(ctx, user.ID, institutionID, db.ListWorkshopsOptions{
		Search:     opt(req.Search),
		SortBy:     opt(req.SortBy),
		SortDir:    opt(req.SortDir),
		ActiveOnly: activeOnly,
	})
	if err != nil {
		return nil, appError(err)
	}
	return workshops, nil
}

// ListWorkshopsReq filters and sorts the workshop list.
type ListWorkshopsReq struct {
	InstitutionID *string `query:"institutionId"`
	Search        *string `query:"search"`
	SortBy        *string `query:"sortBy"`
	SortDir       *string `query:"sortDir"`
	ActiveOnly    *string `query:"activeOnly"`
}

// GetWorkshopEndpoint returns a workshop.
type GetWorkshopEndpoint struct{}

func (GetWorkshopEndpoint) Method() string             { return "GET" }
func (GetWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (GetWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetWorkshopEndpoint) Handle(ctx context.Context, req WorkshopIDReq) (WorkshopResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Workshop{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid workshop ID")
	if err != nil {
		return obj.Workshop{}, err
	}
	workshop, err := db.GetWorkshopByID(ctx, user.ID, id)
	if err != nil {
		return obj.Workshop{}, appError(err)
	}
	return *workshop, nil
}

// UpdateWorkshopEndpoint updates a workshop's settings and its public page.
//
// The name is required. Omitting publicSlug, publicDescription or publicLinks
// keeps the current value; an empty description or list clears it.
type UpdateWorkshopEndpoint struct{}

func (UpdateWorkshopEndpoint) Method() string             { return "PATCH" }
func (UpdateWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (UpdateWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateWorkshopEndpoint) Handle(ctx context.Context, req UpdateWorkshopReq) (WorkshopResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Workshop{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid workshop ID")
	if err != nil {
		return obj.Workshop{}, err
	}
	if req.Name == "" {
		return obj.Workshop{}, appError(obj.ErrValidation("Name is required"))
	}
	workshop, err := db.UpdateWorkshop(ctx, id, user.ID, db.WorkshopUpdate{
		Name:                       req.Name,
		Active:                     req.Active,
		Public:                     req.Public,
		AiQualityTier:              req.AiQualityTier,
		PromptConstraints:          req.PromptConstraints,
		ShowPublicGames:            req.ShowPublicGames,
		ShowOtherParticipantsGames: req.ShowOtherParticipantsGames,
		IsPaused:                   req.IsPaused,
		AllowGameSharing:           req.AllowGameSharing,
		PublicSlug:                 req.PublicSlug,
		PublicDescription:          req.PublicDescription,
		PublicLinks:                req.PublicLinks,
	})
	if err != nil {
		return obj.Workshop{}, appError(err)
	}
	events.GetBroker().PublishWorkshopUpdated(id)
	return *workshop, nil
}

// UpdateWorkshopReq is the workshop from the path and its new settings from
// the body.
type UpdateWorkshopReq struct {
	ID                         string  `path:"id"`
	Name                       string  `json:"name"`
	Active                     bool    `json:"active"`
	Public                     bool    `json:"public"`
	AiQualityTier              *string `json:"aiQualityTier,omitempty"`
	PromptConstraints          *string `json:"promptConstraints,omitempty"`
	ShowPublicGames            bool    `json:"showPublicGames"`
	ShowOtherParticipantsGames bool    `json:"showOtherParticipantsGames"`
	IsPaused                   bool    `json:"isPaused"`
	AllowGameSharing           bool    `json:"allowGameSharing"`
	// PublicSlug: omitted keeps the current link.
	PublicSlug *string `json:"publicSlug,omitempty"`
	// PublicDescription: omitted keeps the current text, "" clears it.
	PublicDescription *string `json:"publicDescription,omitempty"`
	// PublicLinks: omitted keeps the current further reading, an empty list clears it.
	PublicLinks *[]obj.PublicWorkshopLink `json:"publicLinks,omitempty"`
}

// DeleteWorkshopEndpoint soft-deletes a workshop.
type DeleteWorkshopEndpoint struct{}

func (DeleteWorkshopEndpoint) Method() string             { return "DELETE" }
func (DeleteWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (DeleteWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteWorkshopEndpoint) Handle(ctx context.Context, req WorkshopIDReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid workshop ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.DeleteWorkshop(ctx, id, user.ID); err != nil {
		return MessageResp{}, appError(err)
	}
	return MessageResp{Message: "Workshop deleted"}, nil
}

// SetWorkshopApiKeyEndpoint sets the API key a workshop's members play
// with; staff and heads only.
//
// A null or empty apiKeyShareId removes it.
type SetWorkshopApiKeyEndpoint struct{}

func (SetWorkshopApiKeyEndpoint) Method() string             { return "PUT" }
func (SetWorkshopApiKeyEndpoint) Path() string               { return "/api/workshops/{id}/api-key" }
func (SetWorkshopApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetWorkshopApiKeyEndpoint) Handle(ctx context.Context, req SetWorkshopApiKeyReq) (WorkshopResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.Workshop{}, err
	}
	id, err := parseIDValidation(req.ID, "Invalid workshop ID")
	if err != nil {
		return obj.Workshop{}, err
	}
	var apiKeyShareID *uuid.UUID
	if req.ApiKeyShareID != nil && *req.ApiKeyShareID != "" {
		parsed, err := parseIDValidation(*req.ApiKeyShareID, "Invalid API key share ID")
		if err != nil {
			return obj.Workshop{}, err
		}
		apiKeyShareID = &parsed
	}
	workshop, err := db.SetWorkshopDefaultApiKey(ctx, id, user.ID, apiKeyShareID)
	if err != nil {
		return obj.Workshop{}, appError(err)
	}
	events.GetBroker().PublishWorkshopUpdated(id)
	return *workshop, nil
}

// SetWorkshopApiKeyReq is the workshop from the path and the API key share
// from the body.
type SetWorkshopApiKeyReq struct {
	ID            string  `path:"id"`
	ApiKeyShareID *string `json:"apiKeyShareId"`
}

// AddWorkshopMemberEndpoint adds a member of the organization to a workshop by
// making it their active workshop.
type AddWorkshopMemberEndpoint struct{}

func (AddWorkshopMemberEndpoint) Method() string             { return "POST" }
func (AddWorkshopMemberEndpoint) Path() string               { return "/api/workshops/{id}/members" }
func (AddWorkshopMemberEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (AddWorkshopMemberEndpoint) Handle(ctx context.Context, req AddWorkshopMemberReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	workshopID, err := parseIDValidation(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return MessageResp{}, err
	}
	if req.UserID == "" {
		return MessageResp{}, appError(obj.ErrValidation("userId is required"))
	}
	targetUserID, err := parseIDValidation(req.UserID, "Invalid user ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.AddMemberToWorkshop(ctx, user.ID, workshopID, targetUserID); err != nil {
		return MessageResp{}, appError(err)
	}
	events.GetBroker().PublishMembersUpdated(workshopID)
	return MessageResp{Message: "Member added to workshop"}, nil
}

// AddWorkshopMemberReq is the workshop from the path and the user to add from
// the body.
type AddWorkshopMemberReq struct {
	WorkshopID string `path:"id"`
	UserID     string `json:"userId"`
}

// RemoveWorkshopMemberEndpoint removes a member who is not permanent from a
// workshop by clearing their active workshop.
type RemoveWorkshopMemberEndpoint struct{}

func (RemoveWorkshopMemberEndpoint) Method() string             { return "DELETE" }
func (RemoveWorkshopMemberEndpoint) Path() string               { return "/api/workshops/{id}/members/{userId}" }
func (RemoveWorkshopMemberEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveWorkshopMemberEndpoint) Handle(ctx context.Context, req RemoveWorkshopMemberReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	workshopID, err := parseIDValidation(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return MessageResp{}, err
	}
	targetUserID, err := parseIDValidation(req.UserID, "Invalid user ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := db.RemoveMemberFromWorkshop(ctx, user.ID, workshopID, targetUserID); err != nil {
		return MessageResp{}, appError(err)
	}
	events.GetBroker().PublishMembersUpdated(workshopID)
	return MessageResp{Message: "Member removed from workshop"}, nil
}

// RemoveWorkshopMemberReq names a workshop and the member to remove.
type RemoveWorkshopMemberReq struct {
	WorkshopID string `path:"id"`
	UserID     string `path:"userId"`
}

// WorkshopGuestReq names a Guest in a workshop by their user id.
type WorkshopGuestReq struct {
	GuestID string `path:"guestId"`
}

// GuestTokenResp is a Guest's sign-in code.
type GuestTokenResp struct {
	Token string `json:"token"`
}

// GetGuestTokenEndpoint returns the sign-in code of a Guest in a workshop;
// staff and heads only.
type GetGuestTokenEndpoint struct{}

func (GetGuestTokenEndpoint) Method() string { return "GET" }
func (GetGuestTokenEndpoint) Path() string {
	return "/api/workshops/guests/{guestId}/token"
}
func (GetGuestTokenEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGuestTokenEndpoint) Handle(ctx context.Context, req WorkshopGuestReq) (GuestTokenResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return GuestTokenResp{}, err
	}
	guestID, err := parseIDValidation(req.GuestID, "Invalid guest ID")
	if err != nil {
		return GuestTokenResp{}, err
	}
	token, err := db.GetGuestToken(ctx, guestID, user.ID)
	if err != nil {
		return GuestTokenResp{}, appError(err)
	}
	return GuestTokenResp{Token: token}, nil
}

// ResetGuestTokenEndpoint replaces the sign-in code of a Guest in a workshop;
// staff and heads only.
//
// Earlier sign-in links stop working and the Guest's sessions end.
type ResetGuestTokenEndpoint struct{}

func (ResetGuestTokenEndpoint) Method() string { return "POST" }
func (ResetGuestTokenEndpoint) Path() string {
	return "/api/workshops/guests/{guestId}/token/reset"
}
func (ResetGuestTokenEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ResetGuestTokenEndpoint) Handle(ctx context.Context, req WorkshopGuestReq) (GuestTokenResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return GuestTokenResp{}, err
	}
	guestID, err := parseIDValidation(req.GuestID, "Invalid guest ID")
	if err != nil {
		return GuestTokenResp{}, err
	}
	token, err := db.ResetGuestToken(ctx, guestID, user.ID)
	if err != nil {
		return GuestTokenResp{}, appError(err)
	}
	// The old token signed sessions in; they end with it.
	if err := schemafapi.RevokeSubject(ctx, guestID.String()); err != nil {
		return GuestTokenResp{}, fmt.Errorf("failed to end guest sessions: %w", err)
	}
	return GuestTokenResp{Token: token}, nil
}
