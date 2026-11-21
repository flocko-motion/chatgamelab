// package: db / database access and repository layer
// type:    data
// job:     what v1 code still reads and writes of institutions; the rest is the core's (core/organisation*.go).
// limits:  does not define SQL queries (-> db/queries) or expose HTTP endpoints (-> api).
package db

import (
	"context"

	"cgl/legal"
	"cgl/obj"

	"github.com/google/uuid"
)

// DeleteInstitutionCascade deletes an institution and all its data, for the
// core, which has checked that deletedBy may (core.DeleteOrganisation).
// Cascade: workshops (with their Guests and games), member users (with their data),
// invites, API key shares, free-use key ref.
func DeleteInstitutionCascade(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {

	// 1. Delete all workshops (cascades to their Guests, games, etc.)
	workshopIDs, _ := queries().GetWorkshopIDsByInstitution(ctx, id)
	for _, wsID := range workshopIDs {
		_ = DeleteWorkshopCascade(ctx, wsID)
	}

	// 2. Delete the remaining Guests
	guestIDs, _ := queries().GetGuestUserIDsByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	for _, guestID := range guestIDs {
		_ = DeleteUser(ctx, guestID)
	}

	// 3. Collect the other members (head/staff/individual) before removing roles
	memberIDs, _ := queries().GetNonGuestUserIDsByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})

	// 4. Clean up remaining FK references before hard-delete
	_ = queries().DeleteInvitesByInstitution(ctx, id)
	_ = queries().DeleteApiKeySharesByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	_ = queries().DeleteUserRolesByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	_ = queries().HardDeleteWorkshopsByInstitution(ctx, id)
	_ = queries().ClearInstitutionFreeUseApiKeyShare(ctx, id)

	// 5. Assign individual role to former members so they aren't left without a role
	for _, memberID := range memberIDs {
		_ = assignDefaultIndividualRole(ctx, memberID)
	}

	// 6. Hard-delete the institution
	if err := queries().HardDeleteInstitution(ctx, id); err != nil {
		return obj.ErrServerError("failed to delete institution")
	}
	return nil
}

// IsWorkshopInstitutionSuspended reports whether the workshop's institution
// is suspended, by an admin or for want of a consent; an error counts as not
// suspended, as v1 has it.
func IsWorkshopInstitutionSuspended(ctx context.Context, workshopID uuid.UUID) bool {
	declaration := legal.CurrentDeclaration()
	suspended, err := queries().IsWorkshopInstitutionSuspended(ctx, IsWorkshopInstitutionSuspendedParams{
		WorkshopID: workshopID, Version: declaration.Version, Hash: declaration.Hash,
	})
	return err == nil && suspended
}

// IsInstitutionSuspended reports whether the institution is suspended, by an
// admin or for want of a consent; an error counts as not suspended, as v1 has
// it.
func IsInstitutionSuspended(ctx context.Context, id uuid.UUID) bool {
	declaration := legal.CurrentDeclaration()
	suspended, err := queries().IsInstitutionSuspended(ctx, IsInstitutionSuspendedParams{
		ID: id, Version: declaration.Version, Hash: declaration.Hash,
	})
	return err == nil && suspended
}
