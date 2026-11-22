package core

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/events"
	"cgl/legal"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// maxWorkshopName is the longest a workshop's name may be, in characters.
const maxWorkshopName = 150

// aiQualityTiers are the tiers a workshop may play at; "" is the server's
// default.
var aiQualityTiers = []string{"", "high", "medium", "low"}

// WorkshopDraft is a new workshop: its organisation, name, and whether it
// runs and has a public page. Without an organisation it is the caller's.
type WorkshopDraft struct {
	Organisation *uuid.UUID
	Name         string
	Active       bool
	Public       bool
}

// CreateWorkshop creates a workshop and returns it. It takes {Create,
// WorkshopCard}: in the caller's organisation, or, for the admin, in any
// they name. It gets a free address for its public page.
func CreateWorkshop(ctx context.Context, d WorkshopDraft) (*WorkshopCard, error) {
	org := d.Organisation
	if org == nil {
		org = from(ctx).Organisation()
	}
	if org == nil {
		return nil, refuse("institutionId is required")
	}
	if !mayCreateWorkshop(ctx, *org) {
		return nil, forbidden("you may not create a workshop in this organisation")
	}
	if err := requireField("name", &d.Name, maxWorkshopName); err != nil {
		return nil, err
	}
	id := uuid.New()
	if err := checkWorkshopNameFree(ctx, *org, id, d.Name); err != nil {
		return nil, err
	}
	slug, err := newPublicSlug(ctx, d.Name)
	if err != nil {
		return nil, err
	}
	err = store.Queries().V2CreateWorkshop(ctx, db.V2CreateWorkshopParams{
		ID:            id,
		Creator:       account(ctx).UUID,
		Name:          d.Name,
		InstitutionID: *org,
		Active:        d.Active,
		Public:        d.Public,
		PublicSlug:    slug,
	})
	if err != nil {
		return nil, err
	}
	log.Info("workshop created", "workshop_id", id, "institution_id", *org, "by", account(ctx).UUID)
	return ReadWorkshop(ctx, id)
}

// WorkshopChanges are the changes to a workshop the caller asks for; nil
// leaves a field as it is.
type WorkshopChanges struct {
	// card
	Name              *string
	Active            *bool
	Paused            *bool
	Public            *bool
	PublicSlug        *string // its public page's address
	AiQualityTier     *string // high, medium, low; "": the server's default
	ShowsPublicGames  *bool
	ShowsOthersGames  *bool
	AllowsGameSharing *bool

	// settings
	PromptConstraint  *string // "": none
	PublicDescription *string // "": none
	PublicLinks       *[]obj.PublicWorkshopLink
}

// UpdateWorkshop changes a workshop as the caller asks and returns it as
// stored.
//
// A workshop the caller may not read does not exist for them (ErrNotFound).
// Each section the request changes must be one the caller may edit: {Edit,
// WorkshopCard}, {Edit, WorkshopSettings}; otherwise the whole request is
// refused and nothing is written. A request that changes nothing writes
// nothing. Switching its public page off ends the page's share links.
func UpdateWorkshop(ctx context.Context, id uuid.UUID, ch WorkshopChanges) (*WorkshopCard, error) {
	w, err := ReadWorkshop(ctx, id)
	if err != nil {
		return nil, err
	}
	next := w.row
	cardChanged, err := ch.applyCard(&next)
	if err != nil {
		return nil, err
	}
	settingsChanged, err := ch.applySettings(&next)
	if err != nil {
		return nil, err
	}
	switch {
	case cardChanged && !w.MayEditCard():
		return nil, forbidden("you may not change this workshop")
	case settingsChanged && !w.MayEditSettings():
		return nil, forbidden("you may not change this workshop's settings")
	case !cardChanged && !settingsChanged:
		return w, nil
	}
	if next.name != w.row.name {
		if err := checkWorkshopNameFree(ctx, w.row.organisationID, id, next.name); err != nil {
			return nil, err
		}
	}
	if next.publicSlug != nil && (w.row.publicSlug == nil || *next.publicSlug != *w.row.publicSlug) {
		if err := checkPublicSlugFree(ctx, *next.publicSlug); err != nil {
			return nil, err
		}
	}
	links, err := marshalPublicLinks(next.publicLinks)
	if err != nil {
		return nil, err
	}
	err = store.Queries().V2UpdateWorkshop(ctx, db.V2UpdateWorkshopParams{
		ID:                         id,
		Name:                       next.name,
		Active:                     next.active,
		IsPaused:                   next.paused,
		Public:                     next.public,
		PublicSlug:                 deref(next.publicSlug),
		AiQualityTier:              sqlString(next.aiQualityTier),
		ShowPublicGames:            next.showsPublicGames,
		ShowOtherParticipantsGames: next.showsOthersGames,
		AllowGameSharing:           next.allowsGameSharing,
		PromptConstraints:          sqlString(next.promptConstraint),
		PublicDescription:          sqlString(next.publicDescription),
		PublicLinks:                links,
		ModifiedBy:                 account(ctx).UUID,
	})
	if err := slugTakenOr(err); err != nil {
		return nil, err
	}
	if w.row.public && !next.public {
		if err := endPublicPageLinks(ctx, &id, nil); err != nil {
			log.Warn("failed to end a workshop's public page links", "workshop_id", id, "error", err)
		}
	}
	events.GetBroker().PublishWorkshopUpdated(id)
	return ReadWorkshop(ctx, id)
}

// applyCard writes the card's changes into row, validated, and says whether
// any differs.
func (ch WorkshopChanges) applyCard(row *workshopRow) (bool, error) {
	changed := false
	if ch.Name != nil {
		name := *ch.Name
		if err := requireField("name", &name, maxWorkshopName); err != nil {
			return false, err
		}
		set(&row.name, &name, &changed)
	}
	set(&row.active, ch.Active, &changed)
	set(&row.paused, ch.Paused, &changed)
	set(&row.public, ch.Public, &changed)
	set(&row.showsPublicGames, ch.ShowsPublicGames, &changed)
	set(&row.showsOthersGames, ch.ShowsOthersGames, &changed)
	set(&row.allowsGameSharing, ch.AllowsGameSharing, &changed)
	if ch.AiQualityTier != nil && !slices.Contains(aiQualityTiers, *ch.AiQualityTier) {
		return false, refuse("unknown aiQualityTier %q; known: high, medium, low", *ch.AiQualityTier)
	}
	setOptional(&row.aiQualityTier, ch.AiQualityTier, &changed)
	if ch.PublicSlug != nil {
		slug := normalizePublicSlug(*ch.PublicSlug)
		if row.publicSlug == nil || slug != *row.publicSlug {
			if err := validatePublicSlug(slug); err != nil {
				return false, fromV1(err)
			}
			row.publicSlug, changed = &slug, true
		}
	}
	return changed, nil
}

// applySettings writes the settings' changes into row, validated, and says
// whether any differs.
func (ch WorkshopChanges) applySettings(row *workshopRow) (bool, error) {
	changed := false
	if ch.PromptConstraint != nil {
		constraint := strings.TrimSpace(*ch.PromptConstraint)
		if utf8.RuneCountInString(constraint) > legal.MaxPromptConstraintLength {
			return false, refuse("promptConstraints has at most %d characters", legal.MaxPromptConstraintLength)
		}
		setOptional(&row.promptConstraint, &constraint, &changed)
	}
	if ch.PublicDescription != nil {
		description := strings.TrimSpace(*ch.PublicDescription)
		if utf8.RuneCountInString(description) > PublicDescriptionMaxLength {
			return false, badRequest(obj.ErrCodePublicDescriptionTooLong, "the description has at most %d characters", PublicDescriptionMaxLength)
		}
		setOptional(&row.publicDescription, &description, &changed)
	}
	if ch.PublicLinks != nil {
		links, err := validatePublicLinks(*ch.PublicLinks)
		if err != nil {
			return false, fromV1(err)
		}
		if !slices.Equal(links, row.publicLinks) {
			row.publicLinks, changed = links, true
		}
	}
	return changed, nil
}

// DeleteWorkshop deletes a workshop with its guests; its other games stay
// with their creators. A workshop the caller may not read does not exist for
// them (ErrNotFound). It takes {Delete, WorkshopCard}.
func DeleteWorkshop(ctx context.Context, id uuid.UUID) error {
	w, err := ReadWorkshop(ctx, id)
	if err != nil {
		return err
	}
	if !w.MayDelete() {
		return forbidden("you may not delete this workshop")
	}
	if err := deleteWorkshop(ctx, id); err != nil {
		return err
	}
	log.Info("workshop deleted", "workshop_id", id, "by", account(ctx).UUID)
	return nil
}

// RemoveFromWorkshop takes someone out of a workshop: a guest of it loses
// their place in it, a member working in it leaves it. It takes {Manage,
// WorkshopParticipants}; a workshop or user the caller may not read does not
// exist for them (ErrNotFound).
func RemoveFromWorkshop(ctx context.Context, id, user uuid.UUID) error {
	_, member, err := workshopAndUser(ctx, id, user)
	if err != nil {
		return err
	}
	if member.row.workshopID == nil || *member.row.workshopID != id {
		return refuse("the user is not in this workshop")
	}
	q := store.Queries()
	if err := q.V2ClearUserActiveWorkshop(ctx, user); err != nil {
		return err
	}
	if err := q.V2DeleteUserGuestRole(ctx, db.V2DeleteUserGuestRoleParams{
		UserID: user, WorkshopID: uuid.NullUUID{UUID: id, Valid: true},
	}); err != nil {
		return err
	}
	events.GetBroker().PublishMembersUpdated(id)
	return nil
}

// workshopAndUser loads workshop id and user, for a caller who may manage
// the workshop's participants.
func workshopAndUser(ctx context.Context, id, user uuid.UUID) (*WorkshopCard, *UserCard, error) {
	w, err := ReadWorkshop(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !w.MayManageParticipants() {
		return nil, nil, forbidden("you may not manage who is in this workshop")
	}
	member, err := ReadUser(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return w, member, nil
}

// EnterWorkshop makes workshop id the one the caller works in, by its id:
// {Enter, WorkshopCard, OfOwnOrganisation}, for a head or staff any running
// workshop of their organisation. Anyone else takes part through the
// workshop's invite (AcceptInvite).
func EnterWorkshop(ctx context.Context, id uuid.UUID) error {
	return enterWorkshop(ctx, id, false)
}

// enterWorkshop enters workshop id, through an invite the caller accepts or
// by its id: through an invite, {Enter, WorkshopCard, ViaInvite} holds too.
func enterWorkshop(ctx context.Context, id uuid.UUID, viaInvite bool) error {
	w, err := store.Queries().V2GetWorkshopByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !mayEnterWorkshop(ctx, w.InstitutionID, w.Active, viaInvite) {
		return forbidden("you may not enter this workshop")
	}
	if err := store.Queries().V2SetUserActiveWorkshop(ctx, db.V2SetUserActiveWorkshopParams{
		UserID: account(ctx).UUID, ActiveWorkshopID: uuid.NullUUID{UUID: id, Valid: true},
	}); err != nil {
		return err
	}
	events.GetBroker().PublishMembersUpdated(id)
	return nil
}

// LeaveWorkshop takes the caller out of workshop mode: {Leave,
// WorkshopCard, Current}, head, staff and individual. A guest never leaves
// their workshop: the model relies on it.
func LeaveWorkshop(ctx context.Context) error {
	current := from(ctx).CurrentWorkshop()
	if current == nil {
		return nil // nowhere to leave
	}
	if !granted(ctx, rules.Leave, rules.WorkshopCard, rules.Current) {
		return forbidden("a guest stays in their workshop")
	}
	if err := store.Queries().V2ClearUserActiveWorkshop(ctx, account(ctx).UUID); err != nil {
		return err
	}
	events.GetBroker().PublishMembersUpdated(*current)
	return nil
}

// ResetLoginCode gives a guest a new login code, which ends the old one and
// every link carrying it and the sessions it signed in, and returns it. It takes {Edit, UserSecrets}: a
// guest of the caller's organisation. A user the caller may not read does
// not exist for them (ErrNotFound); a guest without a code has none to
// replace (ErrNotFound).
func ResetLoginCode(ctx context.Context, user uuid.UUID) (string, error) {
	guest, err := ReadUser(ctx, user)
	if err != nil {
		return "", err
	}
	if guest.row.loginToken == nil {
		return "", ErrNotFound // an account without a code has none to replace
	}
	if !mayResetLoginCode(ctx, guest) {
		return "", forbidden("you may not give this user a new login code")
	}
	code, err := newGuestToken(ctx)
	if err != nil {
		return "", err
	}
	n, err := store.Queries().V2UpdateGuestToken(ctx, db.V2UpdateGuestTokenParams{
		ID: user, GuestToken: sqlString(&code), ModifiedBy: account(ctx),
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", ErrNotFound
	}
	// the old code signed sessions in; they end with it
	if err := schemafapi.RevokeSubject(ctx, user.String()); err != nil {
		return "", err
	}
	return code, nil
}

// checkWorkshopNameFree refuses a name another workshop of the organisation
// has.
func checkWorkshopNameFree(ctx context.Context, org, id uuid.UUID, name string) error {
	taken, err := store.Queries().V2WorkshopNameTaken(ctx, db.V2WorkshopNameTakenParams{
		InstitutionID: org, Name: name, ID: id,
	})
	if err != nil {
		return err
	}
	if taken {
		return conflict(obj.ErrCodeConflict, "the organisation has a workshop with this name already")
	}
	return nil
}

// checkPublicSlugFree refuses a public page address another workshop has.
func checkPublicSlugFree(ctx context.Context, slug string) error {
	taken, err := publicSlugExists(ctx, slug)
	if err != nil {
		return err
	}
	if taken {
		return errPublicSlugTaken
	}
	return nil
}

var errPublicSlugTaken = conflict(obj.ErrCodePublicSlugTaken, "another workshop has this public page address")

// slugTakenOr is err, read as a taken public page address where the database
// refused one another workshop took meanwhile.
func slugTakenOr(err error) error {
	if nameTakenOr(err) == errOrganisationNameTaken {
		return errPublicSlugTaken
	}
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
