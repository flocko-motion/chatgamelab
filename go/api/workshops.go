// package: api / schemaf endpoints
// type:    logic
// job:     creates, lists, reads, updates and deletes workshops, puts people in and out of them, and reads and resets guests' login codes
// limits:  decides nothing: the core does (core/workshop*.go); a workshop's key is apikeys.go's
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// WorkshopResp is a workshop as the caller sees it.
type WorkshopResp = wire.WorkshopView

// WorkshopsResp is a list of workshops.
// WorkshopViewPageResp is a page of workshops and the total.
type WorkshopViewPageResp = wire.Page[wire.WorkshopView]

// WorkshopIDReq names a workshop by its id.
type WorkshopIDReq struct {
	ID string `path:"id"`
}

// CreateWorkshopEndpoint creates a workshop in the caller's organisation, or,
// for the admin, in the one named (institutionId).
type CreateWorkshopEndpoint struct{}

func (CreateWorkshopEndpoint) Method() string             { return "POST" }
func (CreateWorkshopEndpoint) Path() string               { return "/api/workshops" }
func (CreateWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateWorkshopEndpoint) Handle(ctx context.Context, req CreateWorkshopReq) (WorkshopResp, error) {
	org, err := optionalID(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return WorkshopResp{}, err
	}
	return wire.WorkshopViewOf(core.CreateWorkshop(ctx, core.WorkshopDraft{
		Organisation: org, Name: req.Name, Active: req.Active, Public: req.Public,
	}))
}

// CreateWorkshopReq is a new workshop.
type CreateWorkshopReq struct {
	InstitutionID *string `json:"institutionId,omitempty"`
	Name          string  `json:"name"`
	Active        bool    `json:"active"`
	Public        bool    `json:"public"`
}

// ListWorkshopsEndpoint lists a page of the workshops the caller may list,
// and the total: an admin all of them, a head or staff their
// organisation's. institutionId narrows it to one organisation.
type ListWorkshopsEndpoint struct{}

func (ListWorkshopsEndpoint) Method() string             { return "GET" }
func (ListWorkshopsEndpoint) Path() string               { return "/api/workshops" }
func (ListWorkshopsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListWorkshopsEndpoint) Handle(ctx context.Context, req ListWorkshopsReq) (WorkshopViewPageResp, error) {
	org, err := optionalID(req.InstitutionID, "Invalid institution ID")
	if err != nil {
		return WorkshopViewPageResp{}, err
	}
	return wire.WorkshopViewPageOf(core.List(ctx, core.Workshops(org), req.query()))
}

// WorkshopFilter narrows the workshop list.
type WorkshopFilter string

const (
	WorkshopFilterActive WorkshopFilter = "active" // the active workshops
)

// WorkshopSort orders the workshop list; the default is createdAt, newest first.
type WorkshopSort string

const (
	WorkshopSortName             WorkshopSort = "name"
	WorkshopSortCreatedAt        WorkshopSort = "createdAt"
	WorkshopSortModifiedAt       WorkshopSort = "modifiedAt"
	WorkshopSortParticipantCount WorkshopSort = "participantCount"
)

// ListWorkshopsReq narrows the list to an organisation, and filters, sorts
// and pages it.
type ListWorkshopsReq struct {
	InstitutionID *string `query:"institutionId"`
	ListParams[WorkshopFilter, WorkshopSort]
}

// ListParticipantsEndpoint lists a page of a workshop's participants, and
// the total: for those who may list them (an empty list for others); a
// workshop the caller may not read, 404.
type ListParticipantsEndpoint struct{}

func (ListParticipantsEndpoint) Method() string             { return "GET" }
func (ListParticipantsEndpoint) Path() string               { return "/api/workshops/{id}/participants" }
func (ListParticipantsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListParticipantsEndpoint) Handle(ctx context.Context, req ListParticipantsReq) (ParticipantPageResp, error) {
	id, err := parseID(req.ID, "Invalid workshop ID")
	if err != nil {
		return ParticipantPageResp{}, err
	}
	return wire.WorkshopParticipantViewPageOf(id)(core.List(ctx, core.Participants(id), req.query()))
}

// ParticipantPageResp is a page of a workshop's participants and the total.
type ParticipantPageResp = wire.Page[wire.WorkshopParticipantView]

// ParticipantSort orders a workshop's participants; the default is
// createdAt (when they joined), first first.
type ParticipantSort string

const (
	ParticipantSortName       ParticipantSort = "name"
	ParticipantSortRole       ParticipantSort = "role"
	ParticipantSortGamesCount ParticipantSort = "gamesCount"
	ParticipantSortCreatedAt  ParticipantSort = "createdAt"
)

// ListParticipantsReq names the workshop, and sorts and pages its participants.
type ListParticipantsReq struct {
	ID string `path:"id"`
	ListParams[NoFilter, ParticipantSort]
}

// GetWorkshopEndpoint returns a workshop the caller may read: one of their
// organisation's, the one they are in, one with a public page, or any, for
// an admin. Others get 404.
type GetWorkshopEndpoint struct{}

func (GetWorkshopEndpoint) Method() string             { return "GET" }
func (GetWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (GetWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetWorkshopEndpoint) Handle(ctx context.Context, req WorkshopIDReq) (WorkshopResp, error) {
	id, err := parseID(req.ID, "Invalid workshop ID")
	if err != nil {
		return WorkshopResp{}, err
	}
	return wire.WorkshopViewOf(core.ReadWorkshop(ctx, id))
}

// UpdateWorkshopEndpoint changes a workshop and answers with it as stored.
//
// Only what is given changes. An empty aiQualityTier, promptConstraints or
// publicDescription removes it; an empty publicLinks list clears the links.
// Switching public off ends the public page's share links.
type UpdateWorkshopEndpoint struct{}

func (UpdateWorkshopEndpoint) Method() string             { return "PATCH" }
func (UpdateWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (UpdateWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateWorkshopEndpoint) Handle(ctx context.Context, req UpdateWorkshopReq) (WorkshopResp, error) {
	id, err := parseID(req.ID, "Invalid workshop ID")
	if err != nil {
		return WorkshopResp{}, err
	}
	return wire.WorkshopViewOf(core.UpdateWorkshop(ctx, id, core.WorkshopChanges{
		Name:              req.Name,
		Active:            req.Active,
		Paused:            req.IsPaused,
		Public:            req.Public,
		PublicSlug:        req.PublicSlug,
		AiQualityTier:     req.AiQualityTier,
		ShowsPublicGames:  req.ShowPublicGames,
		ShowsOthersGames:  req.ShowOtherParticipantsGames,
		AllowsGameSharing: req.AllowGameSharing,
		PromptConstraint:  req.PromptConstraints,
		PublicDescription: req.PublicDescription,
		PublicLinks:       req.PublicLinks,
	}))
}

// UpdateWorkshopReq is the workshop from the path and what changes.
type UpdateWorkshopReq struct {
	ID                         string                    `path:"id"`
	Name                       *string                   `json:"name,omitempty"`
	Active                     *bool                     `json:"active,omitempty"`
	IsPaused                   *bool                     `json:"isPaused,omitempty"`
	Public                     *bool                     `json:"public,omitempty"`
	PublicSlug                 *string                   `json:"publicSlug,omitempty"`
	AiQualityTier              *string                   `json:"aiQualityTier,omitempty"`
	ShowPublicGames            *bool                     `json:"showPublicGames,omitempty"`
	ShowOtherParticipantsGames *bool                     `json:"showOtherParticipantsGames,omitempty"`
	AllowGameSharing           *bool                     `json:"allowGameSharing,omitempty"`
	PromptConstraints          *string                   `json:"promptConstraints,omitempty"`
	PublicDescription          *string                   `json:"publicDescription,omitempty"`
	PublicLinks                *[]obj.PublicWorkshopLink `json:"publicLinks,omitempty"`
}

// DeleteWorkshopEndpoint deletes a workshop with its guests; its other games
// stay with their creators.
type DeleteWorkshopEndpoint struct{}

func (DeleteWorkshopEndpoint) Method() string             { return "DELETE" }
func (DeleteWorkshopEndpoint) Path() string               { return "/api/workshops/{id}" }
func (DeleteWorkshopEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteWorkshopEndpoint) Handle(ctx context.Context, req WorkshopIDReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid workshop ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.DeleteWorkshop(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Workshop deleted"}, nil
}

// RemoveWorkshopMemberEndpoint takes someone out of a workshop: a guest of it
// loses their place, a member working in it leaves it.
type RemoveWorkshopMemberEndpoint struct{}

func (RemoveWorkshopMemberEndpoint) Method() string             { return "DELETE" }
func (RemoveWorkshopMemberEndpoint) Path() string               { return "/api/workshops/{id}/members/{userId}" }
func (RemoveWorkshopMemberEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RemoveWorkshopMemberEndpoint) Handle(ctx context.Context, req RemoveWorkshopMemberReq) (MessageResp, error) {
	id, err := parseID(req.WorkshopID, "Invalid workshop ID")
	if err != nil {
		return MessageResp{}, err
	}
	user, err := parseID(req.UserID, "Invalid user ID")
	if err != nil {
		return MessageResp{}, err
	}
	if err := core.RemoveFromWorkshop(ctx, id, user); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Member removed from workshop"}, nil
}

// RemoveWorkshopMemberReq names a workshop and the person to take out.
type RemoveWorkshopMemberReq struct {
	WorkshopID string `path:"id"`
	UserID     string `path:"userId"`
}

// WorkshopGuestReq names a guest in a workshop by their user id.
type WorkshopGuestReq struct {
	GuestID string `path:"guestId"`
}

// GuestTokenResp is a guest's login code.
type GuestTokenResp struct {
	Token string `json:"token"`
}

// GetGuestTokenEndpoint returns the login code of a guest in one of the
// caller's organisation's workshops: for its head and staff, helping a guest
// who lost it.
type GetGuestTokenEndpoint struct{}

func (GetGuestTokenEndpoint) Method() string { return "GET" }
func (GetGuestTokenEndpoint) Path() string {
	return "/api/workshops/guests/{guestId}/token"
}
func (GetGuestTokenEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGuestTokenEndpoint) Handle(ctx context.Context, req WorkshopGuestReq) (GuestTokenResp, error) {
	id, err := parseID(req.GuestID, "Invalid guest ID")
	if err != nil {
		return GuestTokenResp{}, err
	}
	secrets, err := wire.UserSecretsViewOf(core.ReadUser(ctx, id))
	if err != nil {
		return GuestTokenResp{}, err
	}
	if secrets.Token == nil {
		return GuestTokenResp{}, core.ErrNotFound
	}
	return GuestTokenResp{Token: *secrets.Token}, nil
}

// ResetGuestTokenEndpoint gives a guest in one of the caller's organisation's
// workshops a new login code: for its head and staff. Earlier login links
// stop working and the guest's sessions end.
type ResetGuestTokenEndpoint struct{}

func (ResetGuestTokenEndpoint) Method() string { return "POST" }
func (ResetGuestTokenEndpoint) Path() string {
	return "/api/workshops/guests/{guestId}/token/reset"
}
func (ResetGuestTokenEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ResetGuestTokenEndpoint) Handle(ctx context.Context, req WorkshopGuestReq) (GuestTokenResp, error) {
	id, err := parseID(req.GuestID, "Invalid guest ID")
	if err != nil {
		return GuestTokenResp{}, err
	}
	code, err := core.ResetLoginCode(ctx, id)
	if err != nil {
		return GuestTokenResp{}, err
	}
	return GuestTokenResp{Token: code}, nil
}

// optionalID parses an optional id; nil or empty is none.
func optionalID(raw *string, msg string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	id, err := parseID(*raw, msg)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
