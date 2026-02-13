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

const maxCandidates = 3

// resolveApiKeyCandidates collects all matching API keys from the priority chain,
// deduplicated by API key ID, up to maxCandidates. The first candidate is the
// highest-priority match; subsequent entries are fallbacks.
//
// Priority chain:
//  1. Workshop key + workshop.aiQualityTier
//  2. Sponsored game key (public sponsor on the game)
//  3. Institution free-use key + institution.freeUseAiQualityTier
//  4. User's default API key + user.aiQualityTier
//  5. System free-use key + system_settings.freeUseAiQualityTier
//
// If the source's tier is empty, falls back to system_settings.defaultAiQualityTier.
func resolveApiKeyCandidates(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) ([]resolvedKey, *obj.HTTPError) {
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

	var candidates []resolvedKey
	seen := make(map[uuid.UUID]bool) // deduplicate by API key ID

	add := func(share *obj.ApiKeyShare, tier string) {
		if share == nil || share.ApiKey == nil || seen[share.ApiKey.ID] || len(candidates) >= maxCandidates {
			return
		}
		seen[share.ApiKey.ID] = true
		candidates = append(candidates, resolvedKey{Share: share, AiQualityTier: tier})
	}

	if share, tier := resolveWorkshopKey(ctx, user); share != nil {
		add(share, tierOrDefault(tier, defaultTier))
	}
	if share := resolveSponsoredGameKey(ctx, userID, gameID); share != nil {
		add(share, defaultTier)
	}
	if share, tier := resolveInstitutionFreeUseKey(ctx, user); share != nil {
		add(share, tierOrDefault(tier, defaultTier))
	}
	if share, tier := resolveUserDefaultKey(ctx, user); share != nil {
		add(share, tierOrDefault(tier, defaultTier))
	}
	if share, tier := resolveSystemFreeUseKey(ctx, settings); share != nil {
		add(share, tierOrDefault(tier, defaultTier))
	}

	if len(candidates) == 0 {
		log.Debug("no API key available for session", "user_id", userID, "game_id", gameID)
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "No API key available. Please configure an API key in your settings.")
	}

	return candidates, nil
}

// resolveApiKeyForSession resolves the highest-priority API key for a session.
// Convenience wrapper around resolveApiKeyCandidates that returns only the first match.
func resolveApiKeyForSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (*resolvedKey, *obj.HTTPError) {
	candidates, httpErr := resolveApiKeyCandidates(ctx, userID, gameID)
	if httpErr != nil {
		return nil, httpErr
	}
	return &candidates[0], nil
}

// IsApiKeyAvailable checks whether an API key can be resolved for the given user+game
// without exposing any key details. Used for upfront checks before starting/resuming a game.
func IsApiKeyAvailable(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) bool {
	resolved, _ := resolveApiKeyForSession(ctx, userID, gameID)
	return resolved != nil
}

// ResolveSessionApiKey re-resolves the API key for an existing session using the standard priority chain.
// It updates session.ApiKey, session.AiPlatform, and session.AiModel in-place.
// This must be called before every DoSessionAction to ensure the key is still valid
// (e.g. sponsorship may have been removed since the session was created).
func ResolveSessionApiKey(ctx context.Context, session *obj.GameSession) *obj.HTTPError {
	resolved, httpErr := resolveApiKeyForSession(ctx, session.UserID, session.GameID)
	if httpErr != nil {
		return httpErr
	}
	applyResolvedKey(session, resolved)
	return nil
}

// ResolveSessionApiKeyCandidates re-resolves all API key candidates for an existing session.
// Returns the ordered list of candidates for retry logic.
func ResolveSessionApiKeyCandidates(ctx context.Context, session *obj.GameSession) ([]resolvedKey, *obj.HTTPError) {
	return resolveApiKeyCandidates(ctx, session.UserID, session.GameID)
}

// applyResolvedKey updates session fields from a resolved key candidate.
func applyResolvedKey(session *obj.GameSession, resolved *resolvedKey) {
	session.ApiKey = resolved.Share.ApiKey
	session.ApiKeyID = &resolved.Share.ApiKey.ID
	session.AiPlatform = resolved.Share.ApiKey.Platform
	session.AiModel = resolved.AiQualityTier
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

	return share, workshop.AiQualityTier
}

// resolveSponsoredGameKey checks if the game has a public sponsored API key share.
func resolveSponsoredGameKey(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) *obj.ApiKeyShare {
	game, err := db.GetGameByID(ctx, nil, gameID)
	if err != nil || game.PublicSponsoredApiKeyShareID == nil {
		return nil
	}

	share, err := db.GetApiKeyShareByID(ctx, userID, *game.PublicSponsoredApiKeyShareID)
	if err != nil {
		log.Warn("sponsored game key share not accessible", "game_id", gameID, "share_id", *game.PublicSponsoredApiKeyShareID, "error", err)
		return nil
	}

	return share
}

// resolveInstitutionFreeUseKey checks if the user's institution has a free-use API key configured.
// Participants are excluded — they can only use the workshop key (priority 1).
func resolveInstitutionFreeUseKey(ctx context.Context, user *obj.User) (*obj.ApiKeyShare, *string) {
	if user.Role == nil || user.Role.Institution == nil || user.Role.Institution.FreeUseApiKeyShareID == nil {
		return nil, nil
	}
	if user.Role.Role == obj.RoleParticipant {
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

	return share, user.AiQualityTier
}
