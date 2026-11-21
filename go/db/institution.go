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

// GetInstitutionName retrieves just the institution name by ID (no permission check, for display only)
func GetInstitutionName(ctx context.Context, id uuid.UUID) (string, error) {
	result, err := queries().GetInstitutionByID(ctx, id)
	if err != nil {
		return "", obj.ErrNotFound("institution not found")
	}
	return result.Name, nil
}

// GetInstitutionFreeUseTier is the AI quality tier of an institution's free
// use; nil: the server's default. No permission check: resolving who pays
// for a session reads it for any player of the institution (until #451).
func GetInstitutionFreeUseTier(ctx context.Context, id uuid.UUID) (*string, error) {
	result, err := queries().GetInstitutionByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("institution not found")
	}
	if !result.FreeUseAiQualityTier.Valid {
		return nil, nil
	}
	return &result.FreeUseAiQualityTier.String, nil
}

// SetInstitutionFreeUseApiKeyShare sets or clears the free-use API key share for an institution.
// Permission check: head of the institution or admin.
func SetInstitutionFreeUseApiKeyShare(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID, shareID *uuid.UUID) error {
	// Check permission - must be head or admin
	if err := canAccessInstitution(ctx, userID, OpUpdate, &institutionID); err != nil {
		return err
	}

	return queries().SetInstitutionFreeUseApiKeyShare(ctx, SetInstitutionFreeUseApiKeyShareParams{
		ID:                   institutionID,
		FreeUseApiKeyShareID: uuidPtrToNullUUID(shareID),
	})
}

// DeleteInstitutionCascade deletes an institution and all its data, for the
// core, which has checked that deletedBy may (core.DeleteOrganisation).
// Cascade: workshops (with their Guests and games), member users (with their data),
// invites, API key shares, free-use key ref.
func DeleteInstitutionCascade(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {

	// 1. Delete all workshops (cascades to their Guests, games, etc.)
	workshopIDs, _ := queries().GetWorkshopIDsByInstitution(ctx, id)
	for _, wsID := range workshopIDs {
		// Use deletedBy (admin) to bypass per-workshop permission checks
		_ = DeleteWorkshop(ctx, wsID, deletedBy)
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
