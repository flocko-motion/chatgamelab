package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// A membership is a user's role in an organisation: individual, staff or
// head, or guest in one of its workshops. Its members are the users holding
// one of the first three; guests are not listed among them. Removing a
// membership leaves the user their account, as an individual of no
// organisation.

// ListMembers lists the members of organisation id the caller may list:
// {List, UserCard}, narrowed to the organisation. An organisation the caller
// may not read does not exist for them (ErrNotFound).
func ListMembers(ctx context.Context, id uuid.UUID) ([]*UserCard, error) {
	org, err := ReadOrganisation(ctx, id)
	if err != nil {
		return nil, err
	}
	return org.Members(), nil
}

// listMembers is the members of organisation id the caller may list, for a
// card already through its gate.
func listMembers(ctx context.Context, id uuid.UUID) ([]*UserCard, error) {
	rows, err := store.Queries().V2Users(ctx, listMembersGates(ctx, id))
	if err != nil {
		return nil, err
	}
	members := make([]*UserCard, len(rows))
	for i, r := range rows {
		members[i] = userCardOf(ctx, r)
	}
	return members, nil
}

// RemoveMember removes user from organisation id: {Delete, Membership}. A
// head removes members and guests, staff remove guests, a member leaves
// (Own), an admin removes anyone; an organisation's last head stays. A user
// the caller may not read, or one not of this organisation, does not exist
// for them (ErrNotFound). The user's key shares to the organisation and its
// workshops go; they become an individual of no organisation.
func RemoveMember(ctx context.Context, id, user uuid.UUID) error {
	if _, err := ReadOrganisation(ctx, id); err != nil {
		return err
	}
	member, err := ReadUser(ctx, user)
	if err != nil {
		return err
	}
	if member.row.organisationID == nil || *member.row.organisationID != id {
		return ErrNotFound
	}
	if !mayRemoveMember(ctx, member) {
		return forbidden("you may not remove this member")
	}
	if member.row.role == "head" {
		heads, err := store.Queries().V2CountHeadsByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
		if err != nil {
			return err
		}
		if heads <= 1 {
			return conflict(obj.ErrCodeLastHead, "an organisation needs at least one head")
		}
	}

	err = store.InTx(ctx, func(q *db.Queries) error {
		if err := q.V2DeleteApiKeySharesByOwnerForInstitution(ctx, db.V2DeleteApiKeySharesByOwnerForInstitutionParams{
			UserID: user, InstitutionID: uuid.NullUUID{UUID: id, Valid: true},
		}); err != nil {
			return err
		}
		if err := q.V2DeleteApiKeySharesByOwnerForInstitutionWorkshops(ctx, db.V2DeleteApiKeySharesByOwnerForInstitutionWorkshopsParams{
			UserID: user, InstitutionID: id,
		}); err != nil {
			return err
		}
		if err := q.V2DeleteUserRoles(ctx, user); err != nil {
			return err
		}
		_, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{UserID: user, Role: string(obj.RoleIndividual)})
		return err
	})
	if err != nil {
		return err
	}
	log.Info("member removed", "institution_id", id, "user_id", user, "by", account(ctx).UUID)
	return nil
}
