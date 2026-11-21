// package: db / database access and repository layer
// type:    data
// job:     read and update the global system settings row (defaults, youth-protection prompt constraints).
// limits:  does not define SQL queries (-> db/queries) or expose HTTP endpoints (-> api).
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
		apiKey, err := queries().GetApiKeyByID(ctx, row.FreeUseApiKeyID.UUID)
		if err == nil {
			settings.FreeUseApiKeyName = apiKey.Name
			settings.FreeUseApiKeyPlatform = apiKey.Platform
			if apiKey.LastUsageSuccess.Valid {
				settings.FreeUseApiKeyWorking = &apiKey.LastUsageSuccess.Bool
			}
		}
	}
	return settings, nil
}

// ClearSystemSettingsFreeUseApiKeyByOwner clears the free-use API key if it
// references a key owned by the given user (e.g. when the user loses admin role).
func ClearSystemSettingsFreeUseApiKeyByOwner(ctx context.Context, userID uuid.UUID) error {
	return queries().ClearSystemSettingsFreeUseApiKeyByOwner(ctx, userID)
}
