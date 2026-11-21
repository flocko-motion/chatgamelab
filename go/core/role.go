package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// An account holds one role. Invites and registration give it; these change
// it directly: the admin any role, a head their members' between head and
// staff. An organisation keeps at least one head.

// knownRoles are the roles an account may hold.
var knownRoles = []string{
	string(obj.RoleAdmin), string(obj.RoleHead), string(obj.RoleStaff),
	string(obj.RoleIndividual), string(obj.RoleGuest),
}

// RoleChange is an account's new role and where it holds it.
type RoleChange struct {
	Role         string
	Organisation *uuid.UUID
	Workshop     *uuid.UUID
}

// SetRole gives user a new role and returns them. It takes {Edit,
// Membership}: Everywhere, the admin's, any role; MemberOfOwnOrganisation, a
// head's, making a member of their organisation head or staff there. Only an
// individual becomes admin, and an organisation keeps its last head (409). A
// user the caller may not read does not exist for them (ErrNotFound).
func SetRole(ctx context.Context, user uuid.UUID, ch RoleChange) (*UserCard, error) {
	target, err := ReadUser(ctx, user)
	if err != nil {
		return nil, err
	}
	if !contains(knownRoles, ch.Role) {
		return nil, refuse("unknown role %q", ch.Role)
	}
	any, withinOwn := mayChangeRole(ctx, target)
	switch {
	case any:
	case !withinOwn:
		return nil, forbidden("you may not change this user's role")
	case ch.Role != string(obj.RoleHead) && ch.Role != string(obj.RoleStaff):
		return nil, forbidden("in your organisation, you make members head or staff")
	case ch.Organisation == nil || *ch.Organisation != *target.row.organisationID:
		return nil, forbidden("you change roles within your organisation only")
	}
	// the role says where someone is: head and staff in an organisation,
	// everyone else in none; a guest is in a workshop
	inOrganisation := ch.Role == string(obj.RoleHead) || ch.Role == string(obj.RoleStaff)
	switch {
	case inOrganisation && ch.Organisation == nil:
		return nil, refuse("head and staff are in an organisation: name it")
	case !inOrganisation && ch.Organisation != nil:
		return nil, refuse("only head and staff are in an organisation")
	case ch.Role == string(obj.RoleGuest) && ch.Workshop == nil:
		return nil, refuse("a guest is in a workshop: name it")
	}
	if ch.Role == string(obj.RoleAdmin) && target.row.role != string(obj.RoleIndividual) {
		return nil, forbidden("only an individual becomes admin")
	}
	return target, changeRole(ctx, target, ch)
}

// RemoveRole takes user's role away: they become an individual of no
// organisation. It takes {Edit, Membership, Everywhere}: the admin's; a head
// removes a member from the organisation instead (RemoveMember).
func RemoveRole(ctx context.Context, user uuid.UUID) (*UserCard, error) {
	target, err := ReadUser(ctx, user)
	if err != nil {
		return nil, err
	}
	if any, _ := mayChangeRole(ctx, target); !any {
		return nil, forbidden("only an admin takes a role away")
	}
	return target, changeRole(ctx, target, RoleChange{Role: string(obj.RoleIndividual)})
}

// changeRole writes target's new role, for a caller who may: it keeps an
// organisation's last head, ends what the old role brought (the system's
// free key of a former admin, key shares to a former organisation), and
// makes ch the account's only role.
func changeRole(ctx context.Context, target *UserCard, ch RoleChange) error {
	q := store.Queries()
	oldOrg := target.row.organisationID
	leaves := oldOrg != nil && (ch.Organisation == nil || *ch.Organisation != *oldOrg)
	if target.row.role == string(obj.RoleHead) && oldOrg != nil && (ch.Role != string(obj.RoleHead) || leaves) {
		heads, err := q.V2CountHeadsByInstitution(ctx, uuid.NullUUID{UUID: *oldOrg, Valid: true})
		if err != nil {
			return err
		}
		if heads <= 1 {
			return conflict(obj.ErrCodeLastHead, "an organisation needs at least one head")
		}
	}
	if target.row.role == string(obj.RoleAdmin) && ch.Role != string(obj.RoleAdmin) {
		// v1's system settings; they move into the core when ported
		if err := store.Queries().V2ClearFreeUseKeyOf(ctx, target.row.id); err != nil {
			return err
		}
	}
	err := store.InTx(ctx, func(q *db.Queries) error {
		if leaves && target.row.role != string(obj.RoleGuest) {
			if err := q.V2DeleteApiKeySharesByOwnerForInstitution(ctx, db.V2DeleteApiKeySharesByOwnerForInstitutionParams{
				UserID: target.row.id, InstitutionID: nullUUID(oldOrg),
			}); err != nil {
				return err
			}
			if err := q.V2DeleteApiKeySharesByOwnerForInstitutionWorkshops(ctx, db.V2DeleteApiKeySharesByOwnerForInstitutionWorkshopsParams{
				UserID: target.row.id, InstitutionID: *oldOrg,
			}); err != nil {
				return err
			}
		}
		if err := q.V2DeleteUserRoles(ctx, target.row.id); err != nil {
			return err
		}
		_, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{
			UserID:        target.row.id,
			Role:          ch.Role,
			InstitutionID: nullUUID(ch.Organisation),
			WorkshopID:    nullUUID(ch.Workshop),
		})
		return err
	})
	if err != nil {
		return err
	}
	log.Info("role changed", "user_id", target.row.id, "role", ch.Role, "by", account(ctx).UUID)
	return nil
}
