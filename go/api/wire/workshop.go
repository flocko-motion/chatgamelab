package wire

import (
	"time"

	"cgl/core"
	"cgl/obj"

	"github.com/google/uuid"
)

// WorkshopView is a workshop as the caller sees it: the card always; its
// settings, participants and invites only when the caller may see them,
// which hasSettings, hasParticipants and hasInvites say. Field names follow
// v1's workshop, so clients read the same keys.
type WorkshopView struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Meta        WorkshopMeta `json:"meta"`
	Institution NamedRef     `json:"institution"`

	Active                     bool    `json:"active"`
	IsPaused                   bool    `json:"isPaused"`
	Public                     bool    `json:"public"`
	PublicSlug                 *string `json:"publicSlug,omitempty"`
	AiQualityTier              *string `json:"aiQualityTier,omitempty"`
	ShowPublicGames            bool    `json:"showPublicGames"`
	ShowOtherParticipantsGames bool    `json:"showOtherParticipantsGames"`
	AllowGameSharing           bool    `json:"allowGameSharing"`

	// The caller's rights on the workshop, by section, and on the whole
	// workshop.
	MayEditCard           bool `json:"mayEditCard"`
	MayEditSettings       bool `json:"mayEditSettings"`
	MayDelete             bool `json:"mayDelete"`
	MayManageParticipants bool `json:"mayManageParticipants"`

	// HasSettings: the caller sees how the workshop is set up.
	HasSettings          bool                     `json:"hasSettings"`
	PromptConstraints    *string                  `json:"promptConstraints,omitempty"`
	DefaultApiKeyShareID *uuid.UUID               `json:"defaultApiKeyShareId,omitempty"`
	PublicDescription    *string                  `json:"publicDescription,omitempty"`
	PublicLinks          []obj.PublicWorkshopLink `json:"publicLinks,omitempty"`

	// HasParticipants: the caller sees who is in the workshop.
	HasParticipants bool                      `json:"hasParticipants"`
	Participants    []WorkshopParticipantView `json:"participants"`

	// Invites are its invites the caller may see, with their links.
	Invites []InviteView `json:"invites"`
}

// WorkshopMeta is when a workshop was made and last changed.
type WorkshopMeta struct {
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// WorkshopParticipantView is someone in a workshop. Permanent: a guest of
// it, or its creator; others only work in it for now.
type WorkshopParticipantView struct {
	ID         uuid.UUID       `json:"id"`
	WorkshopID uuid.UUID       `json:"workshopId"`
	Name       string          `json:"name"`
	Role       string          `json:"role"`
	Meta       ParticipantMeta `json:"meta"`
	GamesCount int             `json:"gamesCount"`
	Permanent  bool            `json:"permanent"`
}

// ParticipantMeta is when someone joined a workshop.
type ParticipantMeta struct {
	CreatedAt time.Time `json:"createdAt"`
}

// InviteView is an invite as those who may list it see it: with its link.
type InviteView struct {
	ID            uuid.UUID  `json:"id"`
	Meta          InviteMeta `json:"meta"`
	InstitutionID uuid.UUID  `json:"institutionId"`
	Role          string     `json:"role"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InvitedUserID *uuid.UUID `json:"invitedUserId,omitempty"`
	InvitedEmail  *string    `json:"invitedEmail,omitempty"`
	InviteToken   *string    `json:"inviteToken,omitempty"`
	MaxUses       *int32     `json:"maxUses,omitempty"`
	UsesCount     int32      `json:"usesCount"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Status        string     `json:"status"`
	AcceptedAt    *time.Time `json:"acceptedAt,omitempty"`
	AcceptedBy    *uuid.UUID `json:"acceptedBy,omitempty"`
}

// InviteMeta is when an invite was made and last changed.
type InviteMeta struct {
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// WorkshopViewOf converts one workshop card, passing a loading error
// through:
//
//	return wire.WorkshopViewOf(core.ReadWorkshop(ctx, id))
func WorkshopViewOf(w *core.WorkshopCard, err error) (WorkshopView, error) {
	if err != nil {
		return WorkshopView{}, err
	}
	return workshopView(w), nil
}

// WorkshopViewsOf converts a list of workshop cards, passing a loading
// error through.
func WorkshopViewsOf(ws []*core.WorkshopCard, err error) ([]WorkshopView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]WorkshopView, len(ws))
	for i, w := range ws {
		views[i] = workshopView(w)
	}
	return views, nil
}

// workshopView converts a workshop card, with every section the caller may
// see.
func workshopView(w *core.WorkshopCard) WorkshopView {
	v := WorkshopView{
		ID:                         w.ID(),
		Name:                       w.Name(),
		Meta:                       WorkshopMeta{CreatedAt: w.CreatedAt(), ModifiedAt: w.ModifiedAt()},
		Institution:                NamedRef{ID: w.OrganisationID(), Name: w.OrganisationName()},
		Active:                     w.Active(),
		IsPaused:                   w.Paused(),
		Public:                     w.Public(),
		PublicSlug:                 w.PublicSlug(),
		AiQualityTier:              w.AiQualityTier(),
		ShowPublicGames:            w.ShowsPublicGames(),
		ShowOtherParticipantsGames: w.ShowsOthersGames(),
		AllowGameSharing:           w.AllowsGameSharing(),
		MayEditCard:                w.MayEditCard(),
		MayEditSettings:            w.MayEditSettings(),
		MayDelete:                  w.MayDelete(),
		MayManageParticipants:      w.MayManageParticipants(),
		HasParticipants:            w.HasParticipants(),
		Participants:               []WorkshopParticipantView{},
		Invites:                    []InviteView{},
	}
	if s, err := w.AsWorkshopSettings(); err == nil {
		v.HasSettings = true
		v.PromptConstraints = s.PromptConstraint()
		v.DefaultApiKeyShareID = s.KeyShare()
		v.PublicDescription = s.PublicDescription()
		v.PublicLinks = s.PublicLinks()
	}
	for _, p := range w.Participants() {
		v.Participants = append(v.Participants, participantView(w.ID(), p))
	}
	for _, inv := range w.Invites() {
		v.Invites = append(v.Invites, inviteView(inv))
	}
	return v
}

// inviteView converts an invite.
func inviteView(inv core.Invite) InviteView {
	return InviteView{
		ID:            inv.ID,
		Meta:          InviteMeta{CreatedAt: inv.CreatedAt, ModifiedAt: inv.ModifiedAt},
		InstitutionID: inv.OrganisationID,
		Role:          inv.Role,
		WorkshopID:    inv.WorkshopID,
		InvitedUserID: inv.InvitedUserID,
		InvitedEmail:  inv.InvitedEmail,
		InviteToken:   inv.Token,
		MaxUses:       inv.MaxUses,
		UsesCount:     inv.UsesCount,
		ExpiresAt:     inv.ExpiresAt,
		Status:        inv.Status,
		AcceptedAt:    inv.AcceptedAt,
		AcceptedBy:    inv.AcceptedBy,
	}
}

func participantView(workshop uuid.UUID, p core.WorkshopParticipant) WorkshopParticipantView {
	return WorkshopParticipantView{
		ID: p.ID, WorkshopID: workshop, Name: p.Name, Role: p.Role, Meta: ParticipantMeta{CreatedAt: p.JoinedAt},
		GamesCount: p.GamesCount, Permanent: p.Permanent,
	}
}
