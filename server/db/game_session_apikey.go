// package: db / database access and repository layer
// type:    data
// job:     api-key resolution and ai-session/state updates for game sessions.
// limits:  does not create or list sessions (-> game_sessions.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// UpdateGameSessionAiSession updates the AI session state for a game session
func UpdateGameSessionAiSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, aiSession string) error {
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	_, err = queries().UpdateGameSessionAiSession(ctx, db.UpdateGameSessionAiSessionParams{
		ID:        sessionID,
		AiSession: []byte(aiSession),
	})
	if err != nil {
		return obj.ErrServerError("failed to update session AI state")
	}
	return nil
}

// ResolveAndUpdateGameSessionApiKey re-resolves the API key for a session using the standard
// priority chain (workshop → sponsored game → institution free-use → user default → system free-use) and updates the session.
// Used when resuming a session whose API key was deleted.
// Only accepts keys from the same AI platform as the session to prevent mid-session platform switches.
func ResolveAndUpdateGameSessionApiKey(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*obj.GameSession, error) {
	// Load and verify session ownership
	session, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}
	if session.UserID != userID {
		return nil, obj.ErrForbidden("not the owner of this session")
	}

	// Only accept keys from the same platform as the session.
	// Switching platforms mid-session would break AiSession state (platform-specific conversation IDs).
	requiredPlatform := session.AiPlatform

	// matchesPlatform returns false if the share's platform doesn't match the session's locked platform.
	matchesPlatform := func(s *obj.ApiKeyShare) bool {
		return s != nil && s.ApiKey != nil && (requiredPlatform == "" || s.ApiKey.Platform == requiredPlatform)
	}

	// Resolve the API key and AI quality tier using the same priority chain as session creation:
	// 1. Workshop key + tier → 2. Sponsored game key → 3. Institution free-use key + tier → 4. User default key + tier → 5. System free-use key + tier
	var share *obj.ApiKeyShare
	var sourceTier *string

	user, userErr := GetUserByID(ctx, userID)

	// Load system settings for default tier fallback
	settings, _ := GetSystemSettings(ctx)
	defaultTier := obj.AiModelBalanced
	if settings != nil && settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}

	// 1. Check for workshop key
	if userErr == nil && user.Role != nil && user.Role.Workshop != nil {
		workshop, wsErr := GetWorkshopByID(ctx, userID, user.Role.Workshop.ID)
		if wsErr == nil && workshop.DefaultApiKeyShareID != nil {
			candidate, _ := GetApiKeyShareByID(ctx, userID, *workshop.DefaultApiKeyShareID)
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = workshop.AiQualityTier
			}
		}
	}

	// 2. Check sponsored game key
	if share == nil {
		game, gameErr := loadGameByID(ctx, session.GameID)
		if gameErr == nil && game.PublicSponsoredApiKeyShareID != nil {
			candidate, shareErr := GetApiKeyShareByID(ctx, userID, *game.PublicSponsoredApiKeyShareID)
			if shareErr == nil && matchesPlatform(candidate) {
				share = candidate
			}
		}
	}

	// 3. Check institution free-use key
	if share == nil && userErr == nil && user.Role != nil && user.Role.Institution != nil && user.Role.Institution.FreeUseApiKeyShareID != nil {
		candidate, _ := GetApiKeyShareByID(ctx, userID, *user.Role.Institution.FreeUseApiKeyShareID)
		if matchesPlatform(candidate) {
			share = candidate
			institution, instErr := GetInstitutionByID(ctx, userID, user.Role.Institution.ID)
			if instErr == nil {
				sourceTier = institution.FreeUseAiQualityTier
			}
		}
	}

	// 4. Check user's default API key (is_default=true on api_key table)
	if share == nil && userErr == nil {
		defaultKey, _ := GetDefaultApiKeyForUser(ctx, userID)
		if defaultKey != nil {
			candidate, _ := GetSelfShareForApiKey(ctx, userID, defaultKey.ID)
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = user.AiQualityTier
			}
		}
	}

	// 5. Check system free-use key (stored as api_key_id, not a share)
	if share == nil && settings != nil && settings.FreeUseApiKeyID != nil {
		apiKey, keyErr := GetApiKeyByID(ctx, *settings.FreeUseApiKeyID)
		if keyErr == nil {
			candidate := &obj.ApiKeyShare{
				ApiKeyID: apiKey.ID,
				ApiKey:   apiKey,
			}
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = settings.FreeUseAiQualityTier
			}
		}
	}

	if share == nil || share.ApiKey == nil {
		if requiredPlatform != "" {
			return nil, &obj.HTTPError{StatusCode: 400, Code: obj.ErrCodeNoApiKey, Message: "No API key available for platform " + requiredPlatform + ". All available keys use a different AI platform."}
		}
		return nil, &obj.HTTPError{StatusCode: 400, Code: obj.ErrCodeNoApiKey, Message: "No API key available. Please configure an API key in your settings."}
	}

	// Resolve the AI model tier: source tier → system default → hardcoded fallback
	aiModel := defaultTier
	if sourceTier != nil && *sourceTier != "" {
		aiModel = *sourceTier
	}

	// Update the session
	result, err := queries().UpdateGameSessionApiKey(ctx, db.UpdateGameSessionApiKeyParams{
		ID:         sessionID,
		ApiKeyID:   uuid.NullUUID{UUID: share.ApiKey.ID, Valid: true},
		AiPlatform: share.ApiKey.Platform,
		AiModel:    aiModel,
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to update session: " + err.Error())
	}

	return &obj.GameSession{
		ID:         result.ID,
		GameID:     result.GameID,
		UserID:     result.UserID,
		ApiKeyID:   nullUUIDToPtr(result.ApiKeyID),
		AiPlatform: result.AiPlatform,
		AiModel:    result.AiModel,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}
