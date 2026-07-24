// package: db / database access and repository layer
// type:    data
// job:     create, delete, default-selection, and metadata updates for API keys.
// limits:  does not manage shares (-> api_key_shares.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/game/ai"
	"cgl/obj"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const apiKeyShortenLength = 6

// GetApiKeyByID returns an API key by its ID (no permission check).
func GetApiKeyByID(ctx context.Context, apiKeyID uuid.UUID) (*obj.ApiKey, error) {
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, obj.ErrNotFound("api key not found")
	}
	return &obj.ApiKey{
		ID:       key.ID,
		Name:     key.Name,
		Platform: key.Platform,
		Key:      key.Key,
	}, nil
}

func createApiKeyAndSelfShare(ctx context.Context, userID uuid.UUID, name, platform, key string) (apiKeyID uuid.UUID, shareID uuid.UUID, err error) {
	if !ai.IsValidApiKeyPlatform(platform) {
		return uuid.Nil, uuid.Nil, obj.ErrInvalidPlatformf("unknown platform: %s", platform)
	}

	// Auto-set as default if this is the user's first key
	count, err := queries().CountApiKeysByUser(ctx, userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, obj.ErrServerError("failed to count user keys")
	}
	isDefault := count == 0

	now := time.Now()
	arg := db.CreateApiKeyParams{
		CreatedBy:  uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:  now,
		ModifiedBy: uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt: now,
		UserID:     userID,
		Name:       name,
		Platform:   platform,
		Key:        key,
		IsDefault:  isDefault,
	}
	result, err := queries().CreateApiKey(ctx, arg)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// Create a self-share so the user can access their own key via the shares API
	selfShareID, err := createApiKeyShareInternal(ctx, userID, result.ID, &userID, nil, nil, nil)
	if err != nil {
		return uuid.Nil, uuid.Nil, obj.ErrServerError("failed to create self-share")
	}
	if selfShareID == nil {
		return uuid.Nil, uuid.Nil, obj.ErrServerError("failed to create self-share")
	}

	return result.ID, *selfShareID, nil
}

// CreateApiKey creates a new API key for a user with a self-share
func CreateApiKey(ctx context.Context, userID uuid.UUID, name, platform, key string) (*uuid.UUID, error) {
	// Check permission
	if err := canAccessApiKey(ctx, userID, OpCreate, uuid.Nil, uuid.Nil, nil, nil, nil); err != nil {
		return nil, err
	}

	apiKeyID, _, err := createApiKeyAndSelfShare(ctx, userID, name, platform, key)
	if err != nil {
		return nil, err
	}
	return &apiKeyID, nil
}

// CreateApiKeyWithSelfShare creates a new API key and returns the user's self-share.
func CreateApiKeyWithSelfShare(ctx context.Context, userID uuid.UUID, name, platform, key string) (*obj.ApiKeyShare, error) {
	// Check permission
	if err := canAccessApiKey(ctx, userID, OpCreate, uuid.Nil, uuid.Nil, nil, nil, nil); err != nil {
		return nil, err
	}

	_, shareID, err := createApiKeyAndSelfShare(ctx, userID, name, platform, key)
	if err != nil {
		return nil, err
	}
	return GetApiKeyShareByID(ctx, userID, shareID)
}

// DeleteApiKey deletes the underlying API key and all its shares (owner only).
func DeleteApiKey(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return obj.ErrNotFound("api key not found")
	}

	// Check permission
	if err := canAccessApiKey(ctx, userID, OpDelete, key.ID, key.UserID, nil, nil, nil); err != nil {
		return err
	}

	// Clear session api_key_id references (sessions can continue with a new key)
	if err := queries().ClearSessionApiKeyID(ctx, uuid.NullUUID{UUID: key.ID, Valid: true}); err != nil {
		return obj.ErrServerError("failed to clear session api key references")
	}

	// Clear user default_api_key_share_id references before deleting shares
	if err := queries().ClearUserDefaultApiKeyShareByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to clear user default api key references")
	}

	// Clear workshop default_api_key_share_id references before deleting shares
	if err := queries().ClearWorkshopDefaultApiKeyShareByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to clear workshop default api key references")
	}

	// Clear institution free_use_api_key_share_id references before deleting shares
	if err := queries().ClearInstitutionFreeUseApiKeyShareByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to clear institution free-use api key references")
	}

	// Clear game sponsored API key references before deleting the key
	if err := queries().ClearGameSponsoredApiKeyByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to clear game sponsored api key references")
	}

	// Clear system free-use key reference if it points to this key
	if err := queries().ClearSystemSettingsFreeUseApiKey(ctx, uuid.NullUUID{UUID: key.ID, Valid: true}); err != nil {
		return obj.ErrServerError("failed to clear system free-use api key reference")
	}

	wasDefault := key.IsDefault

	// Clean up game_shares (and their guest data) before deleting api_key_shares,
	// because game_share.api_key_share_id has a FK constraint.
	gameShareIDs, _ := queries().GetGameShareIDsByApiKeyID(ctx, key.ID)
	for _, gsID := range gameShareIDs {
		_ = DeleteGuestDataByShareID(ctx, gsID)
		_ = queries().DeleteGameShare(ctx, gsID)
	}

	// Delete all shares
	if err := queries().DeleteApiKeySharesByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to delete shares")
	}

	if err := queries().DeleteApiKey(ctx, db.DeleteApiKeyParams{
		ID:     key.ID,
		UserID: userID,
	}); err != nil {
		return err
	}

	// If the deleted key was the default, promote the next key
	if wasDefault {
		promoteNextDefaultKey(ctx, userID)
	}

	return nil
}

// promoteNextDefaultKey sets the oldest remaining key as default for a user.
// Best-effort: errors are logged but not returned.
func promoteNextDefaultKey(ctx context.Context, userID uuid.UUID) {
	// Find the user's remaining keys via their self-shares
	shares, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil || len(shares) == 0 {
		return
	}
	// Pick the first one (oldest by share creation) that the user owns
	for _, s := range shares {
		if s.OwnerID == userID {
			_ = queries().SetDefaultApiKey(ctx, db.SetDefaultApiKeyParams{
				ID:     s.ApiKeyID,
				UserID: userID,
			})
			return
		}
	}
}

// SetDefaultApiKey sets the given API key as the user's default (clears any previous default).
func SetDefaultApiKey(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID) error {
	// Verify the key belongs to this user
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return obj.ErrNotFound("api key not found")
	}
	if key.UserID != userID {
		return obj.ErrForbidden("not the owner of this key")
	}

	// Clear existing default, then set the new one
	if err := queries().ClearDefaultApiKey(ctx, userID); err != nil {
		return obj.ErrServerError("failed to clear default key")
	}
	return queries().SetDefaultApiKey(ctx, db.SetDefaultApiKeyParams{
		ID:     apiKeyID,
		UserID: userID,
	})
}

// GetDefaultApiKey returns the user's default API key, or nil if none is set.
func GetDefaultApiKeyForUser(ctx context.Context, userID uuid.UUID) (*obj.ApiKey, error) {
	key, err := queries().GetDefaultApiKey(ctx, userID)
	if err != nil {
		return nil, nil // No default key
	}
	return &obj.ApiKey{
		ID:               key.ID,
		UserID:           key.UserID,
		Name:             key.Name,
		Platform:         key.Platform,
		Key:              key.Key,
		IsDefault:        key.IsDefault,
		LastUsageSuccess: sqlNullBoolToMaybeBool(key.LastUsageSuccess),
		LastErrorCode:    sqlNullStringToMaybeString(key.LastErrorCode),
	}, nil
}

// UpdateApiKeyLastUsageSuccess updates the last_usage_success flag and error code on an API key.
// On success, the error code is cleared. On failure, the error code is stored.
func UpdateApiKeyLastUsageSuccess(ctx context.Context, apiKeyID uuid.UUID, success bool, errorCode ...string) {
	var lastErrorCode sql.NullString
	if !success && len(errorCode) > 0 && errorCode[0] != "" {
		lastErrorCode = sql.NullString{String: errorCode[0], Valid: true}
	}
	_ = queries().UpdateApiKeyLastUsageSuccess(ctx, db.UpdateApiKeyLastUsageSuccessParams{
		ID:               apiKeyID,
		LastUsageSuccess: sql.NullBool{Bool: success, Valid: true},
		LastErrorCode:    lastErrorCode,
	})
}

// UpdateApiKeyName updates an API key's name (owner only).
func UpdateApiKeyName(ctx context.Context, userID uuid.UUID, shareID uuid.UUID, name string) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return obj.ErrNotFound("api key not found")
	}

	// Check permission
	if err := canAccessApiKey(ctx, userID, OpUpdate, key.ID, key.UserID, nil, nil, nil); err != nil {
		return err
	}

	now := time.Now()
	_, err = queries().UpdateApiKey(ctx, db.UpdateApiKeyParams{
		ID:         key.ID,
		ModifiedBy: uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt: now,
		Name:       name,
	})
	return err
}
