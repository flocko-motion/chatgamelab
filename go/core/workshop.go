package core

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// A workshop is split into sections, because who may see them differs:
// WorkshopCard, WorkshopSettings and WorkshopParticipants. Its invites are
// listed with it for those who may list them. The card is the base type:
// whatever passes the gate is a card; the other sections stay locked until
// asked for.

// WorkshopQuery is what a list of workshops is narrowed and ordered by.
type WorkshopQuery struct {
	Organisation *uuid.UUID // only this organisation's
	Search       string     // a name containing it, ignoring case
	ActiveOnly   bool       // only those that run
	SortBy       string     // WorkshopSortName, WorkshopSortCreatedAt (the default), WorkshopSortParticipants
	SortDir      string     // "asc" or "desc"; the default is ascending, but newest first for the default order
}

// The orders a list of workshops knows.
const (
	WorkshopSortName         = "name"
	WorkshopSortCreatedAt    = "createdAt"
	WorkshopSortParticipants = "participantCount"
)

// ListWorkshops lists the workshops whose card the caller may list,
// {List, WorkshopCard}, narrowed and ordered by q; an unknown order is
// refused. One query; which grants open which of its fragments is in
// gates.go.
func ListWorkshops(ctx context.Context, q WorkshopQuery) ([]*WorkshopCard, error) {
	p := listWorkshopsGates(ctx)
	p.Organisation = nullUUID(q.Organisation)
	if s := strings.TrimSpace(q.Search); s != "" {
		p.Search = sql.NullString{String: s, Valid: true}
	}
	p.ActiveOnly = q.ActiveOnly
	less, err := workshopOrder(q.SortBy, q.SortDir)
	if err != nil {
		return nil, err
	}
	rows, err := store.Queries().V2Workshops(ctx, p)
	if err != nil {
		return nil, err
	}
	workshops := make([]*WorkshopCard, len(rows))
	for i, r := range rows {
		if workshops[i], err = workshopCardOf(ctx, r); err != nil {
			return nil, err
		}
	}
	if less != nil {
		sort.SliceStable(workshops, func(i, j int) bool { return less(workshops[i], workshops[j]) })
	}
	return workshops, nil
}

// workshopOrder is how to order a list by sortBy and sortDir; nil keeps the
// query's order, newest first.
func workshopOrder(sortBy, sortDir string) (func(a, b *WorkshopCard) bool, error) {
	desc := sortDir == "desc"
	if sortDir != "" && sortDir != "asc" && !desc {
		return nil, refuse("unknown sort direction %q; known: asc, desc", sortDir)
	}
	switch sortBy {
	case "":
		return nil, nil
	case WorkshopSortName:
		return func(a, b *WorkshopCard) bool { return (a.row.name < b.row.name) != desc }, nil
	case WorkshopSortCreatedAt:
		if sortDir == "" {
			desc = true // newest first, as without an order
		}
		return func(a, b *WorkshopCard) bool { return a.row.createdAt.Before(b.row.createdAt) != desc }, nil
	case WorkshopSortParticipants:
		return func(a, b *WorkshopCard) bool { return (len(a.participants) < len(b.participants)) != desc }, nil
	}
	return nil, refuse("unknown order %q; known: %s, %s, %s", sortBy, WorkshopSortName, WorkshopSortCreatedAt, WorkshopSortParticipants)
}

// ReadWorkshop opens one workshop's card, if the caller may read it:
// {Read, WorkshopCard}. Otherwise it does not exist for them (ErrNotFound).
func ReadWorkshop(ctx context.Context, id uuid.UUID) (*WorkshopCard, error) {
	rows, err := store.Queries().V2Workshops(ctx, readWorkshopGates(ctx, id))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return workshopCardOf(ctx, rows[0])
}

// workshopCardOf is a loaded row as the caller's card, with their rights on
// it, and its participants and invites as far as the caller may list them.
func workshopCardOf(ctx context.Context, r db.V2WorkshopsRow) (*WorkshopCard, error) {
	w := &WorkshopCard{
		row: workshopRow{
			id:               r.ID,
			name:             r.Name,
			organisationID:   r.InstitutionID,
			organisationName: r.OrganisationName,
			createdAt:        r.CreatedAt,
			modifiedAt:       r.ModifiedAt,

			active:            r.Active,
			paused:            r.IsPaused,
			public:            r.Public,
			publicSlug:        nullString(r.PublicSlug),
			aiQualityTier:     nullString(r.AiQualityTier),
			showsPublicGames:  r.ShowPublicGames,
			showsOthersGames:  r.ShowOtherParticipantsGames,
			allowsGameSharing: r.AllowGameSharing,

			promptConstraint:  nullString(r.PromptConstraints),
			keyShare:          ptrUUID(r.DefaultApiKeyShareID),
			publicDescription: nullString(r.PublicDescription),
			publicLinks:       unmarshalPublicLinks(r.PublicLinks),
		},
		mayReadSettings:       r.MayReadSettings,
		mayListParticipants:   r.MayListParticipants,
		mayEditCard:           r.MayEditCard,
		mayEditSettings:       r.MayEditSettings,
		mayDelete:             r.MayDelete,
		mayManageParticipants: r.MayManageParticipants,
	}
	if w.mayListParticipants {
		rows, err := store.Queries().V2GetWorkshopParticipants(ctx, uuid.NullUUID{UUID: r.ID, Valid: true})
		if err != nil {
			return nil, err
		}
		for _, p := range rows {
			w.participants = append(w.participants, WorkshopParticipant{
				ID: p.ID, Name: p.Name.String, Role: p.Role, JoinedAt: p.JoinedAt,
				GamesCount: int(p.GamesCount), Permanent: p.Permanent,
			})
		}
	}
	invites, err := ListInvites(ctx, InviteFilter{Workshop: &w.row.id})
	if err != nil {
		return nil, err
	}
	for _, inv := range invites {
		w.invites = append(w.invites, inv.Invite)
	}
	return w, nil
}

// workshopRow is a workshop's full row, loaded once. It never leaves the
// core; each section's type offers only its own fields.
type workshopRow struct {
	// card
	id                uuid.UUID
	name              string
	organisationID    uuid.UUID
	organisationName  string
	createdAt         time.Time
	modifiedAt        time.Time
	active            bool
	paused            bool
	public            bool
	publicSlug        *string
	aiQualityTier     *string
	showsPublicGames  bool
	showsOthersGames  bool
	allowsGameSharing bool

	// settings
	promptConstraint  *string
	keyShare          *uuid.UUID
	publicDescription *string
	publicLinks       []obj.PublicWorkshopLink
}

// WorkshopCard is what a workshop is to those in it. It holds the full row,
// the caller's rights on it, and its participants and invites as far as the
// caller may list them; it shows only the card.
type WorkshopCard struct {
	row          workshopRow
	participants []WorkshopParticipant
	invites      []Invite

	mayReadSettings       bool
	mayListParticipants   bool
	mayEditCard           bool
	mayEditSettings       bool
	mayDelete             bool
	mayManageParticipants bool
}

func (w *WorkshopCard) ID() uuid.UUID             { return w.row.id }
func (w *WorkshopCard) Name() string              { return w.row.name }
func (w *WorkshopCard) OrganisationID() uuid.UUID { return w.row.organisationID }
func (w *WorkshopCard) OrganisationName() string  { return w.row.organisationName }
func (w *WorkshopCard) CreatedAt() time.Time      { return w.row.createdAt }
func (w *WorkshopCard) ModifiedAt() time.Time     { return w.row.modifiedAt }
func (w *WorkshopCard) Active() bool              { return w.row.active }
func (w *WorkshopCard) Paused() bool              { return w.row.paused }
func (w *WorkshopCard) Public() bool              { return w.row.public }
func (w *WorkshopCard) PublicSlug() *string       { return w.row.publicSlug }
func (w *WorkshopCard) ShowsPublicGames() bool    { return w.row.showsPublicGames }
func (w *WorkshopCard) ShowsOthersGames() bool    { return w.row.showsOthersGames }
func (w *WorkshopCard) AllowsGameSharing() bool   { return w.row.allowsGameSharing }

// AiQualityTier is the AI quality tier it plays at (high, medium, low); nil:
// the server's default.
func (w *WorkshopCard) AiQualityTier() *string { return w.row.aiQualityTier }

// MayEditCard says whether the caller may change what the workshop is:
// {Edit, WorkshopCard}.
func (w *WorkshopCard) MayEditCard() bool { return w.mayEditCard }

// MayEditSettings says whether the caller may change how it is set up:
// {Edit, WorkshopSettings}.
func (w *WorkshopCard) MayEditSettings() bool { return w.mayEditSettings }

// MayDelete says whether the caller may delete it: {Delete, WorkshopCard}.
func (w *WorkshopCard) MayDelete() bool { return w.mayDelete }

// MayManageParticipants says whether the caller may take people out of it:
// {Manage, WorkshopParticipants}.
func (w *WorkshopCard) MayManageParticipants() bool { return w.mayManageParticipants }

// Participants are who is in it, for a caller who may list them:
// {List, WorkshopParticipants}; nil otherwise, which HasParticipants tells
// from an empty workshop.
func (w *WorkshopCard) Participants() []WorkshopParticipant { return w.participants }
func (w *WorkshopCard) HasParticipants() bool               { return w.mayListParticipants }

// Invites are its invites the caller may list (ListInvites).
func (w *WorkshopCard) Invites() []Invite { return w.invites }

// AsWorkshopSettings unlocks how the workshop is set up: a flag check, no
// query.
func (w *WorkshopCard) AsWorkshopSettings() (*WorkshopSettings, error) {
	if !w.mayReadSettings {
		return nil, ErrLocked
	}
	return &WorkshopSettings{card: w}, nil
}

// WorkshopSettings is how a workshop is set up.
type WorkshopSettings struct{ card *WorkshopCard }

// PromptConstraint is its youth-protection restriction; nil: none.
func (s *WorkshopSettings) PromptConstraint() *string { return s.card.row.promptConstraint }

// KeyShare is the key share it plays with; nil: none.
func (s *WorkshopSettings) KeyShare() *uuid.UUID { return s.card.row.keyShare }

// PublicDescription is its public page's text; nil: none.
func (s *WorkshopSettings) PublicDescription() *string { return s.card.row.publicDescription }

// PublicLinks are the further reading on its public page.
func (s *WorkshopSettings) PublicLinks() []obj.PublicWorkshopLink { return s.card.row.publicLinks }

// WorkshopParticipant is someone in a workshop: a guest of it, its creator,
// or a member of its organisation working in it.
type WorkshopParticipant struct {
	ID         uuid.UUID
	Name       string
	Role       string
	JoinedAt   time.Time
	GamesCount int
	Permanent  bool // a guest of it, or its creator; others only work in it for now
}

// Invite is an invite into an organisation or a workshop, as those who may
// see it get it: with its link.
type Invite struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	ModifiedAt     time.Time
	OrganisationID uuid.UUID
	Role           string
	WorkshopID     *uuid.UUID
	InvitedUserID  *uuid.UUID
	InvitedEmail   *string
	Token          *string
	MaxUses        *int32
	UsesCount      int32
	ExpiresAt      *time.Time
	Status         string
	AcceptedAt     *time.Time
	AcceptedBy     *uuid.UUID
}

// inviteOf is an invite's row as the core hands it out.
func inviteOf(r db.UserRoleInvite) Invite {
	inv := Invite{
		ID: r.ID, CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
		OrganisationID: r.InstitutionID, Role: r.Role,
		WorkshopID: ptrUUID(r.WorkshopID), InvitedUserID: ptrUUID(r.InvitedUserID),
		InvitedEmail: nullString(r.InvitedEmail), Token: nullString(r.InviteToken),
		UsesCount: r.UsesCount, ExpiresAt: nullTime(r.ExpiresAt), Status: r.Status,
		AcceptedAt: nullTime(r.AcceptedAt), AcceptedBy: ptrUUID(r.AcceptedBy),
	}
	if r.MaxUses.Valid {
		inv.MaxUses = &r.MaxUses.Int32
	}
	return inv
}
