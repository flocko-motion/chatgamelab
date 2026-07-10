// package: db / database access and repository layer
// type:    data
// job:     game share links, public sponsorship, and guest-user data.
// limits:  does not manage games (-> game.go) or sessions (-> game_sessions.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// SetGamePublicSponsorship sets a public sponsorship on a game.
// Creates a game-scoped API key share and links it to the game.
// The user must own the API key and have permission to update the game.
func SetGamePublicSponsorship(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, apiKeyShareID uuid.UUID) error {
	// Load game and check update permission
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if err := canAccessGame(ctx, userID, OpUpdate, game, nil); err != nil {
		return err
	}

	// Verify the share exists and the user is authorized to use it
	share, err := queries().GetApiKeyShareByID(ctx, apiKeyShareID)
	if err != nil {
		return obj.ErrNotFound("api key share not found")
	}
	if err := canUseShareForSponsoring(ctx, userID, share); err != nil {
		return err
	}

	// Verify the key has been proven to work (last_usage_success must be true)
	if share.KeyLastUsageSuccess.Valid && !share.KeyLastUsageSuccess.Bool {
		return obj.ErrValidation("api key must be proven to work before sponsoring")
	}

	// Remove any existing sponsorship and its game-scoped shares first
	if game.PublicSponsoredApiKeyShareID != nil {
		if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
			return obj.ErrServerError("failed to clear existing sponsorship")
		}
		if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
			log.Debug("failed to delete old game-scoped shares", "game_id", gameID, "error", err)
		}
	}

	// Create a game-scoped share for this sponsorship
	sponsorShareID, err := createApiKeyShareInternal(ctx, userID, share.ApiKeyID, &userID, nil, nil, &gameID)
	if err != nil {
		return obj.ErrServerError("failed to create sponsorship share")
	}

	// Set the sponsor share on the game
	if err := queries().SetGamePublicSponsor(ctx, db.SetGamePublicSponsorParams{
		ID:                           gameID,
		PublicSponsoredApiKeyShareID: uuid.NullUUID{UUID: *sponsorShareID, Valid: true},
	}); err != nil {
		return obj.ErrServerError("failed to set game sponsorship")
	}

	return nil
}

// ClearGamePublicSponsorship removes the public sponsorship from a game.
// Also deletes the game-scoped API key share.
// Allowed by: game owner (OpUpdate) OR the API key owner who sponsors the game.
func ClearGamePublicSponsorship(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}

	if game.PublicSponsoredApiKeyShareID == nil {
		return nil // Already no sponsorship
	}

	// Allow if user can update the game (game owner / higher role)
	accessErr := canAccessGame(ctx, userID, OpUpdate, game, nil)
	if accessErr != nil {
		// Also allow if user owns the API key behind the sponsorship
		share, shareErr := queries().GetApiKeyShareByID(ctx, *game.PublicSponsoredApiKeyShareID)
		if shareErr != nil {
			return accessErr
		}
		key, keyErr := queries().GetApiKeyByID(ctx, share.ApiKeyID)
		if keyErr != nil || key.UserID != userID {
			return accessErr
		}
	}

	// Clear the sponsor on the game
	if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
		return obj.ErrServerError("failed to clear game sponsorship")
	}

	// Delete game-scoped shares for this game
	if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
		log.Debug("failed to delete game-scoped shares", "game_id", gameID, "error", err)
	}

	return nil
}

// ClearGamePublicSponsorshipByShareID removes sponsorship if the given share is the sponsor.
// Used when auto-removing sponsorship on key failure.
func ClearGamePublicSponsorshipByShareID(ctx context.Context, gameID uuid.UUID, shareID uuid.UUID) error {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game.PublicSponsoredApiKeyShareID == nil || *game.PublicSponsoredApiKeyShareID != shareID {
		return nil // Not the sponsor
	}

	if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
		return err
	}

	// Delete game-scoped shares for this game
	if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
		log.Debug("failed to delete game-scoped shares", "game_id", gameID, "error", err)
	}

	return nil
}

// CreateGameShare creates a game share link with a game-scoped API key share.
// The sourceShareID is the user's personal/workshop share that will be cloned into a game-scoped share.
func CreateGameShare(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, sourceShareID uuid.UUID, institutionID, workshopID *uuid.UUID, maxSessions *int, aiQualityTier *string) (*obj.GameShare, error) {
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

	// Create a game-scoped API key share (accessible by uuid.Nil in guest play flow)
	gameScopedShareID, err := createApiKeyShareInternal(ctx, userID, share.ApiKeyID, &userID, nil, nil, &gameID)
	if err != nil {
		return nil, obj.ErrServerError("failed to create game-scoped share")
	}

	// Generate a secure token for the share link
	token, err := functional.GenerateSecureToken(20)
	if err != nil {
		return nil, obj.ErrServerError("failed to generate share token")
	}

	// Create the game_share row
	gs, err := queries().CreateGameShare(ctx, db.CreateGameShareParams{
		GameID:        gameID,
		Token:         token,
		ApiKeyShareID: *gameScopedShareID,
		InstitutionID: uuidPtrToNullUUID(institutionID),
		WorkshopID:    uuidPtrToNullUUID(workshopID),
		Remaining:     intPtrToNullInt32(maxSessions),
		AiQualityTier: stringPtrToNullString(aiQualityTier),
		CreatedBy:     uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to create game share")
	}

	return dbGameShareToObj(gs), nil
}

// GetGameShareByToken loads a game share by its token.
func GetGameShareByToken(ctx context.Context, token string) (*obj.GameShare, error) {
	gs, err := queries().GetGameShareByToken(ctx, token)
	if err != nil {
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
	rows, err := queries().GetGameSharesByGameIDAndCreator(ctx, db.GetGameSharesByGameIDAndCreatorParams{
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
	rows, err := queries().GetGameSharesByGameIDAndWorkshop(ctx, db.GetGameSharesByGameIDAndWorkshopParams{
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
	rows, err := queries().GetGameSharesByGameIDAndInstitution(ctx, db.GetGameSharesByGameIDAndInstitutionParams{
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
	gs, err := queries().GetWorkshopGameShare(ctx, db.GetWorkshopGameShareParams{
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

// GetGameSharesWithGameByApiKeyID returns game shares (with game name) for all shares of an API key.
func GetGameSharesWithGameByApiKeyID(ctx context.Context, apiKeyID uuid.UUID) ([]GameShareWithGame, error) {
	rows, err := queries().GetGameSharesWithGameByApiKeyID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	result := make([]GameShareWithGame, len(rows))
	for i, r := range rows {
		result[i] = GameShareWithGame{
			GameShare: *dbGameShareToObjFromJoin(r.ID, r.GameID, r.Token, r.ApiKeyShareID, r.InstitutionID, r.WorkshopID, r.Remaining, r.AiQualityTier, r.CreatedBy, r.CreatedAt),
			GameName:  r.GameName,
		}
	}
	return result, nil
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

// UpdateGameShare updates the remaining sessions and AI quality tier on a game share.
func UpdateGameShare(ctx context.Context, shareID uuid.UUID, remaining *int, aiQualityTier *string) (*obj.GameShare, error) {
	var nullRemaining sql.NullInt32
	if remaining != nil {
		nullRemaining = sql.NullInt32{Int32: int32(*remaining), Valid: true}
	}
	gs, err := queries().UpdateGameShare(ctx, db.UpdateGameShareParams{
		ID:            shareID,
		Remaining:     nullRemaining,
		AiQualityTier: stringPtrToNullString(aiQualityTier),
	})
	if err != nil {
		return nil, err
	}
	return dbGameShareToObj(gs), nil
}

// DecrementGameShareRemaining atomically decrements the remaining counter on a game share.
func DecrementGameShareRemaining(ctx context.Context, shareID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().DecrementGameShareRemaining(ctx, shareID)
	if err != nil {
		return nil, obj.ErrForbidden("share link has reached its play limit")
	}
	return dbGameShareToObj(gs), nil
}

// CreateGuestUser creates an anonymous user for guest play sessions.
// shareID links the guest to the game_share for cleanup on revoke.
func CreateGuestUser(ctx context.Context, userID uuid.UUID, name string, shareID uuid.UUID) error {
	_, err := queries().CreateGuestUser(ctx, db.CreateGuestUserParams{
		ID:             userID,
		Name:           name,
		PrivateShareID: uuid.NullUUID{UUID: shareID, Valid: true},
	})
	return err
}

// DeleteGuestDataByShareID removes all guest users, their sessions, and messages
// that were created via a game share link.
// Must delete in order: messages → sessions → users (FK constraints).
func DeleteGuestDataByShareID(ctx context.Context, shareID uuid.UUID) error {
	if err := queries().DeleteGuestSessionMessagesByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true}); err != nil {
		return err
	}
	if err := queries().DeleteGuestSessionsByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true}); err != nil {
		return err
	}
	return queries().DeleteGuestUsersByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true})
}

// DeleteGuestDataByGameID removes all guest data for all shares of a game.
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
func dbGameShareToObj(gs db.GameShare) *obj.GameShare {
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
