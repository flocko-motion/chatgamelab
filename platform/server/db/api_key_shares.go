// package: db / database access and repository layer
// type:    data
// job:     create, delete, and look up API key shares (per-user and per-institution).
// limits:  does not manage API keys themselves (-> api_keys.go) or run join/availability queries (-> api_key_shares_query.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"time"

	"github.com/google/uuid"
)

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

// CreateApiKeyShare creates a new share for an API key via an existing share. Verifies ownership first.
func CreateApiKeyShare(ctx context.Context, userID uuid.UUID, shareID uuid.UUID, targetUserID, workshopID, institutionID *uuid.UUID) (*uuid.UUID, error) {
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

	return createApiKeyShareInternal(ctx, userID, key.ID, targetUserID, workshopID, institutionID, nil)
}

// createApiKeyShareInternal creates a share without ownership verification (for internal use)
func createApiKeyShareInternal(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID, gameID *uuid.UUID) (*uuid.UUID, error) {
	now := time.Now()
	arg := db.CreateApiKeyShareParams{
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

// DeleteApiKeyShare deletes a single share.
// Allowed by: key owner, share target user, or head/staff of the institution the share targets.
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

	// Head/staff of the target institution can remove org-scoped shares from colleagues
	isOrgMember := false
	if share.InstitutionID.Valid {
		user, lookupErr := GetUserByID(ctx, userID)
		if lookupErr == nil && user.Role != nil && user.Role.Institution != nil &&
			user.Role.Institution.ID == share.InstitutionID.UUID &&
			(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) {
			isOrgMember = true
		}
	}

	if !isOwner && !isOwnShare && !isOrgMember {
		return obj.ErrForbidden("not authorized to delete this share")
	}

	// Clear workshop default_api_key_share_id if it references this share
	_ = queries().ClearWorkshopDefaultApiKeyShareByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true})

	// Clear game sponsor references if they reference this share
	_ = queries().ClearGameSponsoredApiKeyByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true})

	// Clean up game_shares that reference this api_key_share (delete guest data first)
	gsIDs, _ := queries().GetGameShareIDsByApiKeyShareID(ctx, shareID)
	for _, gsID := range gsIDs {
		_ = DeleteGuestDataByShareID(ctx, gsID)
	}
	_ = queries().DeleteGameSharesByApiKeyShareID(ctx, shareID)

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
		ApiKeyID: s.ApiKeyID,
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
				UserName:         s.OwnerName,
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

			Institution: &obj.Institution{ID: institutionID},
		}
		result = append(result, share)
	}

	return result, nil
}
