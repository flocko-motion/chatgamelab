// package: db / database access and repository layer
// type:    data
// job:     game share links, public sponsorship, and guest-user data.
// limits:  does not manage games (-> game.go) or game plays (-> game).
package db

import (
	"cgl/functional"
	"cgl/obj"
	"cgl/tokenlock"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// insertGameShare is CreateGameShare with publicPage marking a link of the public workshop page.
func insertGameShare(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, sourceShareID uuid.UUID, institutionID, workshopID *uuid.UUID, maxSessions *int, aiQualityTier *string, publicPage bool) (*obj.GameShare, error) {
	// Verify the source share exists and the user is authorized to use it
	share, err := queries().GetApiKeyShareByID(ctx, sourceShareID)
	if err != nil {
		return nil, obj.ErrNotFound("api key share not found")
	}
	// For workshop shares, the route handler already verified workshop access and sharing permissions.
	// Only check personal sponsoring permissions for non-workshop shares.
	if workshopID == nil {
		if err := canUseShareForSponsoring(ctx, userID, share); err != nil {
			return nil, err
		}
	}

	// Verify the key hasn't been proven to NOT work
	if share.KeyLastUsageSuccess.Valid && !share.KeyLastUsageSuccess.Bool {
		return nil, obj.ErrValidation("api key must be working before it can be used for sharing")
	}

	// The share drawn for the link from its source (#451): taking the
	// source back stops the link.
	drawnID, err := queries().V2CreateKeyShare(ctx, V2CreateKeyShareParams{
		CreatedBy:     userID,
		ApiKeyID:      share.ApiKeyID,
		GameID:        uuid.NullUUID{UUID: gameID, Valid: true},
		SourceShareID: uuid.NullUUID{UUID: sourceShareID, Valid: true},
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to create game-scoped share")
	}
	gameScopedShareID := &drawnID

	// Generate a secure token for the share link
	token, err := functional.GenerateSecureToken(20)
	if err != nil {
		_ = queries().DeleteApiKeyShare(ctx, *gameScopedShareID)
		return nil, obj.ErrServerError("failed to generate share token")
	}

	// Create the game_share row
	gs, err := queries().CreateGameShare(ctx, CreateGameShareParams{
		GameID:        gameID,
		Token:         token,
		ApiKeyShareID: *gameScopedShareID,
		InstitutionID: uuidPtrToNullUUID(institutionID),
		WorkshopID:    uuidPtrToNullUUID(workshopID),
		Remaining:     intPtrToNullInt32(maxSessions),
		AiQualityTier: stringPtrToNullString(aiQualityTier),
		CreatedBy:     uuid.NullUUID{UUID: userID, Valid: true},
		PublicPage:    publicPage,
	})
	if err != nil {
		_ = queries().DeleteApiKeyShare(ctx, *gameScopedShareID)
		return nil, obj.ErrServerError("failed to create game share")
	}

	return dbGameShareToObj(gs), nil
}

// GetGameShareByToken loads a game share by its token.
func GetGameShareByToken(ctx context.Context, token string) (*obj.GameShare, error) {
	gs, err := queries().GetGameShareByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tokenlock.Fail()
		}
		return nil, obj.ErrNotFound("share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GetGameShareByID loads a game share by its ID.
func GetGameShareByID(ctx context.Context, shareID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().GetGameShareByID(ctx, shareID)
	if err != nil {
		return nil, obj.ErrNotFound("share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GetGameSharesByGameIDAndCreator returns shares for a game created by a specific user.
func GetGameSharesByGameIDAndCreator(ctx context.Context, gameID uuid.UUID, userID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndCreator(ctx, GetGameSharesByGameIDAndCreatorParams{
		GameID:    gameID,
		CreatedBy: uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetGameSharesByGameIDAndWorkshop returns shares for a game in a specific workshop.
func GetGameSharesByGameIDAndWorkshop(ctx context.Context, gameID uuid.UUID, workshopID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndWorkshop(ctx, GetGameSharesByGameIDAndWorkshopParams{
		GameID:     gameID,
		WorkshopID: uuid.NullUUID{UUID: workshopID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetGameSharesByGameIDAndInstitution returns org-level shares (non-workshop) for a game.
func GetGameSharesByGameIDAndInstitution(ctx context.Context, gameID uuid.UUID, institutionID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndInstitution(ctx, GetGameSharesByGameIDAndInstitutionParams{
		GameID:        gameID,
		InstitutionID: uuid.NullUUID{UUID: institutionID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetWorkshopGameShare finds an existing workshop share for a game (for reuse).
func GetWorkshopGameShare(ctx context.Context, gameID uuid.UUID, workshopID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().GetWorkshopGameShare(ctx, GetWorkshopGameShareParams{
		GameID:     gameID,
		WorkshopID: uuid.NullUUID{UUID: workshopID, Valid: true},
	})
	if err != nil {
		return nil, obj.ErrNotFound("workshop share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GameShareWithGame represents a game share enriched with game name.
type GameShareWithGame struct {
	obj.GameShare
	GameName string
}

// dbGameShareToObjFromJoin converts individual fields (from JOIN query results) to obj.GameShare.
func dbGameShareToObjFromJoin(id, gameID uuid.UUID, token string, apiKeyShareID uuid.UUID, institutionID, workshopID uuid.NullUUID, remaining sql.NullInt32, aiQualityTier sql.NullString, createdBy uuid.NullUUID, createdAt time.Time) *obj.GameShare {
	gs := &obj.GameShare{
		ID:            id,
		GameID:        gameID,
		Token:         token,
		ApiKeyShareID: apiKeyShareID,
		CreatedAt:     createdAt,
	}
	if institutionID.Valid {
		gs.InstitutionID = &institutionID.UUID
	}
	if workshopID.Valid {
		gs.WorkshopID = &workshopID.UUID
	}
	if remaining.Valid {
		r := int(remaining.Int32)
		gs.Remaining = &r
	}
	if aiQualityTier.Valid {
		gs.AiQualityTier = &aiQualityTier.String
	}
	if createdBy.Valid {
		gs.CreatedBy = &createdBy.UUID
	}
	return gs
}

// DeleteGameShare deletes a game share and cleans up associated guest data and the game-scoped API key share.
func DeleteGameShare(ctx context.Context, shareID uuid.UUID) error {
	gs, err := queries().GetGameShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	// Clean up guest data linked to this share
	_ = DeleteGuestDataByShareID(ctx, shareID)

	// Delete the game_share row
	if err := queries().DeleteGameShare(ctx, shareID); err != nil {
		return err
	}

	// Delete the game-scoped API key share
	_ = queries().DeleteApiKeyShare(ctx, gs.ApiKeyShareID)

	return nil
}

// DecrementGameShareRemaining atomically decrements the remaining counter on a game share.
func DecrementGameShareRemaining(ctx context.Context, shareID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().DecrementGameShareRemaining(ctx, shareID)
	if err != nil {
		return nil, obj.ErrForbidden("share link has reached its play limit")
	}
	return dbGameShareToObj(gs), nil
}

// CreateGuestUser creates an anonymous user for guest plays.
// shareID links the guest to the game_share for cleanup on revoke.
func CreateGuestUser(ctx context.Context, userID uuid.UUID, name string, shareID uuid.UUID) error {
	_, err := queries().CreateGuestUser(ctx, CreateGuestUserParams{
		ID:             userID,
		Name:           userName(name),
		PrivateShareID: uuid.NullUUID{UUID: shareID, Valid: true},
	})
	return err
}

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
