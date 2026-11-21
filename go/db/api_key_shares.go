// package: db / database access and repository layer
// type:    data
// job:     create, delete, and look up API key shares (per-user and per-institution).
// limits:  does not manage API keys themselves (-> api_keys.go) or run join/availability queries (-> api_key_shares_query.go).
package db

import (
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"time"

	"github.com/google/uuid"
)

// createApiKeyShareInternal creates a share without ownership verification (for internal use)
func createApiKeyShareInternal(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID, gameID *uuid.UUID) (*uuid.UUID, error) {
	now := time.Now()
	arg := CreateApiKeyShareParams{
		CreatedBy:     uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:     now,
		ModifiedBy:    uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:    now,
		ApiKeyID:      apiKeyID,
		UserID:        uuidPtrToNullUUID(targetUserID),
		WorkshopID:    uuidPtrToNullUUID(workshopID),
		InstitutionID: uuidPtrToNullUUID(institutionID),
		GameID:        uuidPtrToNullUUID(gameID),
	}

	result, err := queries().CreateApiKeyShare(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &result.ID, nil
}

// GetApiKeyShareByID returns an API key share by its ID, including the full API key.
func GetApiKeyShareByID(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) (*obj.ApiKeyShare, error) {
	s, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, obj.ErrNotFound("share not found")
	}

	// Check permission - user must have read access to the API key
	// First check via standard canAccessApiKey
	if err := canAccessApiKey(ctx, userID, OpRead, s.ApiKeyID, s.KeyOwnerID, nil, nil, nil); err != nil {
		// Game-scoped shares (sponsorships) are accessible by any user
		if s.GameID.Valid {
			log.Debug("access granted via game sponsorship share", "share_id", shareID, "user_id", userID, "game_id", s.GameID.UUID)
		} else {
			// Also check if this share is a workshop's default API key
			// and the user is a member of that workshop
			canAccess, checkErr := queries().CanUserAccessShareViaWorkshopDefault(ctx, CanUserAccessShareViaWorkshopDefaultParams{
				DefaultApiKeyShareID: uuid.NullUUID{UUID: shareID, Valid: true},
				UserID:               userID,
			})
			if checkErr != nil || !canAccess {
				return nil, err // Return original error
			}
			log.Debug("access granted via workshop default API key", "share_id", shareID, "user_id", userID)
		}
	}
	share := &obj.ApiKeyShare{
		ID: s.ID,
		Meta: obj.Meta{
			CreatedBy:  s.CreatedBy,
			CreatedAt:  &s.CreatedAt,
			ModifiedBy: s.ModifiedBy,
			ModifiedAt: &s.ModifiedAt,
		},
		ApiKeyID: s.ApiKeyID,
		ApiKey: &obj.ApiKey{
			ID:               s.KeyID,
			UserID:           s.KeyOwnerID,
			UserName:         s.KeyOwnerName.String,
			Name:             s.KeyName,
			Platform:         s.KeyPlatform,
			Key:              s.KeyKey,
			KeyShortened:     functional.ShortenLeft(s.KeyKey, apiKeyShortenLength),
			IsDefault:        s.KeyIsDefault,
			LastUsageSuccess: sqlNullBoolToMaybeBool(s.KeyLastUsageSuccess),
			LastErrorCode:    sqlNullStringToMaybeString(s.KeyLastErrorCode),
		},
	}
	if s.UserID.Valid {
		share.User = &obj.User{ID: s.UserID.UUID}
	}
	if s.WorkshopID.Valid {
		share.Workshop = &obj.Workshop{ID: s.WorkshopID.UUID}
	}
	if s.InstitutionID.Valid {
		share.Institution = &obj.Institution{ID: s.InstitutionID.UUID}
	}
	if s.GameID.Valid {
		share.Game = &obj.Game{ID: s.GameID.UUID}
	}
	return share, nil
}

// GetApiKeySharesByUser returns all API key shares accessible to a user
func GetApiKeySharesByUser(ctx context.Context, userID uuid.UUID) ([]obj.ApiKeyShare, error) {
	// Check permission - users can list their own keys plus shared keys
	if err := canAccessApiKey(ctx, userID, OpList, uuid.Nil, uuid.Nil, nil, nil, nil); err != nil {
		return nil, err
	}

	sharedKeys, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, obj.ErrServerError("failed to get api key shares")
	}

	result := make([]obj.ApiKeyShare, 0, len(sharedKeys))
	for _, s := range sharedKeys {
		// Skip game sponsorship shares — they are shown via linkedShares in the detail view
		if s.GameID.Valid {
			continue
		}
		share := obj.ApiKeyShare{
			ID: s.ID,
			Meta: obj.Meta{
				CreatedBy:  s.CreatedBy,
				CreatedAt:  &s.CreatedAt,
				ModifiedBy: s.ModifiedBy,
				ModifiedAt: &s.ModifiedAt,
			},
			ApiKeyID: s.ApiKeyID,
			ApiKey: &obj.ApiKey{
				ID:               s.ApiKeyID,
				UserID:           s.OwnerID,
				UserName:         s.OwnerName.String,
				Name:             s.ApiKeyName,
				Platform:         s.ApiKeyPlatform,
				Key:              s.ApiKeyKey,
				KeyShortened:     functional.ShortenLeft(s.ApiKeyKey, apiKeyShortenLength),
				IsDefault:        s.ApiKeyIsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(s.ApiKeyLastUsageSuccess),
				LastErrorCode:    sqlNullStringToMaybeString(s.ApiKeyLastErrorCode),
			},
		}
		result = append(result, share)
	}

	return result, nil
}
