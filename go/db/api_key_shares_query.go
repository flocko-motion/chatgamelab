// package: db / database access and repository layer
// type:    data
// job:     aggregate and availability queries joining API keys with their shares.
// limits:  does not create or delete keys or shares (-> api_keys.go, api_key_shares.go).
package db

import (
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

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
				Meta:             obj.Meta{CreatedAt: &s.CreatedAt},
				UserID:           s.OwnerID,
				UserName:         s.OwnerName,
				Name:             s.ApiKeyName,
				Platform:         s.ApiKeyPlatform,
				KeyShortened:     functional.ShortenLeft(s.ApiKeyKey, apiKeyShortenLength),
				IsDefault:        s.ApiKeyIsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(s.ApiKeyLastUsageSuccess),
				LastErrorCode:    sqlNullStringToMaybeString(s.ApiKeyLastErrorCode),
			})
			// Track keys owned by this user
			if s.OwnerID == userID {
				ownedKeyIDs = append(ownedKeyIDs, s.ApiKeyID)
			}
		}
	}

	// Collect game_share entries (private share links) first to know which
	// api_key_share IDs are internal game-scoped shares (not user-facing).
	var allShares []obj.ApiKeyShare
	gameShareApiKeyShareIDs := make(map[uuid.UUID]bool)
	for _, keyID := range ownedKeyIDs {
		gameShares, err := GetGameSharesWithGameByApiKeyID(ctx, keyID)
		if err != nil {
			continue
		}
		for _, gs := range gameShares {
			gameShareApiKeyShareIDs[gs.ApiKeyShareID] = true
			ls := obj.ApiKeyShare{
				ID:             gs.ApiKeyShareID,
				ApiKeyID:       keyID,
				IsPrivateShare: true,
				Game:           &obj.Game{ID: gs.GameID, Name: gs.GameName},
				Remaining:      gs.Remaining,
				GameShareID:    &gs.ID,
			}
			if gs.InstitutionID != nil {
				ls.Institution = &obj.Institution{ID: *gs.InstitutionID}
			}
			if gs.WorkshopID != nil {
				ls.Workshop = &obj.Workshop{ID: *gs.WorkshopID}
			}
			allShares = append(allShares, ls)
		}
	}

	// For each owned key, get all shares (self-shares, org shares, sponsorships, etc.)
	// Skip game-scoped internal shares that back private share links.
	for _, keyID := range ownedKeyIDs {
		shares, err := queries().GetApiKeySharesByApiKeyID(ctx, keyID)
		if err != nil {
			continue
		}
		for _, s := range shares {
			// Skip game-scoped shares that are backing a game_share entry
			if s.GameID.Valid && gameShareApiKeyShareIDs[s.ID] {
				continue
			}
			ls := obj.ApiKeyShare{
				ID:       s.ID,
				ApiKeyID: s.ApiKeyID,
				Meta: obj.Meta{
					CreatedBy:  s.CreatedBy,
					CreatedAt:  &s.CreatedAt,
					ModifiedBy: s.ModifiedBy,
					ModifiedAt: &s.ModifiedAt,
				},
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

	// Game share sponsored keys (from game_share table)
	gameShares, _ := queries().GetGameSharesByGameID(ctx, gameID)
	for _, gs := range gameShares {
		if gs.ApiKeyShareID == game.PublicSponsoredApiKeyShareID.UUID {
			continue // already added above
		}
		share, err := queries().GetApiKeyShareByID(ctx, gs.ApiKeyShareID)
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
			LastErrorCode:    sqlNullStringToMaybeString(key.LastErrorCode),
		},
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
				ApiKeyID: s.ApiKeyID,
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
