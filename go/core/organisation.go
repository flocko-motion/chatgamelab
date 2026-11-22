package core

import (
	"context"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"

	"github.com/google/uuid"
)

// An organisation is split into sections, because who may see them differs:
// OrganisationCard, OrganisationSettings, OrganisationDetails and
// OrganisationRecord. No field is in two sections. The card is the base
// type: whatever passes the gate is a card; the other sections stay locked
// until asked for. Its members are users (ListMembers).

// The wishes a list of organisations knows.
const (
	// OrganisationFilterRecentlyRegistered narrows the list to those that
	// registered themselves in the last recentlyRegisteredDays days, among
	// those whose record the caller may read.
	OrganisationFilterRecentlyRegistered = "recentlyRegistered"
)

// recentlyRegisteredDays is how far back OrganisationFilterRecentlyRegistered
// looks: the organisations admins check after they registered.
const recentlyRegisteredDays = 14

// ListOrganisations lists the organisations whose card the caller may list,
// {List, OrganisationCard}, narrowed by filters; an unknown filter is
// refused. One query; which grants open which of its fragments is in
// gates.go.
func ListOrganisations(ctx context.Context, filters []string) ([]*OrganisationCard, error) {
	var registeredSince *time.Time
	for _, f := range filters {
		if f != OrganisationFilterRecentlyRegistered {
			return nil, refuse("unknown filter %q; known: %s", f, OrganisationFilterRecentlyRegistered)
		}
		since := time.Now().AddDate(0, 0, -recentlyRegisteredDays)
		registeredSince = &since
	}
	rows, err := store.Queries().V2Organisations(ctx, listOrganisationsGates(ctx))
	if err != nil {
		return nil, err
	}
	orgs := make([]*OrganisationCard, 0, len(rows))
	for _, r := range rows {
		org, err := organisationCardOf(ctx, r)
		if err != nil {
			return nil, err
		}
		if registeredSince != nil && !org.registeredSince(*registeredSince) {
			continue
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// ReadOrganisation opens one organisation's card, if the caller may read it:
// {Read, OrganisationCard}. Otherwise it does not exist for them
// (ErrNotFound).
func ReadOrganisation(ctx context.Context, id uuid.UUID) (*OrganisationCard, error) {
	rows, err := store.Queries().V2Organisations(ctx, readOrganisationGates(ctx, id))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return organisationCardOf(ctx, rows[0])
}

// organisationCardOf is a loaded row as the caller's card, with their rights
// on it. Whoever may read the record also gets the organisation's consent
// log, read from it here, and its members, as far as they may list them.
func organisationCardOf(ctx context.Context, r db.V2OrganisationsRow) (*OrganisationCard, error) {
	org := &OrganisationCard{
		row: organisationRow{
			id:         r.ID,
			name:       r.Name,
			createdAt:  r.CreatedAt,
			modifiedAt: r.ModifiedAt,

			promptConstraint: nullString(r.PromptConstraints),
			freeUseKeyShare:  ptrUUID(r.FreeUseApiKeyShareID),
			freeUseTier:      nullString(r.FreeUseAiQualityTier),

			operator:   nullString(r.Operator),
			street:     nullString(r.Street),
			postalCode: nullString(r.PostalCode),
			city:       nullString(r.City),

			suspended:       r.Suspended,
			suspendedAt:     nullTime(r.SuspendedAt),
			suspendedReason: nullString(r.SuspendedReason),
		},
		mayReadSettings: r.MayReadSettings,
		mayReadDetails:  r.MayReadDetails,
		mayReadRecord:   r.MayReadRecord,
		mayEditCard:     r.MayEditCard,
		mayEditSettings: r.MayEditSettings,
		mayEditDetails:  r.MayEditDetails,
		mayDelete:       r.MayDelete,
		maySuspend:      r.MaySuspend,
	}
	// the invite rights, from the gate CreateInvite checks; who may invite
	// lists the invites too (the grant table grants both together)
	org.mayInviteStaff = mayCreateInvite(ctx, r.ID, false)
	org.mayInviteHead = mayCreateInvite(ctx, r.ID, true)
	// creating its workshops, from the gate CreateWorkshop checks; listing
	// the keys given to it, from the gate its key list opens
	org.mayCreateWorkshop = mayCreateWorkshop(ctx, r.ID)
	own := ownOrganisation(ctx)
	org.mayListKeys = keyShareGates(ctx).CardOrganisation && own.Valid && own.UUID == r.ID
	if org.mayReadRecord {
		if err := org.loadConsentLog(ctx, r.CreatedBy); err != nil {
			return nil, err
		}
	}
	members, err := listMembers(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	org.members = members
	return org, nil
}

// loadConsentLog reads the organisation's consent status, and how it
// registered itself: its first consent, if its creator gave it. One an admin
// created was consented to by a head later.
func (o *OrganisationCard) loadConsentLog(ctx context.Context, creator uuid.NullUUID) error {
	status, err := consentStatus(ctx, o.row.id)
	if err != nil {
		return err
	}
	o.row.consent = status
	first, err := store.Queries().V2FirstConsent(ctx, o.row.id)
	if err != nil {
		return err
	}
	if len(first) > 0 && creator.Valid && first[0].UserID == creator.UUID {
		o.row.registration = &OrganisationRegistration{
			At:             first[0].CreatedAt,
			PersonName:     first[0].PersonName.String,
			PersonFunction: first[0].PersonFunction.String,
			Email:          first[0].Email.String,
		}
	}
	return nil
}

// registeredSince says whether the organisation registered itself since
// then, as far as the caller may read its record.
func (o *OrganisationCard) registeredSince(since time.Time) bool {
	return o.mayReadRecord && o.row.registration != nil && !o.row.registration.At.Before(since)
}

// organisationRow is an organisation's full row, loaded once. It never
// leaves the core; each section's type offers only its own fields.
type organisationRow struct {
	// card
	id         uuid.UUID
	name       string
	createdAt  time.Time
	modifiedAt time.Time

	// settings
	promptConstraint *string
	freeUseKeyShare  *uuid.UUID
	freeUseTier      *string

	// details
	operator   *string
	street     *string
	postalCode *string
	city       *string

	// record
	suspended       bool
	suspendedAt     *time.Time
	suspendedReason *string
	consent         legal.ConsentStatus
	registration    *OrganisationRegistration
}

// OrganisationCard is who an organisation is: what its members and the
// guests in its workshops get. It holds the full row and the caller's rights
// on it, and shows only the card.
type OrganisationCard struct {
	row     organisationRow
	members []*UserCard

	mayReadSettings bool
	mayReadDetails  bool
	mayReadRecord   bool
	mayEditCard     bool
	mayEditSettings bool
	mayEditDetails  bool
	mayDelete       bool
	maySuspend      bool

	mayInviteStaff    bool
	mayInviteHead     bool
	mayCreateWorkshop bool
	mayListKeys       bool
}

func (o *OrganisationCard) ID() uuid.UUID         { return o.row.id }
func (o *OrganisationCard) Name() string          { return o.row.name }
func (o *OrganisationCard) CreatedAt() time.Time  { return o.row.createdAt }
func (o *OrganisationCard) ModifiedAt() time.Time { return o.row.modifiedAt }

// Members are the organisation's members (head and staff) the
// caller may list: {List, UserCard}. None for a caller who may list none.
func (o *OrganisationCard) Members() []*UserCard { return o.members }

// MayEditCard says whether the caller may rename the organisation:
// {Edit, OrganisationCard}.
func (o *OrganisationCard) MayEditCard() bool { return o.mayEditCard }

// MayEditSettings says whether the caller may change its youth-protection
// restriction: {Edit, OrganisationSettings}.
func (o *OrganisationCard) MayEditSettings() bool { return o.mayEditSettings }

// MayEditDetails says whether the caller may confirm its details, and with
// them consent to the declaration in its name or withdraw that:
// {Edit, OrganisationDetails}.
func (o *OrganisationCard) MayEditDetails() bool { return o.mayEditDetails }

// MayDelete says whether the caller may delete the organisation for good:
// {Delete, OrganisationCard}.
func (o *OrganisationCard) MayDelete() bool { return o.mayDelete }

// MaySuspend says whether the caller may suspend the organisation and lift
// that: {Suspend, OrganisationCard}.
func (o *OrganisationCard) MaySuspend() bool { return o.maySuspend }

// MayInviteStaff says whether the caller may invite someone to work in the
// organisation: {Create, Invite}, in their own organisation or anywhere.
func (o *OrganisationCard) MayInviteStaff() bool { return o.mayInviteStaff }

// MayInviteHead says whether the caller may invite someone to head the
// organisation: {Create, HeadInvite}, in their own organisation or anywhere.
func (o *OrganisationCard) MayInviteHead() bool { return o.mayInviteHead }

// MayCreateWorkshop says whether the caller may create a workshop in the
// organisation: {Create, WorkshopCard}, in their own organisation or anywhere.
func (o *OrganisationCard) MayCreateWorkshop() bool { return o.mayCreateWorkshop }

// MayListKeys says whether the caller may list the keys given to the
// organisation: {List, ApiKeyShare, SharedWithOwnOrganisation}, in their own.
func (o *OrganisationCard) MayListKeys() bool { return o.mayListKeys }

// AsOrganisationSettings unlocks how the organisation runs: a flag check, no
// query.
func (o *OrganisationCard) AsOrganisationSettings() (*OrganisationSettings, error) {
	if !o.mayReadSettings {
		return nil, ErrLocked
	}
	return &OrganisationSettings{card: o}, nil
}

// AsOrganisationDetails unlocks who stands behind the organisation: a flag
// check, no query.
func (o *OrganisationCard) AsOrganisationDetails() (*OrganisationDetails, error) {
	if !o.mayReadDetails {
		return nil, ErrLocked
	}
	return &OrganisationDetails{card: o}, nil
}

// AsOrganisationRecord unlocks the organisation's standing: a flag check, no
// query.
func (o *OrganisationCard) AsOrganisationRecord() (*OrganisationRecord, error) {
	if !o.mayReadRecord {
		return nil, ErrLocked
	}
	return &OrganisationRecord{card: o}, nil
}

// OrganisationSettings is how an organisation runs.
type OrganisationSettings struct{ card *OrganisationCard }

// PromptConstraint is its youth-protection restriction; nil: none, the
// site's rule applies.
func (s *OrganisationSettings) PromptConstraint() *string { return s.card.row.promptConstraint }

// FreeUseKeyShare is the key share its free use spends from; nil: none.
func (s *OrganisationSettings) FreeUseKeyShare() *uuid.UUID { return s.card.row.freeUseKeyShare }

// FreeUseTier is the AI quality tier of its free use (high, medium, low);
// nil: the server's default.
func (s *OrganisationSettings) FreeUseTier() *string { return s.card.row.freeUseTier }

// OrganisationDetails is who stands behind an organisation: its operator
// (Träger) and address, each nil while not given.
type OrganisationDetails struct{ card *OrganisationCard }

func (d *OrganisationDetails) Operator() *string   { return d.card.row.operator }
func (d *OrganisationDetails) Street() *string     { return d.card.row.street }
func (d *OrganisationDetails) PostalCode() *string { return d.card.row.postalCode }
func (d *OrganisationDetails) City() *string       { return d.card.row.city }

// OrganisationRecord is an organisation's standing.
type OrganisationRecord struct{ card *OrganisationCard }

// Suspended says its heads, staff and guests are refused: an admin
// suspended it, or it has no consent to the current declaration.
func (r *OrganisationRecord) Suspended() bool { return r.card.row.suspended }

// SuspendedAt is when an admin suspended it; nil while they have not.
func (r *OrganisationRecord) SuspendedAt() *time.Time { return r.card.row.suspendedAt }

// SuspendedReason is the admin's note on their suspension; nil: none.
func (r *OrganisationRecord) SuspendedReason() *string { return r.card.row.suspendedReason }

// Consent is whether its newest entry in the consent log covers the current
// declaration.
func (r *OrganisationRecord) Consent() legal.ConsentStatus { return r.card.row.consent }

// Registration is how it registered itself; nil for one an admin created.
func (r *OrganisationRecord) Registration() *OrganisationRegistration {
	return r.card.row.registration
}

// OrganisationRegistration is how an organisation registered itself: when,
// and the person who signed for it, from its first consent.
type OrganisationRegistration struct {
	At             time.Time
	PersonName     string
	PersonFunction string
	Email          string
}
