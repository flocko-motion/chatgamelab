package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/game/ai"
	"cgl/log"
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
	selfShareID, err := createApiKeyShareInternal(ctx, userID, result.ID, &userID, nil, nil, nil, true)
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

	// Clear game sponsored API key references before deleting the key
	if err := queries().ClearGameSponsoredApiKeyByApiKeyID(ctx, key.ID); err != nil {
		return obj.ErrServerError("failed to clear game sponsored api key references")
	}

	wasDefault := key.IsDefault

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
	}, nil
}

// UpdateApiKeyLastUsageSuccess updates the last_usage_success flag on an API key.
func UpdateApiKeyLastUsageSuccess(ctx context.Context, apiKeyID uuid.UUID, success bool) {
	_ = queries().UpdateApiKeyLastUsageSuccess(ctx, db.UpdateApiKeyLastUsageSuccessParams{
		ID:               apiKeyID,
		LastUsageSuccess: sql.NullBool{Bool: success, Valid: true},
	})
}

// GetSelfShareForApiKey returns the user's self-share for a given API key.
// A self-share is a share where the share's user_id matches the key owner.
func GetSelfShareForApiKey(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID) (*obj.ApiKeyShare, error) {
	shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
	if err != nil {
		return nil, obj.ErrNotFound("no shares found for API key")
	}
	for _, s := range shares {
		if s.UserID.Valid && s.UserID.UUID == userID {
			return GetApiKeyShareByID(ctx, userID, s.ID)
		}
	}
	return nil, obj.ErrNotFound("self-share not found for API key")
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

// CreateApiKeyShare creates a new share for an API key via an existing share. Verifies ownership first.
func CreateApiKeyShare(ctx context.Context, userID uuid.UUID, shareID uuid.UUID, targetUserID, workshopID, institutionID *uuid.UUID, allowPublic bool) (*uuid.UUID, error) {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return nil, obj.ErrNotFound("api key not found")
	}

	if key.UserID != userID {
		return nil, obj.ErrForbidden("only the owner can share this key")
	}

	return createApiKeyShareInternal(ctx, userID, key.ID, targetUserID, workshopID, institutionID, nil, allowPublic)
}

// createApiKeyShareInternal creates a share without ownership verification (for internal use)
func createApiKeyShareInternal(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID, gameID *uuid.UUID, allowPublic bool) (*uuid.UUID, error) {
	now := time.Now()
	arg := db.CreateApiKeyShareParams{
		CreatedBy:                 uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                 now,
		ModifiedBy:                uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                now,
		ApiKeyID:                  apiKeyID,
		UserID:                    uuidPtrToNullUUID(targetUserID),
		WorkshopID:                uuidPtrToNullUUID(workshopID),
		InstitutionID:             uuidPtrToNullUUID(institutionID),
		GameID:                    uuidPtrToNullUUID(gameID),
		AllowPublicGameSponsoring: allowPublic,
	}

	result, err := queries().CreateApiKeyShare(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &result.ID, nil
}

// DeleteApiKeyShare deletes a single share. Owner can delete any share, others can only delete their own.
func DeleteApiKeyShare(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return obj.ErrNotFound("api key not found")
	}

	isOwner := key.UserID == userID
	isOwnShare := share.UserID.Valid && share.UserID.UUID == userID

	if !isOwner && !isOwnShare {
		return obj.ErrForbidden("not authorized to delete this share")
	}

	return queries().DeleteApiKeyShare(ctx, shareID)
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
			canAccess, checkErr := queries().CanUserAccessShareViaWorkshopDefault(ctx, db.CanUserAccessShareViaWorkshopDefaultParams{
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
		ApiKeyID:                  s.ApiKeyID,
		AllowPublicGameSponsoring: s.AllowPublicGameSponsoring,
		ApiKey: &obj.ApiKey{
			ID:               s.KeyID,
			UserID:           s.KeyOwnerID,
			UserName:         s.KeyOwnerName,
			Name:             s.KeyName,
			Platform:         s.KeyPlatform,
			Key:              s.KeyKey,
			KeyShortened:     functional.ShortenLeft(s.KeyKey, apiKeyShortenLength),
			IsDefault:        s.KeyIsDefault,
			LastUsageSuccess: sqlNullBoolToMaybeBool(s.KeyLastUsageSuccess),
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
				UserName:         s.OwnerName,
				Name:             s.ApiKeyName,
				Platform:         s.ApiKeyPlatform,
				Key:              s.ApiKeyKey,
				KeyShortened:     functional.ShortenLeft(s.ApiKeyKey, apiKeyShortenLength),
				IsDefault:        s.ApiKeyIsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(s.ApiKeyLastUsageSuccess),
			},
			AllowPublicGameSponsoring: s.AllowPublicGameSponsoring,
		}
		result = append(result, share)
	}

	return result, nil
}

// GetApiKeysWithShares returns the user's API keys and all their linked shares (org, sponsorship, etc.)
// This is the combined endpoint: apiKeys are deduplicated actual keys, shares are all non-self sharing relationships.
func GetApiKeysWithShares(ctx context.Context, userID uuid.UUID) ([]obj.ApiKey, []obj.ApiKeyShare, error) {
	if err := canAccessApiKey(ctx, userID, OpList, uuid.Nil, uuid.Nil, nil, nil, nil); err != nil {
		return nil, nil, err
	}

	// Get all shares where user_id = userID (these are the user's personal key shares)
	userShares, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, nil, obj.ErrServerError("failed to get api key shares")
	}

	// Deduplicate keys and collect owned key IDs
	seenKeys := make(map[uuid.UUID]bool)
	var apiKeys []obj.ApiKey
	var ownedKeyIDs []uuid.UUID

	for _, s := range userShares {
		// Skip game sponsorship self-shares (these are shares, not keys)
		if s.GameID.Valid {
			continue
		}
		if !seenKeys[s.ApiKeyID] {
			seenKeys[s.ApiKeyID] = true
			apiKeys = append(apiKeys, obj.ApiKey{
				ID:               s.ApiKeyID,
				UserID:           s.OwnerID,
				UserName:         s.OwnerName,
				Name:             s.ApiKeyName,
				Platform:         s.ApiKeyPlatform,
				KeyShortened:     functional.ShortenLeft(s.ApiKeyKey, apiKeyShortenLength),
				IsDefault:        s.ApiKeyIsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(s.ApiKeyLastUsageSuccess),
			})
			// Track keys owned by this user
			if s.OwnerID == userID {
				ownedKeyIDs = append(ownedKeyIDs, s.ApiKeyID)
			}
		}
	}

	// For each owned key, get all shares (self-shares, org shares, sponsorships, etc.)
	var allShares []obj.ApiKeyShare
	for _, keyID := range ownedKeyIDs {
		shares, err := queries().GetApiKeySharesByApiKeyID(ctx, keyID)
		if err != nil {
			continue
		}
		for _, s := range shares {
			ls := obj.ApiKeyShare{
				ID:       s.ID,
				ApiKeyID: s.ApiKeyID,
				Meta: obj.Meta{
					CreatedBy:  s.CreatedBy,
					CreatedAt:  &s.CreatedAt,
					ModifiedBy: s.ModifiedBy,
					ModifiedAt: &s.ModifiedAt,
				},
				AllowPublicGameSponsoring: s.AllowPublicGameSponsoring,
			}
			if s.UserID.Valid {
				ls.User = &obj.User{ID: s.UserID.UUID, Name: s.UserName.String}
			}
			if s.WorkshopID.Valid {
				ls.Workshop = &obj.Workshop{ID: s.WorkshopID.UUID, Name: s.WorkshopName.String}
			}
			if s.InstitutionID.Valid {
				ls.Institution = &obj.Institution{ID: s.InstitutionID.UUID, Name: s.InstitutionName.String}
			}
			if s.GameID.Valid {
				ls.Game = &obj.Game{ID: s.GameID.UUID, Name: s.GameName.String}
			}
			allShares = append(allShares, ls)
		}
	}

	return apiKeys, allShares, nil
}

// GetApiKeySharesByInstitution returns all API key shares for an institution (heads/staff only)
func GetApiKeySharesByInstitution(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID) ([]obj.ApiKeyShare, error) {
	// Check permission - user must be head or staff of this institution
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	// User must have a role in this institution and be head or staff
	if user.Role == nil || user.Role.Institution == nil || user.Role.Institution.ID != institutionID {
		return nil, obj.ErrForbidden("not a member of this institution")
	}
	if user.Role.Role != obj.RoleHead && user.Role.Role != obj.RoleStaff {
		return nil, obj.ErrForbidden("only heads and staff can view institution API keys")
	}

	shares, err := queries().GetApiKeySharesByInstitutionID(ctx, uuid.NullUUID{UUID: institutionID, Valid: true})
	if err != nil {
		return nil, obj.ErrServerError("failed to get institution API key shares")
	}

	result := make([]obj.ApiKeyShare, 0, len(shares))
	for _, s := range shares {
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
				ID:       s.ApiKeyID,
				UserID:   s.OwnerID,
				UserName: s.OwnerName,
				Name:     s.ApiKeyName,
				Platform: s.ApiKeyPlatform,
				// Key is never exposed
			},
			AllowPublicGameSponsoring: s.AllowPublicGameSponsoring,
			Institution:               &obj.Institution{ID: institutionID},
		}
		result = append(result, share)
	}

	return result, nil
}

// GetAvailableKeysForGame returns a prioritized list of API keys available to a user for a specific game
func GetAvailableKeysForGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) ([]obj.AvailableKey, error) {
	var result []obj.AvailableKey

	// Load the game to check for sponsored keys
	game, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return nil, obj.ErrNotFound("game not found")
	}

	// Load user to get institution/workshop info
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	// Workshop participants ONLY get the workshop's default API key
	// They should not see personal keys or other options
	if user.Role != nil && user.Role.Role == obj.RoleParticipant && user.Role.Workshop != nil {
		log.Debug("user is workshop participant, checking for workshop default API key",
			"user_id", userID, "workshop_id", user.Role.Workshop.ID)

		// Get the workshop to check for default API key
		workshop, err := queries().GetWorkshopByID(ctx, user.Role.Workshop.ID)
		if err != nil {
			log.Warn("failed to get workshop for participant", "workshop_id", user.Role.Workshop.ID, "error", err)
			return result, nil // Return empty - no keys available
		}

		if !workshop.DefaultApiKeyShareID.Valid {
			log.Debug("workshop has no default API key set", "workshop_id", user.Role.Workshop.ID)
			return result, nil // Return empty - workshop has no default key
		}

		// Get the API key share details
		share, err := queries().GetApiKeyShareByID(ctx, workshop.DefaultApiKeyShareID.UUID)
		if err != nil {
			log.Warn("failed to get workshop default API key share", "share_id", workshop.DefaultApiKeyShareID.UUID, "error", err)
			return result, nil // Return empty - share not found
		}

		// Get the actual API key to get name/platform
		key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
		if err != nil {
			log.Warn("failed to get API key for workshop default share", "api_key_id", share.ApiKeyID, "error", err)
			return result, nil // Return empty - key not found
		}

		log.Info("workshop participant using workshop default API key",
			"user_id", userID, "workshop_id", user.Role.Workshop.ID,
			"key_name", key.Name, "key_platform", key.Platform, "share_id", share.ID)

		result = append(result, obj.AvailableKey{
			ShareID:   share.ID,
			Name:      key.Name,
			Platform:  key.Platform,
			Source:    "workshop",
			IsDefault: true,
		})
		return result, nil
	}

	// Get user's default share ID
	defaultShareID, _ := GetUserDefaultApiKeyShare(ctx, userID)

	// 1. Check for sponsor key (highest priority)
	// Public sponsored key share
	if game.PublicSponsoredApiKeyShareID.Valid {
		share, err := queries().GetApiKeyShareByID(ctx, game.PublicSponsoredApiKeyShareID.UUID)
		if err == nil {
			result = append(result, obj.AvailableKey{
				ShareID:   share.ID,
				Name:      share.KeyName,
				Platform:  share.KeyPlatform,
				Source:    "sponsor",
				IsDefault: false,
			})
		}
	}

	// Private sponsored key share (if accessing via share link)
	if game.PrivateSponsoredApiKeyShareID.Valid && game.PrivateSponsoredApiKeyShareID != game.PublicSponsoredApiKeyShareID {
		share, err := queries().GetApiKeyShareByID(ctx, game.PrivateSponsoredApiKeyShareID.UUID)
		if err == nil {
			result = append(result, obj.AvailableKey{
				ShareID:   share.ID,
				Name:      share.KeyName,
				Platform:  share.KeyPlatform,
				Source:    "sponsor",
				IsDefault: false,
			})
		}
	}

	// 2. Check for institution keys (if user is in an institution)
	if user.Role != nil && user.Role.Institution != nil {
		instShares, err := queries().GetApiKeySharesByInstitutionID(ctx, uuid.NullUUID{UUID: user.Role.Institution.ID, Valid: true})
		if err == nil {
			for _, s := range instShares {
				result = append(result, obj.AvailableKey{
					ShareID:   s.ID,
					Name:      s.ApiKeyName,
					Platform:  s.ApiKeyPlatform,
					Source:    "institution",
					IsDefault: defaultShareID != nil && *defaultShareID == s.ID,
				})
			}
		}
	}

	// 3. Add user's personal keys
	personalShares, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err == nil {
		for _, s := range personalShares {
			// Check if the user owns this key (personal key)
			if s.OwnerID == userID {
				result = append(result, obj.AvailableKey{
					ShareID:   s.ID,
					Name:      s.ApiKeyName,
					Platform:  s.ApiKeyPlatform,
					Source:    "personal",
					IsDefault: defaultShareID != nil && *defaultShareID == s.ID,
				})
			}
		}
	}

	return result, nil
}

// UpdateApiKeyShareAllowPublicGameSponsoring updates the allow_public_game_sponsoring flag on a share.
// Only the key owner can update this.
func UpdateApiKeyShareAllowPublicGameSponsoring(ctx context.Context, userID uuid.UUID, shareID uuid.UUID, allow bool) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return obj.ErrNotFound("api key not found")
	}

	if key.UserID != userID {
		return obj.ErrForbidden("only the owner can update this share")
	}

	return queries().UpdateApiKeyShareAllowPublicGameSponsoring(ctx, db.UpdateApiKeyShareAllowPublicGameSponsoringParams{
		ID:                        shareID,
		AllowPublicGameSponsoring: allow,
	})
}

// GetApiKeyShareInfo returns a share and its linked shares (if the user is the owner)
func GetApiKeyShareInfo(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) (*obj.ApiKeyShare, []obj.ApiKeyShare, error) {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, nil, obj.ErrNotFound("share not found")
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return nil, nil, obj.ErrNotFound("api key not found")
	}

	// Check permission
	if err := canAccessApiKey(ctx, userID, OpRead, key.ID, key.UserID, nil, nil, nil); err != nil {
		return nil, nil, err
	}

	isOwner := key.UserID == userID

	result := &obj.ApiKeyShare{
		ID: share.ID,
		Meta: obj.Meta{
			CreatedBy:  share.CreatedBy,
			CreatedAt:  &share.CreatedAt,
			ModifiedBy: share.ModifiedBy,
			ModifiedAt: &share.ModifiedAt,
		},
		ApiKey: &obj.ApiKey{
			ID:               key.ID,
			UserID:           key.UserID,
			Name:             key.Name,
			Platform:         key.Platform,
			KeyShortened:     functional.ShortenLeft(key.Key, apiKeyShortenLength),
			IsDefault:        key.IsDefault,
			LastUsageSuccess: sqlNullBoolToMaybeBool(key.LastUsageSuccess),
		},
		AllowPublicGameSponsoring: share.AllowPublicGameSponsoring,
	}

	if share.UserID.Valid {
		result.User = &obj.User{ID: share.UserID.UUID}
	}
	if share.WorkshopID.Valid {
		result.Workshop = &obj.Workshop{ID: share.WorkshopID.UUID}
	}
	if share.InstitutionID.Valid {
		result.Institution = &obj.Institution{ID: share.InstitutionID.UUID}
	}
	if share.GameID.Valid {
		result.Game = &obj.Game{ID: share.GameID.UUID}
	}

	// If owner, get all linked shares for this API key
	var linkedShares []obj.ApiKeyShare
	if isOwner {
		shares, err := queries().GetApiKeySharesByApiKeyID(ctx, key.ID)
		if err != nil {
			return nil, nil, obj.ErrServerError("failed to get linked shares")
		}
		linkedShares = make([]obj.ApiKeyShare, 0, len(shares))
		for _, s := range shares {
			ls := obj.ApiKeyShare{
				ID: s.ID,
				Meta: obj.Meta{
					CreatedBy:  s.CreatedBy,
					CreatedAt:  &s.CreatedAt,
					ModifiedBy: s.ModifiedBy,
					ModifiedAt: &s.ModifiedAt,
				},
				ApiKeyID:                  s.ApiKeyID,
				AllowPublicGameSponsoring: s.AllowPublicGameSponsoring,
			}
			if s.UserID.Valid {
				ls.User = &obj.User{ID: s.UserID.UUID, Name: s.UserName.String}
			}
			if s.WorkshopID.Valid {
				ls.Workshop = &obj.Workshop{ID: s.WorkshopID.UUID, Name: s.WorkshopName.String}
			}
			if s.InstitutionID.Valid {
				ls.Institution = &obj.Institution{ID: s.InstitutionID.UUID, Name: s.InstitutionName.String}
			}
			if s.GameID.Valid {
				ls.Game = &obj.Game{ID: s.GameID.UUID, Name: s.GameName.String}
			}
			linkedShares = append(linkedShares, ls)
		}
	}

	return result, linkedShares, nil
}
