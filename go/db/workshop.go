// package: db / database access and repository layer
// type:    data
// job:     what v1 code still reads and writes of workshops: reading one for v1 callers, its key (until #451), the delete cascade; the rest is the core's (core/workshop*.go).
// limits:  does not list workshops or fetch member/invite collections (-> workshop_list.go).
package db

import (
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// DeleteWorkshopCascade deletes a workshop with its guests, for the core,
// which has checked that the caller may (core.DeleteWorkshop): v1's cascade
// until guests and share links are ported. Its other games stay with their
// creators.
func DeleteWorkshopCascade(ctx context.Context, id uuid.UUID) error {
	wsNullUUID := uuid.NullUUID{UUID: id, Valid: true}

	// Clean up Guests: collect user IDs, delete roles and user accounts
	guestUserIDs, _ := queries().GetGuestUserIDsByWorkshopID(ctx, wsNullUUID)
	_ = queries().DeleteUserRolesByWorkshopID(ctx, wsNullUUID)
	for _, uid := range guestUserIDs {
		// Delete each Guest's account
		_ = DeleteUser(ctx, uid)
	}

	// Unlink remaining games from this workshop (member games stay with their creator)
	_ = queries().UnlinkGamesFromWorkshop(ctx, wsNullUUID)

	if err := queries().DeleteWorkshop(ctx, id); err != nil {
		return obj.ErrServerError("failed to delete workshop")
	}
	return nil
}
