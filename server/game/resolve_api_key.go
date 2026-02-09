package game

import (
	"context"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// resolvedKey holds the resolved API key share and the AI quality tier from the same source.
type resolvedKey struct {
	Share         *obj.ApiKeyShare
	AiQualityTier string // resolved tier (high/medium/low), never empty
}

// resolveApiKeyForSession resolves the API key and AI quality tier for a new game session.
// Priority chain:
//  1. Workshop key + workshop.aiQualityTier
//  2. Institution free-use key + institution.freeUseAiQualityTier
//  3. System free-use key + system_settings.freeUseAiQualityTier
//  4. User's default API key + user.aiQualityTier
//
// If the source's tier is empty, falls back to system_settings.defaultAiQualityTier.
func resolveApiKeyForSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (*resolvedKey, *obj.HTTPError) {
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get user")
	}

	// Load system settings for the default tier fallback
	settings, _ := db.GetSystemSettings(ctx)
	defaultTier := obj.AiModelBalanced // hardcoded ultimate fallback
	if settings != nil && settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}

	if share, tier := resolveWorkshopKey(ctx, user); share != nil {
		return &resolvedKey{Share: share, AiQualityTier: tierOrDefault(tier, defaultTier)}, nil
	}

	if share, tier := resolveInstitutionFreeUseKey(ctx, user); share != nil {
		return &resolvedKey{Share: share, AiQualityTier: tierOrDefault(tier, defaultTier)}, nil
	}

	if share, tier := resolveSystemFreeUseKey(ctx, settings); share != nil {
		return &resolvedKey{Share: share, AiQualityTier: tierOrDefault(tier, defaultTier)}, nil
	}

	if share, tier := resolveUserDefaultKey(ctx, user); share != nil {
		return &resolvedKey{Share: share, AiQualityTier: tierOrDefault(tier, defaultTier)}, nil
	}

	log.Debug("no API key available for session", "user_id", userID, "game_id", gameID)
	return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "No API key available. Please configure an API key in your settings.")
}

// tierOrDefault returns tier if non-empty, otherwise the default.
func tierOrDefault(tier *string, defaultTier string) string {
	if tier != nil && *tier != "" {
		return *tier
	}
	return defaultTier
}

// resolveWorkshopKey checks if the user is in a workshop that has a default API key configured.
func resolveWorkshopKey(ctx context.Context, user *obj.User) (*obj.ApiKeyShare, *string) {
	if user.Role == nil || user.Role.Workshop == nil {
		return nil, nil
	}

	workshop, err := db.GetWorkshopByID(ctx, user.ID, user.Role.Workshop.ID)
	if err != nil || workshop.DefaultApiKeyShareID == nil {
		return nil, nil
	}

	share, err := db.GetApiKeyShareByID(ctx, user.ID, *workshop.DefaultApiKeyShareID)
	if err != nil {
		log.Warn("workshop default API key share not accessible", "share_id", *workshop.DefaultApiKeyShareID, "error", err)
		return nil, nil
	}

	log.Debug("resolved workshop key", "workshop_id", user.Role.Workshop.ID, "share_id", share.ID, "platform", share.ApiKey.Platform, "tier", workshop.AiQualityTier)
	return share, workshop.AiQualityTier
}

// resolveInstitutionFreeUseKey checks if the user's institution has a free-use API key configured.
func resolveInstitutionFreeUseKey(ctx context.Context, user *obj.User) (*obj.ApiKeyShare, *string) {
	if user.Role == nil || user.Role.Institution == nil || user.Role.Institution.FreeUseApiKeyShareID == nil {
		return nil, nil
	}

	share, err := db.GetApiKeyShareByID(ctx, user.ID, *user.Role.Institution.FreeUseApiKeyShareID)
	if err != nil {
		log.Warn("institution free-use API key share not accessible", "share_id", *user.Role.Institution.FreeUseApiKeyShareID, "error", err)
		return nil, nil
	}

	// Load full institution to get the tier
	institution, err := db.GetInstitutionByID(ctx, user.ID, user.Role.Institution.ID)
	if err != nil {
		log.Warn("failed to load institution for tier", "institution_id", user.Role.Institution.ID, "error", err)
		return share, nil
	}

	log.Debug("resolved institution free-use key", "institution_id", institution.ID, "share_id", share.ID, "platform", share.ApiKey.Platform, "tier", institution.FreeUseAiQualityTier)
	return share, institution.FreeUseAiQualityTier
}

// resolveSystemFreeUseKey checks if the admin has configured a global free-use API key in system settings.
// The key is stored directly (not as a share), so we load it and wrap it in a synthetic ApiKeyShare.
func resolveSystemFreeUseKey(ctx context.Context, settings *obj.SystemSettings) (*obj.ApiKeyShare, *string) {
	if settings == nil || settings.FreeUseApiKeyID == nil {
		return nil, nil
	}

	apiKey, err := db.GetApiKeyByID(ctx, *settings.FreeUseApiKeyID)
	if err != nil {
		log.Warn("system free-use API key not found", "api_key_id", *settings.FreeUseApiKeyID, "error", err)
		return nil, nil
	}

	log.Debug("resolved system free-use key", "api_key_id", apiKey.ID, "platform", apiKey.Platform, "tier", settings.FreeUseAiQualityTier)
	return &obj.ApiKeyShare{
		ApiKeyID: apiKey.ID,
		ApiKey:   apiKey,
	}, settings.FreeUseAiQualityTier
}

// resolveUserDefaultKey checks if the user has a personal default API key configured.
func resolveUserDefaultKey(ctx context.Context, user *obj.User) (*obj.ApiKeyShare, *string) {
	defaultKey, _ := db.GetDefaultApiKeyForUser(ctx, user.ID)
	if defaultKey == nil {
		return nil, nil
	}

	share, err := db.GetSelfShareForApiKey(ctx, user.ID, defaultKey.ID)
	if err != nil {
		log.Warn("user default API key self-share not found", "key_id", defaultKey.ID, "error", err)
		return nil, nil
	}

	log.Debug("resolved user default key", "key_id", defaultKey.ID, "share_id", share.ID, "platform", defaultKey.Platform, "tier", user.AiQualityTier)
	return share, user.AiQualityTier
}
