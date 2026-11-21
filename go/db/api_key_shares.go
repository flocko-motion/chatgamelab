// package: db / database access and repository layer
// type:    data
// job:     create, delete, and look up API key shares (per-user and per-institution).
// limits:  does not manage API keys themselves (-> api_keys.go) or run join/availability queries (-> api_key_shares_query.go).
package db

import (
	"cgl/functional"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

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
