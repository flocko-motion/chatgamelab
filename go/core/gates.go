package core

// The gates: which grants of the table (core/internal/rules) open which
// fragment of a query (go/db/queries/v2_*.sql). Each parameter is one line of
// the table; a maintainer checks table, gates and SQL against each other here.
// gates_test.go fails when the table grants a relation no gate opens.

import (
	"context"

	"cgl/core/internal/rules"
	"cgl/db"
	"cgl/legal"

	"github.com/google/uuid"
)

// listGameCardsGates opens V2GameCards for a list: the rows by {List, GameCard}.
func listGameCardsGates(ctx context.Context) db.V2GameCardsParams {
	p := gameCardGates(ctx)
	p.CardPublic = granted(ctx, rules.List, rules.GameCard, rules.Public)
	p.CardOwn = granted(ctx, rules.List, rules.GameCard, rules.Own)
	p.CardPublicElsewhere = granted(ctx, rules.List, rules.GameCard, rules.PublicElsewhere)
	p.CardOwnOfWorkshop = granted(ctx, rules.List, rules.GameCard, rules.OwnOfCurrentWorkshop)
	p.CardWorkshop = granted(ctx, rules.List, rules.GameCard, rules.OfCurrentWorkshop)
	p.CardEverywhere = granted(ctx, rules.List, rules.GameCard, rules.Everywhere)
	return p
}

// readGameCardGates opens V2GameCards for one game: the row by {Read, GameCard}.
func readGameCardGates(ctx context.Context, id uuid.UUID) db.V2GameCardsParams {
	p := gameCardGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	p.CardPublic = granted(ctx, rules.Read, rules.GameCard, rules.Public)
	p.CardOwn = granted(ctx, rules.Read, rules.GameCard, rules.Own)
	p.CardWorkshop = granted(ctx, rules.Read, rules.GameCard, rules.OfCurrentWorkshop)
	p.CardOrganisation = granted(ctx, rules.Read, rules.GameCard, rules.OfOwnOrganisation)
	p.CardViaShare = granted(ctx, rules.Read, rules.GameCard, rules.ViaShareLink)
	p.CardEverywhere = granted(ctx, rules.Read, rules.GameCard, rules.Everywhere)
	return p
}

// gameCardGates opens what list and read share: the sections by
// {Read, GameContent} and {Read, GameSettings}, the actions by {Edit, <section>},
// {Delete, GameCard}, {Copy, GameContent}, {Share, GameCard} and
// {Publish, GameCard}.
func gameCardGates(ctx context.Context) db.V2GameCardsParams {
	return db.V2GameCardsParams{
		Caller:          account(ctx),
		CurrentWorkshop: currentWorkshop(ctx),
		OwnOrganisation: ownOrganisation(ctx),

		ContentPublic:       granted(ctx, rules.Read, rules.GameContent, rules.Public),
		ContentOwn:          granted(ctx, rules.Read, rules.GameContent, rules.Own),
		ContentWorkshop:     granted(ctx, rules.Read, rules.GameContent, rules.OfCurrentWorkshop),
		ContentOrganisation: granted(ctx, rules.Read, rules.GameContent, rules.OfOwnOrganisation),
		ContentEverywhere:   granted(ctx, rules.Read, rules.GameContent, rules.Everywhere),

		SettingsPublic:       granted(ctx, rules.Read, rules.GameSettings, rules.Public),
		SettingsOwn:          granted(ctx, rules.Read, rules.GameSettings, rules.Own),
		SettingsWorkshop:     granted(ctx, rules.Read, rules.GameSettings, rules.OfCurrentWorkshop),
		SettingsOrganisation: granted(ctx, rules.Read, rules.GameSettings, rules.OfOwnOrganisation),
		SettingsEverywhere:   granted(ctx, rules.Read, rules.GameSettings, rules.Everywhere),

		EditCardOwn:              granted(ctx, rules.Edit, rules.GameCard, rules.Own),
		EditCardOrganisation:     granted(ctx, rules.Edit, rules.GameCard, rules.OfOwnOrganisation),
		EditCardEverywhere:       granted(ctx, rules.Edit, rules.GameCard, rules.Everywhere),
		EditContentOwn:           granted(ctx, rules.Edit, rules.GameContent, rules.Own),
		EditContentOrganisation:  granted(ctx, rules.Edit, rules.GameContent, rules.OfOwnOrganisation),
		EditContentEverywhere:    granted(ctx, rules.Edit, rules.GameContent, rules.Everywhere),
		EditSettingsOwn:          granted(ctx, rules.Edit, rules.GameSettings, rules.Own),
		EditSettingsOrganisation: granted(ctx, rules.Edit, rules.GameSettings, rules.OfOwnOrganisation),
		EditSettingsEverywhere:   granted(ctx, rules.Edit, rules.GameSettings, rules.Everywhere),

		DeleteOwn:          granted(ctx, rules.Delete, rules.GameCard, rules.Own),
		DeleteOrganisation: granted(ctx, rules.Delete, rules.GameCard, rules.OfOwnOrganisation),
		DeleteEverywhere:   granted(ctx, rules.Delete, rules.GameCard, rules.Everywhere),

		CopyPublic: granted(ctx, rules.Copy, rules.GameContent, rules.Public),
		CopyOwn:    granted(ctx, rules.Copy, rules.GameContent, rules.Own),

		SharePublic: granted(ctx, rules.Share, rules.GameCard, rules.Public),
		ShareOwn:    granted(ctx, rules.Share, rules.GameCard, rules.Own),

		PublishOwn:          granted(ctx, rules.Publish, rules.GameCard, rules.Own),
		PublishOrganisation: granted(ctx, rules.Publish, rules.GameCard, rules.OfOwnOrganisation),
		PublishEverywhere:   granted(ctx, rules.Publish, rules.GameCard, rules.Everywhere),
	}
}

// listUserCardsGates opens V2Users for a list: the rows by {List, UserCard}.
func listUserCardsGates(ctx context.Context) db.V2UsersParams {
	p := userCardGates(ctx)
	p.CardMember = granted(ctx, rules.List, rules.UserCard, rules.MemberOfOwnOrganisation)
	p.CardEverywhere = granted(ctx, rules.List, rules.UserCard, rules.Everywhere)
	return p
}

// readUserCardGates opens V2Users for one user: the row by {Read, UserCard}.
func readUserCardGates(ctx context.Context, id uuid.UUID) db.V2UsersParams {
	p := userCardGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	p.CardOwn = granted(ctx, rules.Read, rules.UserCard, rules.Own)
	p.CardOrganisation = granted(ctx, rules.Read, rules.UserCard, rules.OfOwnOrganisation)
	p.CardEverywhere = granted(ctx, rules.Read, rules.UserCard, rules.Everywhere)
	return p
}

// userCardGates opens what list and read share: the sections by
// {Read, UserAccount}, {Read, UserContact} and {Read, UserSecrets}, the
// actions by {Edit, <section>} and {Delete, UserCard}.
func userCardGates(ctx context.Context) db.V2UsersParams {
	return db.V2UsersParams{
		Caller:          account(ctx),
		OwnOrganisation: ownOrganisation(ctx),

		AccountOwn:          granted(ctx, rules.Read, rules.UserAccount, rules.Own),
		AccountOrganisation: granted(ctx, rules.Read, rules.UserAccount, rules.OfOwnOrganisation),
		AccountEverywhere:   granted(ctx, rules.Read, rules.UserAccount, rules.Everywhere),

		ContactOwn:          granted(ctx, rules.Read, rules.UserContact, rules.Own),
		ContactOrganisation: granted(ctx, rules.Read, rules.UserContact, rules.OfOwnOrganisation),
		ContactEverywhere:   granted(ctx, rules.Read, rules.UserContact, rules.Everywhere),

		SecretsOwn:          granted(ctx, rules.Read, rules.UserSecrets, rules.Own),
		SecretsOrganisation: granted(ctx, rules.Read, rules.UserSecrets, rules.OfOwnOrganisation),

		EditCardOwn:           granted(ctx, rules.Edit, rules.UserCard, rules.Own),
		EditCardGuest:         granted(ctx, rules.Edit, rules.UserCard, rules.GuestOfOwnOrganisation),
		EditCardEverywhere:    granted(ctx, rules.Edit, rules.UserCard, rules.Everywhere),
		EditAccountOwn:        granted(ctx, rules.Edit, rules.UserAccount, rules.Own),
		EditAccountGuest:      granted(ctx, rules.Edit, rules.UserAccount, rules.GuestOfOwnOrganisation),
		EditAccountEverywhere: granted(ctx, rules.Edit, rules.UserAccount, rules.Everywhere),
		EditContactOwn:        granted(ctx, rules.Edit, rules.UserContact, rules.Own),
		EditContactEverywhere: granted(ctx, rules.Edit, rules.UserContact, rules.Everywhere),

		DeleteGuest:      granted(ctx, rules.Delete, rules.UserCard, rules.GuestOfOwnOrganisation),
		DeleteEverywhere: granted(ctx, rules.Delete, rules.UserCard, rules.Everywhere),
	}
}

// listOrganisationsGates opens V2Organisations for a list: the rows by
// {List, OrganisationCard}.
func listOrganisationsGates(ctx context.Context) db.V2OrganisationsParams {
	p := organisationGates(ctx)
	p.CardOrganisation = granted(ctx, rules.List, rules.OrganisationCard, rules.OfOwnOrganisation)
	p.CardEverywhere = granted(ctx, rules.List, rules.OrganisationCard, rules.Everywhere)
	return p
}

// readOrganisationGates opens V2Organisations for one organisation: the row
// by {Read, OrganisationCard}.
func readOrganisationGates(ctx context.Context, id uuid.UUID) db.V2OrganisationsParams {
	p := organisationGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	p.CardOrganisation = granted(ctx, rules.Read, rules.OrganisationCard, rules.OfOwnOrganisation)
	p.CardEverywhere = granted(ctx, rules.Read, rules.OrganisationCard, rules.Everywhere)
	return p
}

// organisationGates opens what list and read share: the sections by
// {Read, OrganisationSettings}, {Read, OrganisationDetails} and
// {Read, OrganisationRecord}, the actions by {Edit, <section>},
// {Delete, OrganisationCard} and {Suspend, OrganisationCard}.
func organisationGates(ctx context.Context) db.V2OrganisationsParams {
	declaration := legal.CurrentDeclaration()
	return db.V2OrganisationsParams{
		DeclarationVersion: declaration.Version,
		DeclarationHash:    declaration.Hash,
		OwnOrganisation:    ownOrganisation(ctx),

		SettingsOrganisation: granted(ctx, rules.Read, rules.OrganisationSettings, rules.OfOwnOrganisation),
		SettingsEverywhere:   granted(ctx, rules.Read, rules.OrganisationSettings, rules.Everywhere),
		DetailsOrganisation:  granted(ctx, rules.Read, rules.OrganisationDetails, rules.OfOwnOrganisation),
		DetailsEverywhere:    granted(ctx, rules.Read, rules.OrganisationDetails, rules.Everywhere),
		RecordOrganisation:   granted(ctx, rules.Read, rules.OrganisationRecord, rules.OfOwnOrganisation),
		RecordEverywhere:     granted(ctx, rules.Read, rules.OrganisationRecord, rules.Everywhere),

		EditCardOrganisation:     granted(ctx, rules.Edit, rules.OrganisationCard, rules.OfOwnOrganisation),
		EditCardEverywhere:       granted(ctx, rules.Edit, rules.OrganisationCard, rules.Everywhere),
		EditSettingsOrganisation: granted(ctx, rules.Edit, rules.OrganisationSettings, rules.OfOwnOrganisation),
		EditSettingsEverywhere:   granted(ctx, rules.Edit, rules.OrganisationSettings, rules.Everywhere),
		EditDetailsOrganisation:  granted(ctx, rules.Edit, rules.OrganisationDetails, rules.OfOwnOrganisation),

		DeleteEverywhere:  granted(ctx, rules.Delete, rules.OrganisationCard, rules.Everywhere),
		SuspendEverywhere: granted(ctx, rules.Suspend, rules.OrganisationCard, rules.Everywhere),
	}
}

// listMembersGates opens V2Users for the members of organisation id: the
// rows by {List, UserCard}, narrowed to its members.
func listMembersGates(ctx context.Context, id uuid.UUID) db.V2UsersParams {
	p := listUserCardsGates(ctx)
	p.MembersOf = uuid.NullUUID{UUID: id, Valid: true}
	return p
}

// mayCreateOrganisation says whether the caller may register an
// organisation and become its head, {Create, OrganisationCard, Own}, and
// whether they may create organisations for others, {Create,
// OrganisationCard, Everywhere}.
func mayCreateOrganisation(ctx context.Context) (asHead, forOthers bool) {
	return granted(ctx, rules.Create, rules.OrganisationCard, rules.Own),
		granted(ctx, rules.Create, rules.OrganisationCard, rules.Everywhere)
}

// mayConsentForOwnOrganisation says whether the caller may consent to the
// declaration in their organisation's name, editing its details:
// {Edit, OrganisationDetails, OfOwnOrganisation}.
func mayConsentForOwnOrganisation(ctx context.Context) bool {
	return from(ctx).Organisation() != nil && granted(ctx, rules.Edit, rules.OrganisationDetails, rules.OfOwnOrganisation)
}

// maySuspendOrganisations says whether the caller may suspend organisations
// and lift that: {Suspend, OrganisationCard, Everywhere}.
func maySuspendOrganisations(ctx context.Context) bool {
	return granted(ctx, rules.Suspend, rules.OrganisationCard, rules.Everywhere)
}

// mayRemoveMember says whether the caller may remove user from their
// organisation: {Delete, Membership}. The user's role and organisation come
// from their loaded row; a membership has no query of its own.
func mayRemoveMember(ctx context.Context, user *UserCard) bool {
	own := from(ctx).Organisation()
	inOwn := own != nil && user.row.organisationID != nil && *user.row.organisationID == *own
	isGuest := user.row.role == "guest"
	isSelf := account(ctx).Valid && account(ctx).UUID == user.row.id
	return granted(ctx, rules.Delete, rules.Membership, rules.Everywhere) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.MemberOfOwnOrganisation) && inOwn && !isGuest) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.GuestOfOwnOrganisation) && inOwn && isGuest) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.Own) && isSelf && !isGuest)
}
