package game

import (
	"context"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// resolveApiKeyForSession resolves the API key for a new game session using priority:
//  1. Workshop key (if user is in a workshop with a configured default key)
//  2. Institution free-use key (if user's institution has one configured)
//  3. System free-use key (if an admin has configured a global free-use key)
//  4. User's default API key share
//
// The resolved key determines the AI platform used for the session.
// Returns the resolved share or an HTTPError if no key is available.
func resolveApiKeyForSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (*obj.ApiKeyShare, *obj.HTTPError) {
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to get user")
	}

	if share := resolveWorkshopKey(ctx, user); share != nil {
		return share, nil
	}

	if share := resolveInstitutionFreeUseKey(ctx, user); share != nil {
		return share, nil
	}

	if share := resolveSystemFreeUseKey(ctx); share != nil {
		return share, nil
	}

	if share := resolveUserDefaultKey(ctx, userID); share != nil {
		return share, nil
	}

	log.Debug("no API key available for session", "user_id", userID, "game_id", gameID)
	return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "No API key available. Please configure an API key in your settings.")
}

// resolveWorkshopKey checks if the user is in a workshop that has a default API key configured.
func resolveWorkshopKey(ctx context.Context, user *obj.User) *obj.ApiKeyShare {
	if user.Role == nil || user.Role.Workshop == nil {
		return nil
	}

	workshop, err := db.GetWorkshopByID(ctx, user.ID, user.Role.Workshop.ID)
	if err != nil || workshop.DefaultApiKeyShareID == nil {
		return nil
	}

	share, err := db.GetApiKeyShareByID(ctx, user.ID, *workshop.DefaultApiKeyShareID)
	if err != nil {
		log.Warn("workshop default API key share not accessible", "share_id", *workshop.DefaultApiKeyShareID, "error", err)
		return nil
	}

	log.Debug("resolved workshop key", "workshop_id", user.Role.Workshop.ID, "share_id", share.ID, "platform", share.ApiKey.Platform)
	return share
}

// resolveInstitutionFreeUseKey checks if the user's institution has a free-use API key configured.
func resolveInstitutionFreeUseKey(ctx context.Context, user *obj.User) *obj.ApiKeyShare {
	if user.Role == nil || user.Role.Institution == nil || user.Role.Institution.FreeUseApiKeyShareID == nil {
		return nil
	}

	share, err := db.GetApiKeyShareByID(ctx, user.ID, *user.Role.Institution.FreeUseApiKeyShareID)
	if err != nil {
		log.Warn("institution free-use API key share not accessible", "share_id", *user.Role.Institution.FreeUseApiKeyShareID, "error", err)
		return nil
	}

	log.Debug("resolved institution free-use key", "institution_id", user.Role.Institution.ID, "share_id", share.ID, "platform", share.ApiKey.Platform)
	return share
}

// resolveSystemFreeUseKey checks if the admin has configured a global free-use API key in system settings.
// The key is stored directly (not as a share), so we load it and wrap it in a synthetic ApiKeyShare.
func resolveSystemFreeUseKey(ctx context.Context) *obj.ApiKeyShare {
	settings, err := db.GetSystemSettings(ctx)
	if err != nil || settings.FreeUseApiKeyID == nil {
		return nil
	}

	apiKey, err := db.GetApiKeyByID(ctx, *settings.FreeUseApiKeyID)
	if err != nil {
		log.Warn("system free-use API key not found", "api_key_id", *settings.FreeUseApiKeyID, "error", err)
		return nil
	}

	log.Debug("resolved system free-use key", "api_key_id", apiKey.ID, "platform", apiKey.Platform)
	return &obj.ApiKeyShare{
		ApiKeyID: apiKey.ID,
		ApiKey:   apiKey,
	}
}

// resolveUserDefaultKey checks if the user has a personal default API key configured.
func resolveUserDefaultKey(ctx context.Context, userID uuid.UUID) *obj.ApiKeyShare {
	defaultKey, _ := db.GetDefaultApiKeyForUser(ctx, userID)
	if defaultKey == nil {
		return nil
	}

	share, err := db.GetSelfShareForApiKey(ctx, userID, defaultKey.ID)
	if err != nil {
		log.Warn("user default API key self-share not found", "key_id", defaultKey.ID, "error", err)
		return nil
	}

	log.Debug("resolved user default key", "key_id", defaultKey.ID, "share_id", share.ID, "platform", defaultKey.Platform)
	return share
}
