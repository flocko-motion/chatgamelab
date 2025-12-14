package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/game/ai"
	"cgl/obj"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const apiKeyShortenLength = 6

// CreateApiKey creates a new API key for a user with a self-share
func CreateApiKey(ctx context.Context, userID uuid.UUID, name, platform, key string) (*uuid.UUID, error) {
	if !ai.IsValidApiKeyPlatform(platform) {
		return nil, errors.New("unknown platform: " + platform)
	}

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
	}
	result, err := queries().CreateApiKey(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Create a self-share so the user can access their own key via the shares API
	if _, err := createApiKeyShareInternal(ctx, userID, result.ID, &userID, nil, nil, true); err != nil {
		return nil, fmt.Errorf("failed to create self-share: %w", err)
	}

	return &result.ID, nil
}

// DeleteApiKey deletes the underlying API key and all its shares (owner only).
func DeleteApiKey(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return fmt.Errorf("api key not found: %w", err)
	}

	if key.UserID != userID {
		return errors.New("only the owner can delete this key")
	}

	// Delete all shares first
	if err := queries().DeleteApiKeySharesByApiKeyID(ctx, key.ID); err != nil {
		return fmt.Errorf("failed to delete shares: %w", err)
	}

	return queries().DeleteApiKey(ctx, db.DeleteApiKeyParams{
		ID:     key.ID,
		UserID: userID,
	})
}

// UpdateApiKeyName updates an API key's name (owner only).
func UpdateApiKeyName(ctx context.Context, userID uuid.UUID, shareID uuid.UUID, name string) error {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return fmt.Errorf("api key not found: %w", err)
	}

	if key.UserID != userID {
		return errors.New("only the owner can update this key")
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
		return nil, fmt.Errorf("share not found: %w", err)
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}

	if key.UserID != userID {
		return nil, errors.New("only the owner can share this key")
	}

	return createApiKeyShareInternal(ctx, userID, key.ID, targetUserID, workshopID, institutionID, allowPublic)
}

// createApiKeyShareInternal creates a share without ownership verification (for internal use)
func createApiKeyShareInternal(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID *uuid.UUID, allowPublic bool) (*uuid.UUID, error) {
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
		AllowPublicSponsoredPlays: allowPublic,
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
		return fmt.Errorf("share not found: %w", err)
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return fmt.Errorf("api key not found: %w", err)
	}

	isOwner := key.UserID == userID
	isOwnShare := share.UserID.Valid && share.UserID.UUID == userID

	if !isOwner && !isOwnShare {
		return errors.New("not authorized to delete this share")
	}

	return queries().DeleteApiKeyShare(ctx, shareID)
}

// GetApiKeyShareByID returns an API key share by its ID, including the full API key.
func GetApiKeyShareByID(ctx context.Context, shareID uuid.UUID) (*obj.ApiKeyShare, error) {
	s, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, fmt.Errorf("share not found: %w", err)
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
		AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
		ApiKey: &obj.ApiKey{
			ID:           s.KeyID,
			UserID:       s.KeyOwnerID,
			UserName:     s.KeyOwnerName,
			Name:         s.KeyName,
			Platform:     s.KeyPlatform,
			Key:          s.KeyKey,
			KeyShortened: functional.ShortenLeft(s.KeyKey, apiKeyShortenLength),
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
	return share, nil
}

// GetApiKeySharesByUser returns all API key shares accessible to a user
func GetApiKeySharesByUser(ctx context.Context, userID uuid.UUID) ([]obj.ApiKeyShare, error) {
	sharedKeys, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get api key shares: %w", err)
	}

	// Get the user's default API key share ID
	defaultShareID, err := GetUserDefaultApiKeyShare(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default api key share: %w", err)
	}

	result := make([]obj.ApiKeyShare, 0, len(sharedKeys))
	for _, s := range sharedKeys {
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
				ID:           s.ApiKeyID,
				UserID:       s.OwnerID,
				UserName:     s.OwnerName,
				Name:         s.ApiKeyName,
				Platform:     s.ApiKeyPlatform,
				Key:          s.ApiKeyKey,
				KeyShortened: functional.ShortenLeft(s.ApiKeyKey, apiKeyShortenLength),
			},
			AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
			IsUserDefault:             defaultShareID != nil && *defaultShareID == s.ID,
		}
		result = append(result, share)
	}

	return result, nil
}

// GetApiKeyShareInfo returns a share and its linked shares (if the user is the owner)
func GetApiKeyShareInfo(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) (*obj.ApiKeyShare, []obj.ApiKeyShare, error) {
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return nil, nil, fmt.Errorf("share not found: %w", err)
	}

	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("api key not found: %w", err)
	}

	isOwner := key.UserID == userID
	isShareTarget := share.UserID.Valid && share.UserID.UUID == userID

	if !isOwner && !isShareTarget {
		return nil, nil, errors.New("not authorized to view this share")
	}

	result := &obj.ApiKeyShare{
		ID: share.ID,
		Meta: obj.Meta{
			CreatedBy:  share.CreatedBy,
			CreatedAt:  &share.CreatedAt,
			ModifiedBy: share.ModifiedBy,
			ModifiedAt: &share.ModifiedAt,
		},
		ApiKey: &obj.ApiKey{
			ID:           key.ID,
			UserID:       key.UserID,
			Name:         key.Name,
			Platform:     key.Platform,
			KeyShortened: functional.ShortenLeft(key.Key, apiKeyShortenLength),
		},
		AllowPublicSponsoredPlays: share.AllowPublicSponsoredPlays,
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

	// If owner, get all linked shares for this API key
	var linkedShares []obj.ApiKeyShare
	if isOwner {
		shares, err := queries().GetApiKeySharesByApiKeyID(ctx, key.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get linked shares: %w", err)
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
				AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
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
			linkedShares = append(linkedShares, ls)
		}
	}

	return result, linkedShares, nil
}
