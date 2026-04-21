package db

import (
	"context"

	"cgl/obj"

	"github.com/google/uuid"
)

// GetSystemSettings retrieves the global system settings
func GetSystemSettings(ctx context.Context) (*obj.SystemSettings, error) {
	row, err := queries().GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}

	settings := &obj.SystemSettings{
		ID:                   row.ID,
		CreatedAt:            &row.CreatedAt,
		ModifiedAt:           &row.ModifiedAt,
		DefaultAiQualityTier: row.DefaultAiQualityTier,
	}
	if row.FreeUseAiQualityTier.Valid {
		settings.FreeUseAiQualityTier = &row.FreeUseAiQualityTier.String
	}
	if row.PromptConstraintU13.Valid {
		settings.PromptConstraintU13 = &row.PromptConstraintU13.String
	}
	if row.PromptConstraintU13p.Valid {
		settings.PromptConstraintU13p = &row.PromptConstraintU13p.String
	}
	if row.PromptConstraintU18.Valid {
		settings.PromptConstraintU18 = &row.PromptConstraintU18.String
	}
	if row.FreeUseApiKeyID.Valid {
		settings.FreeUseApiKeyID = &row.FreeUseApiKeyID.UUID
		// Enrich with key details so any admin can see the current key
		apiKey, err := GetApiKeyByID(ctx, row.FreeUseApiKeyID.UUID)
		if err == nil {
			settings.FreeUseApiKeyName = apiKey.Name
			settings.FreeUseApiKeyPlatform = apiKey.Platform
			settings.FreeUseApiKeyWorking = apiKey.LastUsageSuccess
		}
	}
	return settings, nil
}

// UpdateDefaultAiQualityTier updates the server-wide default AI quality tier
func UpdateDefaultAiQualityTier(ctx context.Context, tier string) error {
	return queries().UpdateDefaultAiQualityTier(ctx, tier)
}

// UpdateFreeUseAiQualityTier sets or clears the AI quality tier for the system free-use key
func UpdateFreeUseAiQualityTier(ctx context.Context, tier *string) error {
	return queries().UpdateFreeUseAiQualityTier(ctx, stringPtrToNullString(tier))
}

// UpdatePromptConstraintU13 sets or clears the site-level constraint for users aged 13-17 without parental consent (u13).
func UpdatePromptConstraintU13(ctx context.Context, constraint *string) error {
	return queries().UpdatePromptConstraintU13(ctx, stringPtrToNullString(constraint))
}

// UpdatePromptConstraintU13p sets or clears the site-level constraint for users aged 13-17 with parental consent (u13p).
func UpdatePromptConstraintU13p(ctx context.Context, constraint *string) error {
	return queries().UpdatePromptConstraintU13p(ctx, stringPtrToNullString(constraint))
}

// UpdatePromptConstraintU18 sets or clears the site-level constraint for users aged 18+ (u18).
func UpdatePromptConstraintU18(ctx context.Context, constraint *string) error {
	return queries().UpdatePromptConstraintU18(ctx, stringPtrToNullString(constraint))
}

// UpdateSystemSettingsFreeUseApiKey sets or clears the free-use API key (admin only).
func UpdateSystemSettingsFreeUseApiKey(ctx context.Context, apiKeyID *uuid.UUID) error {
	return queries().UpdateSystemSettingsFreeUseApiKey(ctx, uuidPtrToNullUUID(apiKeyID))
}

// ClearSystemSettingsFreeUseApiKeyByOwner clears the free-use API key if it
// references a key owned by the given user (e.g. when the user loses admin role).
func ClearSystemSettingsFreeUseApiKeyByOwner(ctx context.Context, userID uuid.UUID) error {
	return queries().ClearSystemSettingsFreeUseApiKeyByOwner(ctx, userID)
}

