package core

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/events"
	"cgl/functional"
	"cgl/functional/wordtoken"
	"cgl/log"
	"cgl/obj"
	"cgl/tokenlock"

	ung "github.com/dillonstreator/go-unique-name-generator"
	"github.com/dillonstreator/go-unique-name-generator/dictionaries"
	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// An invite brings someone into an organisation or a workshop. A targeted
// invite is for one account (by id or email): its person accepts or declines
// it. An open invite is a link into a workshop: whoever follows it becomes a
// guest there, or, already a member of an organisation, works in it. An
// invite to become a head is a HeadInvite to the rules, every other one an
// Invite.

// InviteCard is an invite as the caller sees it, with the names of its
// organisation and workshop and the caller's rights on it.
type InviteCard struct {
	Invite
	OrganisationName  string
	WorkshopName      *string
	AddressedToCaller bool

	mayDelete bool
	mayUse    bool
}

// MayRevoke says whether the caller may revoke the invite: {Delete, Invite}
// or {Delete, HeadInvite}.
func (c *InviteCard) MayRevoke() bool { return c.mayDelete }

// MayAnswer says whether the caller may accept or decline it: {Use, Invite}
// or {Use, HeadInvite}.
func (c *InviteCard) MayAnswer() bool { return c.mayUse }

// open says whether it is a link anyone may follow.
func (c *InviteCard) open() bool { return c.InvitedUserID == nil && c.InvitedEmail == nil }

// InviteFilter narrows a list of invites.
type InviteFilter struct {
	Organisation *uuid.UUID // only this organisation's
	Workshop     *uuid.UUID // only this workshop's
	Mine         bool       // only the pending ones addressed to the caller
}

// ListInvites lists the invites the caller may list, {List, Invite} and
// {List, HeadInvite}, narrowed by f.
func ListInvites(ctx context.Context, f InviteFilter) ([]*InviteCard, error) {
	p, err := listInvitesGates(ctx)
	if err != nil {
		return nil, err
	}
	p.Organisation = nullUUID(f.Organisation)
	p.Workshop = nullUUID(f.Workshop)
	p.AddressedOnly, p.PendingOnly = f.Mine, f.Mine
	return loadInvites(ctx, p)
}

// ReadInvite opens one invite by its id, if the caller may read it: {Read,
// Invite} or {Read, HeadInvite}. Otherwise it does not exist for them
// (ErrNotFound).
func ReadInvite(ctx context.Context, id uuid.UUID) (*InviteCard, error) {
	p, err := readInviteGates(ctx, &id, nil)
	if err != nil {
		return nil, err
	}
	return loadInvite(ctx, p)
}

// ReadInviteByLink opens the open invite behind a link's token, as typed:
// {Read, Invite, ViaInviteLink}, for anyone who has it. A token that opens
// none counts as a guess (wordtoken's lock).
func ReadInviteByLink(ctx context.Context, token string) (*InviteCard, error) {
	stored, err := resolveInviteToken(ctx, token)
	if err != nil {
		return nil, err
	}
	p, err := readInviteGates(ctx, nil, &stored)
	if err != nil {
		return nil, err
	}
	return loadInvite(ctx, p)
}

// resolveInviteToken is the stored form of a typed token: the token itself
// if it exists (old random tokens are case-sensitive), else its normalised
// word form. A token found in neither form counts as a guess (tokenlock).
func resolveInviteToken(ctx context.Context, token string) (string, error) {
	for _, candidate := range []string{token, wordtoken.Normalize(token)} {
		ok, err := store.Queries().V2InviteTokenExists(ctx, sql.NullString{String: candidate, Valid: true})
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}
	tokenlock.Fail()
	return token, nil
}

func loadInvite(ctx context.Context, p db.V2InvitesParams) (*InviteCard, error) {
	invites, err := loadInvites(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(invites) == 0 {
		return nil, ErrNotFound
	}
	return invites[0], nil
}

func loadInvites(ctx context.Context, p db.V2InvitesParams) ([]*InviteCard, error) {
	rows, err := store.Queries().V2Invites(ctx, p)
	if err != nil {
		return nil, err
	}
	invites := make([]*InviteCard, len(rows))
	for i, r := range rows {
		inv := &InviteCard{
			Invite: Invite{
				ID: r.ID, CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
				OrganisationID: r.InstitutionID, Role: r.Role,
				WorkshopID: ptrUUID(r.WorkshopID), InvitedUserID: ptrUUID(r.InvitedUserID),
				InvitedEmail: nullString(r.InvitedEmail), Token: nullString(r.InviteToken),
				UsesCount: r.UsesCount, ExpiresAt: nullTime(r.ExpiresAt), Status: r.Status,
				AcceptedAt: nullTime(r.AcceptedAt), AcceptedBy: ptrUUID(r.AcceptedBy),
			},
			OrganisationName:  r.OrganisationName,
			WorkshopName:      nullString(r.WorkshopName),
			AddressedToCaller: r.AddressedToCaller,
			mayDelete:         r.MayDelete,
			mayUse:            r.MayUse,
		}
		if r.MaxUses.Valid {
			inv.MaxUses = &r.MaxUses.Int32
		}
		invites[i] = inv
	}
	return invites, nil
}

// organisationRoles are what an organisation invites as: head and staff are
// in an organisation, an individual is in none. An individual who accepts
// becomes head or staff.
var organisationRoles = []string{string(obj.RoleHead), string(obj.RoleStaff)}

// OrganisationInvite is an invite into an organisation: the role, and the
// account it is for, by id or email.
type OrganisationInvite struct {
	Organisation uuid.UUID
	Role         string // head or staff
	User         *uuid.UUID
	Email        *string
}

// InviteToOrganisation invites an account into an organisation and returns
// the invite. As head it takes {Create, HeadInvite}, as staff {Create,
// Invite}: in the caller's organisation, or, for the admin, in any.
// The account must exist; one with a pending invite there is refused (409).
func InviteToOrganisation(ctx context.Context, in OrganisationInvite) (*InviteCard, error) {
	if !contains(organisationRoles, in.Role) {
		return nil, refuse("an organisation invites as head or staff, not %q", in.Role)
	}
	if !mayCreateInvite(ctx, in.Organisation, in.Role == string(obj.RoleHead)) {
		return nil, forbidden("you may not invite into this organisation as %s", in.Role)
	}
	user, err := inviteeOf(ctx, in.User, in.Email)
	if err != nil {
		return nil, err
	}
	if err := checkNoPendingInvite(ctx, in.Organisation, user, in.Email, nil); err != nil {
		return nil, err
	}
	created, err := store.Queries().V2CreateTargetedInvite(ctx, db.V2CreateTargetedInviteParams{
		CreatedBy:     account(ctx),
		InstitutionID: in.Organisation,
		Role:          in.Role,
		InvitedUserID: uuid.NullUUID{UUID: user, Valid: true},
		InvitedEmail:  sqlString(in.Email),
	})
	if err != nil {
		return nil, err
	}
	return ReadInvite(ctx, created.ID)
}

// WorkshopLink is an open invite into a workshop: how often it may be
// followed, and until when; nil: without a limit.
type WorkshopLink struct {
	Workshop uuid.UUID
	MaxUses  *int32
	Expires  *time.Time
}

// InviteToWorkshop returns the workshop's link: its usable one, or a new
// one, after marking those used up or expired. It takes {Create, Invite} in
// the workshop's organisation; a workshop the caller may not read does not
// exist for them (ErrNotFound).
func InviteToWorkshop(ctx context.Context, l WorkshopLink) (*InviteCard, error) {
	w, err := ReadWorkshop(ctx, l.Workshop)
	if err != nil {
		return nil, err
	}
	if !mayCreateInvite(ctx, w.row.organisationID, false) {
		return nil, forbidden("you may not invite into this workshop")
	}
	q := store.Queries()
	existing, err := q.V2GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: l.Workshop, Valid: true})
	if err != nil {
		return nil, err
	}
	for _, inv := range existing {
		if inv.Status != string(obj.InviteStatusPending) || !inv.InviteToken.Valid {
			continue
		}
		if usedUp(inviteOf(inv)) {
			_ = setInviteStatus(ctx, q, inv.ID, obj.InviteStatusExpired)
			continue
		}
		return ReadInvite(ctx, inv.ID)
	}
	// 3 words = 33 bits; guessing is bounded by the token lock
	token, err := wordtoken.GenerateUnique(ctx, 3, func(ctx context.Context, t string) (bool, error) {
		return q.V2InviteTokenExists(ctx, sql.NullString{String: t, Valid: true})
	})
	if err != nil {
		return nil, err
	}
	params := db.V2CreateOpenInviteParams{
		CreatedBy:     account(ctx),
		InstitutionID: w.row.organisationID,
		Role:          string(obj.RoleGuest),
		WorkshopID:    uuid.NullUUID{UUID: l.Workshop, Valid: true},
		InviteToken:   sql.NullString{String: token, Valid: true},
	}
	if l.MaxUses != nil {
		params.MaxUses = sql.NullInt32{Int32: *l.MaxUses, Valid: true}
	}
	if l.Expires != nil {
		params.ExpiresAt = sql.NullTime{Time: *l.Expires, Valid: true}
	}
	created, err := q.V2CreateOpenInvite(ctx, params)
	if err != nil {
		return nil, err
	}
	return ReadInvite(ctx, created.ID)
}

// InviteToWorkshopByEmail invites the account with this email into a
// workshop: accepting it, they work in it and keep their role. It takes
// {Create, Invite} in the workshop's organisation. Its head and staff need
// none, they work in all its workshops; one with a pending invite to the
// same workshop is refused (409).
func InviteToWorkshopByEmail(ctx context.Context, workshop uuid.UUID, email string) (*InviteCard, error) {
	if email = strings.TrimSpace(email); email == "" {
		return nil, refuse("email is required")
	}
	w, err := ReadWorkshop(ctx, workshop)
	if err != nil {
		return nil, err
	}
	if !mayCreateInvite(ctx, w.row.organisationID, false) {
		return nil, forbidden("you may not invite into this workshop")
	}
	user, err := inviteeOf(ctx, nil, &email)
	if err != nil {
		return nil, err
	}
	// head and staff of the organisation are users the caller reads
	if member, err := ReadUser(ctx, user); err == nil && member.row.organisationID != nil &&
		*member.row.organisationID == w.row.organisationID &&
		(member.row.role == string(obj.RoleHead) || member.row.role == string(obj.RoleStaff)) {
		return nil, refuse("this user is head or staff of the organisation and works in all its workshops")
	}
	if err := checkNoPendingInvite(ctx, w.row.organisationID, user, &email, &workshop); err != nil {
		return nil, err
	}
	created, err := store.Queries().V2CreateTargetedInvite(ctx, db.V2CreateTargetedInviteParams{
		CreatedBy:     account(ctx),
		InstitutionID: w.row.organisationID,
		Role:          string(obj.RoleGuest),
		WorkshopID:    uuid.NullUUID{UUID: workshop, Valid: true},
		InvitedUserID: uuid.NullUUID{UUID: user, Valid: true},
		InvitedEmail:  sql.NullString{String: email, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return ReadInvite(ctx, created.ID)
}

// inviteeOf is the account an invite is for: by id, or the one with the
// email, which must exist.
func inviteeOf(ctx context.Context, id *uuid.UUID, email *string) (uuid.UUID, error) {
	if email != nil {
		u, err := store.Queries().V2GetUserByEmail(ctx, sql.NullString{String: *email, Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return u.ID, err
	}
	if id == nil {
		return uuid.Nil, refuse("invitedUserId or invitedEmail is required")
	}
	return *id, nil
}

// checkNoPendingInvite refuses an invite while the same account has a
// pending one into the organisation; with workshop, only one into that
// workshop.
func checkNoPendingInvite(ctx context.Context, org, user uuid.UUID, email *string, workshop *uuid.UUID) error {
	existing, err := store.Queries().V2GetPendingInviteByTarget(ctx, db.V2GetPendingInviteByTargetParams{
		InstitutionID: org,
		InvitedUserID: uuid.NullUUID{UUID: user, Valid: true},
		InvitedEmail:  sqlString(email),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if workshop == nil || (existing.WorkshopID.Valid && existing.WorkshopID.UUID == *workshop) {
		return conflict(obj.ErrCodeConflict, "a pending invite exists for this user already")
	}
	return nil
}

// usedUp says whether an open invite expired or was followed as often as it
// may be.
func usedUp(inv Invite) bool {
	return (inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now())) ||
		(inv.MaxUses != nil && inv.UsesCount >= *inv.MaxUses)
}

// AcceptedInvite is what accepting an invite did. For someone who joined
// without an account, Guest is who they are now and GuestCode their login
// code.
type AcceptedInvite struct {
	Guest     *NewGuest
	GuestCode *string
	Message   string
}

// NewGuest is a guest account just made by following a link.
type NewGuest struct {
	ID       uuid.UUID
	Name     string
	Language string
}

// AcceptInvite accepts an invite, by its id or its link's token. It takes
// {Use, Invite} or {Use, HeadInvite}: the invite's person, or, for a link,
// whoever has it. Only a pending invite is accepted (409 otherwise).
//
//   - Following a link without an account joins the workshop as a new guest,
//     signed in at once.
//   - An email invite to a workshop gives no role: it enters the workshop
//     (EnterWorkshop), which takes the right to.
//   - Following a workshop's link, head, staff and individuals keep their
//     role and work in the workshop: taking part makes no one part of its
//     organisation. Anyone else takes its role.
//   - A guest accepts none: a guest's place is fixed. A guest of a share
//     link is in no workshop, a workshop's guest in theirs, and neither
//     enters or leaves one; that is what tells the two apart.
//   - Any other targeted invite gives its role, which replaces the
//     account's.
func AcceptInvite(ctx context.Context, idOrToken string, language string) (AcceptedInvite, error) {
	inv, err := ReadInviteRef(ctx, idOrToken)
	if err != nil {
		return AcceptedInvite{}, err
	}
	if err := checkAnswerable(ctx, inv); err != nil {
		return AcceptedInvite{}, err
	}
	c := from(ctx)
	switch {
	case c.Account() == nil:
		return joinAsGuest(ctx, inv, language)
	case c.Role() == rules.Guest:
		if inv.WorkshopID != nil && c.CurrentWorkshop() != nil && *c.CurrentWorkshop() == *inv.WorkshopID {
			return AcceptedInvite{}, conflict(obj.ErrCodeConflict, "you are in this workshop already")
		}
		return AcceptedInvite{}, forbiddenWith(obj.ErrCodeGuestStays, "a guest stays where they are")
	case inv.WorkshopID != nil && !inv.open():
		// an email invite to a workshop gives no role: it enters it
		if err := enterWorkshop(ctx, *inv.WorkshopID, true); err != nil {
			return AcceptedInvite{}, err
		}
		if err := store.Queries().V2AcceptTargetedInvite(ctx, db.V2AcceptTargetedInviteParams{ID: inv.ID, AcceptedBy: account(ctx)}); err != nil {
			return AcceptedInvite{}, err
		}
		return AcceptedInvite{Message: "Workshop entered"}, nil
	case keepsRole(ctx) && inv.WorkshopID != nil:
		if err := enterWorkshop(ctx, *inv.WorkshopID, true); err != nil {
			return AcceptedInvite{}, err
		}
		return AcceptedInvite{Message: "Workshop entered"}, nil
	case inv.open():
		return AcceptedInvite{Message: "Invite accepted"}, takeOpenInvite(ctx, inv)
	}
	return AcceptedInvite{Message: "Invite accepted"}, takeTargetedInvite(ctx, inv)
}

// keepsRole says whether the caller keeps their role in a workshop instead
// of becoming its guest: head, staff and individuals. It is how accepting
// behaves, not who may: that is the invite's grants.
func keepsRole(ctx context.Context) bool {
	switch from(ctx).Role() {
	case rules.Head, rules.Staff, rules.Individual:
		return true
	}
	return false
}

// DeclineInvite declines an invite addressed to the caller: {Use, Invite}
// or {Use, HeadInvite}. A link is not declined; only a pending invite is.
func DeclineInvite(ctx context.Context, id uuid.UUID) error {
	inv, err := ReadInvite(ctx, id)
	if err != nil {
		return err
	}
	if inv.open() {
		return refuse("a link is not declined")
	}
	if err := checkAnswerable(ctx, inv); err != nil {
		return err
	}
	return setInviteStatus(ctx, store.Queries(), id, obj.InviteStatusDeclined)
}

// RevokeInvite deletes a pending invite: {Delete, Invite} or {Delete,
// HeadInvite}. An invite answered already stays as the record of it (409).
func RevokeInvite(ctx context.Context, id uuid.UUID) error {
	inv, err := ReadInvite(ctx, id)
	if err != nil {
		return err
	}
	if !inv.MayRevoke() {
		return forbidden("you may not revoke this invite")
	}
	if inv.Status != string(obj.InviteStatusPending) {
		return conflict(obj.ErrCodeConflict, "only a pending invite is revoked; this one is %s", inv.Status)
	}
	return store.Queries().V2DeleteInvite(ctx, id)
}

// ReadInviteRef reads an invite by its id or, otherwise, its link's token
// (ReadInvite, ReadInviteByLink).
func ReadInviteRef(ctx context.Context, idOrToken string) (*InviteCard, error) {
	if id, err := uuid.Parse(idOrToken); err == nil {
		return ReadInvite(ctx, id)
	}
	return ReadInviteByLink(ctx, idOrToken)
}

// checkAnswerable refuses answering an invite the caller may not answer, one
// no longer pending (409), and a link used up, which it marks expired (409).
func checkAnswerable(ctx context.Context, inv *InviteCard) error {
	if !inv.MayAnswer() {
		return forbidden("this invite is for someone else")
	}
	if inv.Status != string(obj.InviteStatusPending) {
		return conflict(obj.ErrCodeConflict, "this invite is %s, no longer pending", inv.Status)
	}
	if inv.open() && usedUp(inv.Invite) {
		_ = setInviteStatus(ctx, store.Queries(), inv.ID, obj.InviteStatusExpired)
		return conflict(obj.ErrCodeConflict, "this invite link has expired")
	}
	return nil
}

// joinAsGuest makes a new guest account in the link's workshop, with a
// readable name and a login code, signs it in, and returns the code.
func joinAsGuest(ctx context.Context, inv *InviteCard, language string) (AcceptedInvite, error) {
	if inv.WorkshopID == nil {
		return AcceptedInvite{}, refuse("this is not a workshop invite")
	}
	// a readable name, e.g. "red-dragon-a1b2": the suffix keeps it unique
	names := ung.NewUniqueNameGenerator(
		ung.WithDictionaries([][]string{dictionaries.Colors, dictionaries.Animals}),
		ung.WithSeparator("-"),
	)
	name := names.Generate() + "-" + functional.First(functional.GenerateSecureToken(2))
	code, err := newGuestToken(ctx)
	if err != nil {
		return AcceptedInvite{}, err
	}
	id := uuid.New()
	err = store.InTx(ctx, func(q *db.Queries) error {
		if _, err := q.V2CreateUserWithGuestToken(ctx, db.V2CreateUserWithGuestTokenParams{
			ID:         id,
			Name:       sql.NullString{String: name, Valid: true},
			GuestToken: sql.NullString{String: code, Valid: true},
			Language:   language,
		}); err != nil {
			return err
		}
		if _, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{
			UserID:     id,
			Role:       string(obj.RoleGuest),
			WorkshopID: uuid.NullUUID{UUID: *inv.WorkshopID, Valid: true}, // a guest's organisation is their workshop's
		}); err != nil {
			return err
		}
		return q.V2IncrementInviteUses(ctx, inv.ID)
	})
	if err != nil {
		return AcceptedInvite{}, err
	}
	if _, err := schemafapi.StartSessionCtx(ctx, id.String()); err != nil {
		return AcceptedInvite{}, err
	}
	events.GetBroker().PublishMembersUpdated(*inv.WorkshopID)
	log.Info("guest joined a workshop", "workshop_id", *inv.WorkshopID, "user_id", id)
	return AcceptedInvite{
		Guest:     &NewGuest{ID: id, Name: name, Language: language},
		GuestCode: &code,
		Message:   "Workshop invite accepted, user created",
	}, nil
}

// takeOpenInvite gives the caller the link's role in its workshop, replacing
// theirs, and counts the use; a link followed as often as it may be expires.
func takeOpenInvite(ctx context.Context, inv *InviteCard) error {
	me, err := ReadUser(ctx, *from(ctx).Account())
	if err != nil {
		return err
	}
	if inv.WorkshopID != nil && me.row.workshopID != nil && *me.row.workshopID == *inv.WorkshopID {
		return conflict(obj.ErrCodeConflict, "you are in this workshop already")
	}
	previous := me.row.workshopID
	err = store.InTx(ctx, func(q *db.Queries) error {
		if err := replaceRole(ctx, q, me.row.id, inv); err != nil {
			return err
		}
		if err := q.V2IncrementInviteUses(ctx, inv.ID); err != nil {
			return err
		}
		if inv.MaxUses != nil && inv.UsesCount+1 >= *inv.MaxUses {
			return setInviteStatus(ctx, q, inv.ID, obj.InviteStatusExpired)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if previous != nil {
		events.GetBroker().PublishMembersUpdated(*previous)
	}
	if inv.WorkshopID != nil {
		events.GetBroker().PublishMembersUpdated(*inv.WorkshopID)
	}
	return nil
}

// takeTargetedInvite gives the caller the invite's role, replacing theirs,
// and marks it accepted.
func takeTargetedInvite(ctx context.Context, inv *InviteCard) error {
	me := *from(ctx).Account()
	return store.InTx(ctx, func(q *db.Queries) error {
		if err := replaceRole(ctx, q, me, inv); err != nil {
			return err
		}
		return q.V2AcceptTargetedInvite(ctx, db.V2AcceptTargetedInviteParams{ID: inv.ID, AcceptedBy: account(ctx)})
	})
}

// replaceRole makes inv's role the account's only one. Only head and staff
// are in the organisation; a guest is in its workshop.
func replaceRole(ctx context.Context, q *db.Queries, user uuid.UUID, inv *InviteCard) error {
	if err := q.V2DeleteUserRoles(ctx, user); err != nil {
		return err
	}
	inOrganisation := inv.Role == string(obj.RoleHead) || inv.Role == string(obj.RoleStaff)
	_, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{
		UserID:        user,
		Role:          inv.Role,
		InstitutionID: uuid.NullUUID{UUID: inv.OrganisationID, Valid: inOrganisation},
		WorkshopID:    nullUUID(inv.WorkshopID),
	})
	return err
}

// setInviteStatus records an invite's new status.
func setInviteStatus(ctx context.Context, q *db.Queries, id uuid.UUID, status obj.InviteStatus) error {
	return q.V2UpdateInviteStatus(ctx, db.V2UpdateInviteStatusParams{ID: id, Status: string(status)})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
