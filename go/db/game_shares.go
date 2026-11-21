// package: db / database access and repository layer
// type:    data
// job:     game share links, public sponsorship, and guest-user data.
// limits:  does not manage games (-> game.go) or game plays (-> game).
package db

import (
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// DeleteGuestDataByShareID removes the guest users created via a game share link.
func DeleteGuestDataByShareID(ctx context.Context, shareID uuid.UUID) error {
	return queries().DeleteGuestUsersByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true})
}

// DeleteGuestDataByGameID removes all guest data for all shares of a game.
// v1's share links: the core calls it until share links are ported (#451).
func DeleteGuestDataByGameID(ctx context.Context, gameID uuid.UUID) error {
	shares, err := queries().GetGameSharesByGameID(ctx, gameID)
	if err != nil {
		return nil // no shares = nothing to clean up
	}
	for _, gs := range shares {
		_ = DeleteGuestDataByShareID(ctx, gs.ID)
	}
	return nil
}

// dbGameShareToObj converts a DB game_share row to an obj.GameShare.
func dbGameShareToObj(gs GameShare) *obj.GameShare {
	result := &obj.GameShare{
		ID:            gs.ID,
		GameID:        gs.GameID,
		Token:         gs.Token,
		ApiKeyShareID: gs.ApiKeyShareID,
		Remaining:     nullInt32ToIntPtr(gs.Remaining),
		CreatedAt:     gs.CreatedAt,
	}
	if gs.InstitutionID.Valid {
		result.InstitutionID = &gs.InstitutionID.UUID
	}
	if gs.WorkshopID.Valid {
		result.WorkshopID = &gs.WorkshopID.UUID
	}
	if gs.AiQualityTier.Valid {
		result.AiQualityTier = &gs.AiQualityTier.String
	}
	if gs.CreatedBy.Valid {
		result.CreatedBy = &gs.CreatedBy.UUID
	}
	return result
}
