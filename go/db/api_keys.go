// package: db / database access and repository layer
// type:    data
// job:     create, delete, default-selection, and metadata updates for API keys.
// limits:  does not manage shares (-> api_key_shares.go).
package db

import (
	"cgl/obj"
	"context"
	"engine"
	"time"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

const apiKeyShortenLength = 6

// AcceptsKeys says whether a key for the platform may be stored and used: a
// platform of the engine's that takes users' keys, or the mock in development
// mode, so a dev stack and the integration tests play without a provider.
func AcceptsKeys(platform string) bool {
	if schemafapi.DevMode() && engine.Platform(platform) == engine.PlatformMock {
		return true
	}
	return engine.AcceptsKeys(engine.Platform(platform))
}

func createApiKeyAndSelfShare(ctx context.Context, userID uuid.UUID, name, platform, key string) (apiKeyID uuid.UUID, shareID uuid.UUID, err error) {
	if !AcceptsKeys(platform) {
		return uuid.Nil, uuid.Nil, obj.ErrInvalidPlatformf("platform does not take API keys: %s", platform)
	}

	// Auto-set as default if this is the user's first key
	count, err := queries().CountApiKeysByUser(ctx, userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, obj.ErrServerError("failed to count user keys")
	}
	isDefault := count == 0

	now := time.Now()
	arg := CreateApiKeyParams{
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
			_ = queries().SetDefaultApiKey(ctx, SetDefaultApiKeyParams{
				ID:     s.ApiKeyID,
				UserID: userID,
			})
			return
		}
	}
}
