package core

// The gates: which grants of the table (core/internal/rules) open which
// fragment of a query (go/db/queries/v2_*.sql). Each parameter is one line of
// the table; a maintainer checks table, gates and SQL against each other here.
// gates_test.go fails when the table grants a relation no gate opens.

import (
	"context"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
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
// {Delete, GameCard}, {Copy, GameContent}, {Share, GameCard},
// {Publish, GameCard} and {Play, GameCard}.
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
		ContentViaShare:     granted(ctx, rules.Read, rules.GameContent, rules.ViaShareLink),

		PlayPublic:       granted(ctx, rules.Play, rules.GameCard, rules.Public),
		PlayOwn:          granted(ctx, rules.Play, rules.GameCard, rules.Own),
		PlayWorkshop:     granted(ctx, rules.Play, rules.GameCard, rules.OfCurrentWorkshop),
		PlayOrganisation: granted(ctx, rules.Play, rules.GameCard, rules.OfOwnOrganisation),
		PlayViaShare:     granted(ctx, rules.Play, rules.GameCard, rules.ViaShareLink),
		PlayEverywhere:   granted(ctx, rules.Play, rules.GameCard, rules.Everywhere),

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
	p.CardCurrentWorkshop = granted(ctx, rules.Read, rules.OrganisationCard, rules.OfCurrentWorkshop)
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
		CurrentWorkshop:    nullUUID(from(ctx).CurrentWorkshop()),

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
// from their loaded row; a membership has no query of its own. A member is
// head or staff of the organisation; a guest is of it through its workshop.
// An individual taking part in one of its workshops is neither.
func mayRemoveMember(ctx context.Context, user *UserCard) bool {
	own := from(ctx).Organisation()
	isGuest := user.row.role == "guest"
	of := user.row.memberOf
	if isGuest {
		of = user.row.organisationID
	}
	inOwn := own != nil && of != nil && *of == *own
	isSelf := account(ctx).Valid && account(ctx).UUID == user.row.id
	return granted(ctx, rules.Delete, rules.Membership, rules.Everywhere) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.MemberOfOwnOrganisation) && inOwn && !isGuest) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.GuestOfOwnOrganisation) && inOwn && isGuest) ||
		(granted(ctx, rules.Delete, rules.Membership, rules.Own) && isSelf && !isGuest)
}

// listWorkshopsGates opens V2Workshops for a list: the rows by
// {List, WorkshopCard}.
func listWorkshopsGates(ctx context.Context) db.V2WorkshopsParams {
	p := workshopGates(ctx)
	p.CardOrganisation = granted(ctx, rules.List, rules.WorkshopCard, rules.OfOwnOrganisation)
	p.CardEverywhere = granted(ctx, rules.List, rules.WorkshopCard, rules.Everywhere)
	return p
}

// readWorkshopGates opens V2Workshops for one workshop: the row by
// {Read, WorkshopCard}.
func readWorkshopGates(ctx context.Context, id uuid.UUID) db.V2WorkshopsParams {
	p := workshopGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	p.CardOrganisation = granted(ctx, rules.Read, rules.WorkshopCard, rules.OfOwnOrganisation)
	p.CardCurrent = granted(ctx, rules.Read, rules.WorkshopCard, rules.Current)
	p.CardPublic = granted(ctx, rules.Read, rules.WorkshopCard, rules.Public)
	p.CardEverywhere = granted(ctx, rules.Read, rules.WorkshopCard, rules.Everywhere)
	return p
}

// workshopGates opens what list and read share: the sections by
// {Read, WorkshopSettings} and {List, WorkshopParticipants},
// the actions by {Edit, <section>}, {Delete, WorkshopCard} and
// {Manage, WorkshopParticipants}.
func workshopGates(ctx context.Context) db.V2WorkshopsParams {
	return db.V2WorkshopsParams{
		OwnOrganisation: ownOrganisation(ctx),
		CurrentWorkshop: currentWorkshop(ctx),

		SettingsOrganisation:     granted(ctx, rules.Read, rules.WorkshopSettings, rules.OfOwnOrganisation),
		SettingsEverywhere:       granted(ctx, rules.Read, rules.WorkshopSettings, rules.Everywhere),
		ParticipantsOrganisation: granted(ctx, rules.List, rules.WorkshopParticipants, rules.OfOwnOrganisation),
		ParticipantsCurrent:      granted(ctx, rules.List, rules.WorkshopParticipants, rules.Current),
		ParticipantsEverywhere:   granted(ctx, rules.List, rules.WorkshopParticipants, rules.Everywhere),

		EditCardOrganisation:           granted(ctx, rules.Edit, rules.WorkshopCard, rules.OfOwnOrganisation),
		EditCardEverywhere:             granted(ctx, rules.Edit, rules.WorkshopCard, rules.Everywhere),
		EditSettingsOrganisation:       granted(ctx, rules.Edit, rules.WorkshopSettings, rules.OfOwnOrganisation),
		EditSettingsEverywhere:         granted(ctx, rules.Edit, rules.WorkshopSettings, rules.Everywhere),
		DeleteOrganisation:             granted(ctx, rules.Delete, rules.WorkshopCard, rules.OfOwnOrganisation),
		DeleteEverywhere:               granted(ctx, rules.Delete, rules.WorkshopCard, rules.Everywhere),
		ManageParticipantsOrganisation: granted(ctx, rules.Manage, rules.WorkshopParticipants, rules.OfOwnOrganisation),
		ManageParticipantsEverywhere:   granted(ctx, rules.Manage, rules.WorkshopParticipants, rules.Everywhere),
	}
}

// mayCreateWorkshop says whether the caller may create a workshop in
// organisation org: {Create, WorkshopCard, OfOwnOrganisation} in their own,
// {Create, WorkshopCard, Everywhere} in any.
func mayCreateWorkshop(ctx context.Context, org uuid.UUID) bool {
	own := from(ctx).Organisation()
	return granted(ctx, rules.Create, rules.WorkshopCard, rules.Everywhere) ||
		(granted(ctx, rules.Create, rules.WorkshopCard, rules.OfOwnOrganisation) && own != nil && *own == org)
}

// mayResetLoginCode says whether the caller may give user a new login code:
// {Edit, UserSecrets}, for a guest of their organisation.
func mayResetLoginCode(ctx context.Context, user *UserCard) bool {
	own := from(ctx).Organisation()
	return granted(ctx, rules.Edit, rules.UserSecrets, rules.GuestOfOwnOrganisation) &&
		user.row.role == "guest" && own != nil && user.row.organisationID != nil && *user.row.organisationID == *own
}

// verifiedPageRights is what the caller may do with one organisation's
// verified pages.
type verifiedPageRights struct {
	create, list, edit, delete bool
}

// verifiedPageGates says what the caller may do with organisation org's
// verified pages: {Create, List, Edit, Delete} on VerifiedPage,
// OfOwnOrganisation, in their own organisation.
func verifiedPageGates(ctx context.Context, org uuid.UUID) verifiedPageRights {
	own := from(ctx).Organisation()
	inOwn := own != nil && *own == org
	return verifiedPageRights{
		create: inOwn && granted(ctx, rules.Create, rules.VerifiedPage, rules.OfOwnOrganisation),
		list:   inOwn && granted(ctx, rules.List, rules.VerifiedPage, rules.OfOwnOrganisation),
		edit:   inOwn && granted(ctx, rules.Edit, rules.VerifiedPage, rules.OfOwnOrganisation),
		delete: inOwn && granted(ctx, rules.Delete, rules.VerifiedPage, rules.OfOwnOrganisation),
	}
}

// listInvitesGates opens V2Invites for a list: the rows by {List, Invite}
// and {List, HeadInvite}.
func listInvitesGates(ctx context.Context) (db.V2InvitesParams, error) {
	p, err := inviteGates(ctx)
	p.CardOrganisation = granted(ctx, rules.List, rules.Invite, rules.OfOwnOrganisation)
	p.HeadOrganisation = granted(ctx, rules.List, rules.HeadInvite, rules.OfOwnOrganisation)
	p.CardMine = granted(ctx, rules.List, rules.Invite, rules.SharedWithMe)
	p.HeadMine = granted(ctx, rules.List, rules.HeadInvite, rules.SharedWithMe)
	p.CardEverywhere = granted(ctx, rules.List, rules.Invite, rules.Everywhere)
	p.HeadEverywhere = granted(ctx, rules.List, rules.HeadInvite, rules.Everywhere)
	return p, err
}

// readInviteGates opens V2Invites for one invite, by its id or the token of
// its link: the row by {Read, Invite} and {Read, HeadInvite}.
func readInviteGates(ctx context.Context, id *uuid.UUID, token *string) (db.V2InvitesParams, error) {
	p, err := inviteGates(ctx)
	p.ID = nullUUID(id)
	p.Token = sqlString(token)
	p.CardOrganisation = granted(ctx, rules.Read, rules.Invite, rules.OfOwnOrganisation)
	p.HeadOrganisation = granted(ctx, rules.Read, rules.HeadInvite, rules.OfOwnOrganisation)
	p.CardMine = granted(ctx, rules.Read, rules.Invite, rules.SharedWithMe)
	p.HeadMine = granted(ctx, rules.Read, rules.HeadInvite, rules.SharedWithMe)
	p.CardLink = granted(ctx, rules.Read, rules.Invite, rules.ViaInviteLink)
	p.CardEverywhere = granted(ctx, rules.Read, rules.Invite, rules.Everywhere)
	p.HeadEverywhere = granted(ctx, rules.Read, rules.HeadInvite, rules.Everywhere)
	return p, err
}

// inviteGates opens what list and read share: the actions by
// {Delete, Invite}, {Delete, HeadInvite}, {Use, Invite} and
// {Use, HeadInvite}. An invite addressed by email is the caller's when the
// email is theirs.
func inviteGates(ctx context.Context) (db.V2InvitesParams, error) {
	p := db.V2InvitesParams{
		Caller:          account(ctx),
		OwnOrganisation: ownOrganisation(ctx),

		DeleteOrganisation:     granted(ctx, rules.Delete, rules.Invite, rules.OfOwnOrganisation),
		DeleteHeadOrganisation: granted(ctx, rules.Delete, rules.HeadInvite, rules.OfOwnOrganisation),
		DeleteEverywhere:       granted(ctx, rules.Delete, rules.Invite, rules.Everywhere),
		DeleteHeadEverywhere:   granted(ctx, rules.Delete, rules.HeadInvite, rules.Everywhere),
		UseMine:                granted(ctx, rules.Use, rules.Invite, rules.SharedWithMe),
		UseHeadMine:            granted(ctx, rules.Use, rules.HeadInvite, rules.SharedWithMe),
		UseLink:                granted(ctx, rules.Use, rules.Invite, rules.ViaInviteLink),
	}
	if id := from(ctx).Account(); id != nil {
		email, err := store.Queries().V2UserEmail(ctx, *id)
		if err != nil {
			return p, err
		}
		p.CallerEmail = email
	}
	return p, nil
}

// mayCreateInvite says whether the caller may invite into organisation org:
// as head, {Create, HeadInvite}; otherwise {Create, Invite}; in their own
// organisation, or, for the admin, in any.
func mayCreateInvite(ctx context.Context, org uuid.UUID, asHead bool) bool {
	own := from(ctx).Organisation()
	inOwn := own != nil && *own == org
	if asHead {
		return granted(ctx, rules.Create, rules.HeadInvite, rules.Everywhere) ||
			(granted(ctx, rules.Create, rules.HeadInvite, rules.OfOwnOrganisation) && inOwn)
	}
	return granted(ctx, rules.Create, rules.Invite, rules.Everywhere) ||
		(granted(ctx, rules.Create, rules.Invite, rules.OfOwnOrganisation) && inOwn)
}

// mayChangeRole says whether the caller may change user's role, and whether
// only within their organisation, to head or staff: {Edit, Membership}
// Everywhere (any role, the admin's) or MemberOfOwnOrganisation.
func mayChangeRole(ctx context.Context, user *UserCard) (any, withinOwn bool) {
	own := from(ctx).Organisation()
	inOwn := own != nil && user.row.memberOf != nil && *user.row.memberOf == *own
	return granted(ctx, rules.Edit, rules.Membership, rules.Everywhere),
		granted(ctx, rules.Edit, rules.Membership, rules.MemberOfOwnOrganisation) && inOwn
}

// mayEnterWorkshop says whether the caller may work in a running workshop
// of organisation org: {Enter, WorkshopCard, OfOwnOrganisation}, head and
// staff in their organisation's; through an invite, {Enter, WorkshopCard,
// ViaInvite}, an individual in any.
func mayEnterWorkshop(ctx context.Context, org uuid.UUID, active, viaInvite bool) bool {
	own := from(ctx).Organisation()
	return active && ((viaInvite && granted(ctx, rules.Enter, rules.WorkshopCard, rules.ViaInvite)) ||
		(granted(ctx, rules.Enter, rules.WorkshopCard, rules.OfOwnOrganisation) && own != nil && *own == org))
}

// keyGates opens V2Keys: the caller's own keys by {List, ApiKey, Own}.
func keyGates(ctx context.Context) db.V2KeysParams {
	return db.V2KeysParams{
		Caller: account(ctx),
		Own:    granted(ctx, rules.List, rules.ApiKey, rules.Own),
	}
}

// keyShareGates opens V2KeyShares, for a list or one share: the rows by
// {List, ApiKeyShare}, the actions by {Share, ApiKey}, {Share, ApiKeyShare},
// {Manage, ApiKeyShare} and {Delete, ApiKeyShare}.
func keyShareGates(ctx context.Context) db.V2KeySharesParams {
	return db.V2KeySharesParams{
		Caller:          account(ctx),
		OwnOrganisation: ownOrganisation(ctx),

		CardOwn:          granted(ctx, rules.List, rules.ApiKeyShare, rules.PaidByOwnKey),
		CardOrganisation: granted(ctx, rules.List, rules.ApiKeyShare, rules.SharedWithOwnOrganisation),
		CardDrawn:        granted(ctx, rules.List, rules.ApiKeyShare, rules.PaidByOwnOrganisation),

		ShareOwn:          granted(ctx, rules.Share, rules.ApiKey, rules.Own),
		ShareOrganisation: granted(ctx, rules.Share, rules.ApiKeyShare, rules.SharedWithOwnOrganisation),
		ShareWorkshop:     granted(ctx, rules.Share, rules.ApiKeyShare, rules.SharedWithCurrentWorkshop),
		CurrentWorkshop:   currentWorkshop(ctx),

		RevokeOwn:          granted(ctx, rules.Manage, rules.ApiKeyShare, rules.PaidByOwnKey),
		RevokeOrganisation: granted(ctx, rules.Delete, rules.ApiKeyShare, rules.SharedWithOwnOrganisation),
		RevokeDrawn:        granted(ctx, rules.Manage, rules.ApiKeyShare, rules.PaidByOwnOrganisation),
	}
}

// mayManageKey says whether the caller may rename their key, check it, make
// it their default or delete it: {Manage, ApiKey, Own}, for a key they own.
func mayManageKey(ctx context.Context, owner uuid.UUID) bool {
	return granted(ctx, rules.Manage, rules.ApiKey, rules.Own) && account(ctx).Valid && account(ctx).UUID == owner
}

// mayCreateKey says whether the caller may add a key of their own:
// {Manage, ApiKey, Own}.
func mayCreateKey(ctx context.Context) bool {
	return granted(ctx, rules.Manage, rules.ApiKey, rules.Own) && account(ctx).Valid
}

// shareLinkGates opens V2ShareLinks, for a list or one link: the rows by
// {List, ShareLink}, the actions by {Manage, ShareLink}.
func shareLinkGates(ctx context.Context) db.V2ShareLinksParams {
	return db.V2ShareLinksParams{
		Caller:          account(ctx),
		OwnOrganisation: ownOrganisation(ctx),
		CurrentWorkshop: currentWorkshop(ctx),

		CardOwn:          granted(ctx, rules.List, rules.ShareLink, rules.Own),
		CardGame:         granted(ctx, rules.List, rules.ShareLink, rules.OfOwnGame),
		CardKey:          granted(ctx, rules.List, rules.ShareLink, rules.PaidByOwnKey),
		CardOrganisation: granted(ctx, rules.List, rules.ShareLink, rules.PaidByOwnOrganisation),
		CardWorkshop:     granted(ctx, rules.List, rules.ShareLink, rules.OfCurrentWorkshop),

		ManageOwn:          granted(ctx, rules.Manage, rules.ShareLink, rules.Own),
		ManageGame:         granted(ctx, rules.Manage, rules.ShareLink, rules.OfOwnGame),
		ManageKey:          granted(ctx, rules.Manage, rules.ShareLink, rules.PaidByOwnKey),
		ManageOrganisation: granted(ctx, rules.Manage, rules.ShareLink, rules.PaidByOwnOrganisation),
	}
}

// payerGrants says which of the shares that may pay for the caller's plays
// they may use: their workshop's key, {Use, ApiKeyShare,
// SharedWithCurrentWorkshop}; their organisation's free-use key, {Use,
// ApiKeyShare, FreeUseOfOwnOrganisation}; their own default key, {Use,
// ApiKey, Own}.
func payerGrants(ctx context.Context) (workshop, freeUse, own bool) {
	return granted(ctx, rules.Use, rules.ApiKeyShare, rules.SharedWithCurrentWorkshop),
		granted(ctx, rules.Use, rules.ApiKeyShare, rules.FreeUseOfOwnOrganisation),
		granted(ctx, rules.Use, rules.ApiKey, rules.Own)
}

// mayDrawFromWorkshopKey says whether the caller may pay for share links
// with workshop's key: {Share, ApiKeyShare, SharedWithOwnOrganisation} as
// head or staff of its organisation org, {Share, ApiKeyShare,
// SharedWithCurrentWorkshop} in it while it allows sharing.
func mayDrawFromWorkshopKey(ctx context.Context, org, workshop uuid.UUID) bool {
	c := from(ctx)
	own, current := c.Organisation(), c.CurrentWorkshop()
	return (granted(ctx, rules.Share, rules.ApiKeyShare, rules.SharedWithOwnOrganisation) && own != nil && *own == org) ||
		(granted(ctx, rules.Share, rules.ApiKeyShare, rules.SharedWithCurrentWorkshop) && current != nil && *current == workshop)
}
