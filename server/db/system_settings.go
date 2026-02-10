package db

import (
	"context"
	"fmt"

	"cgl/game/ai"
	"cgl/log"
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
	if row.FreeUseApiKeyID.Valid {
		settings.FreeUseApiKeyID = &row.FreeUseApiKeyID.UUID
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

// UpdateSystemSettingsFreeUseApiKey sets or clears the free-use API key (admin only).
func UpdateSystemSettingsFreeUseApiKey(ctx context.Context, apiKeyID *uuid.UUID) error {
	return queries().UpdateSystemSettingsFreeUseApiKey(ctx, uuidPtrToNullUUID(apiKeyID))
}

// InitSystemSettings ensures system settings exist with a default value
// If no settings exist, it creates them using the first available AI model
func InitSystemSettings(ctx context.Context) error {
	// Check if settings already exist
	_, err := GetSystemSettings(ctx)
	if err == nil {
		// Settings already exist
		log.Debug("system settings already exist")
		return nil
	}

	// Settings don't exist, get first available model
	firstModel := getFirstAvailableModel()
	if firstModel == "" {
		return fmt.Errorf("no AI models available to set as default")
	}

	log.Info("initializing system settings", "default_model", firstModel)
	return queries().InitSystemSettings(ctx, firstModel)
}

// getFirstAvailableModel returns the first available AI model from all platforms
func getFirstAvailableModel() string {
	platforms := ai.GetAiPlatformInfos()
	for _, platform := range platforms {
		if len(platform.Models) > 0 {
			return platform.Models[0].ID
		}
	}
	return ""
}
