// package: db / database access and repository layer
// type:    logic
// job:     the AI quality tier a free-use key plays at: the tier asked for, at most the tier the key is released up to.
// limits:  does not pick the key (-> game/resolve_api_key.go) or store a session's tier (-> game_sessions.go).
package db

import (
	sqlc "cgl/db/sqlc"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// DefaultAiQualityTier is the server's default tier, the release of a free-use key without one of its own.
func DefaultAiQualityTier(settings *obj.SystemSettings) string {
	if settings != nil && settings.DefaultAiQualityTier != "" {
		return settings.DefaultAiQualityTier
	}
	return obj.AiModelBalanced
}

// FreeKeyRelease returns the tier a free-use key is released up to: its own, else the server default.
func FreeKeyRelease(release *string, defaultTier string) string {
	if release != nil && *release != "" {
		return *release
	}
	return defaultTier
}

// FreeKeyTier returns the tier a free-use key plays at: the requested tier, at most
// the key's release. Without a request it plays at the release. cappedFrom is the
// requested tier when it was lowered, "" otherwise.
func FreeKeyTier(requested, release *string, defaultTier string) (applied string, cappedFrom string) {
	limit := FreeKeyRelease(release, defaultTier)
	if requested == nil || *requested == "" {
		return limit, ""
	}
	applied, capped := obj.CapAiQualityTier(*requested, limit)
	if capped {
		return applied, *requested
	}
	return applied, ""
}

// RequestedAiQualityTier is the tier a user asks for when a free-use key pays:
// their workshop's while they play in a workshop, else their own. nil when unset.
func RequestedAiQualityTier(ctx context.Context, user *obj.User) *string {
	if user == nil {
		return nil
	}
	if user.Role != nil && user.Role.Workshop != nil {
		ws, err := queries().GetWorkshopByID(ctx, user.Role.Workshop.ID)
		if err != nil || !ws.AiQualityTier.Valid {
			return nil
		}
		return &ws.AiQualityTier.String
	}
	return user.AiQualityTier
}

// UserAiQualityTierLimit names the key a user's own tier plays with, outside a
// workshop, and the highest tier they may choose with it (nil with their own key).
func UserAiQualityTierLimit(ctx context.Context, user *obj.User) (source string, limit *string) {
	if key, _ := GetDefaultApiKeyForUser(ctx, user.ID); key != nil {
		return obj.UserTierSourceOwn, nil
	}
	settings, settingsErr := GetSystemSettings(ctx)
	defaultTier := DefaultAiQualityTier(settings)
	if user.Role != nil && user.Role.Role != obj.RoleParticipant && user.Role.Institution != nil && user.Role.Institution.FreeUseApiKeyShareID != nil {
		if inst, err := queries().GetInstitutionByID(ctx, user.Role.Institution.ID); err == nil {
			release := FreeKeyRelease(nullStringToPtr(inst.FreeUseAiQualityTier), defaultTier)
			return obj.UserTierSourceOrganization, &release
		}
	}
	if settingsErr == nil && settings.FreeUseApiKeyID != nil {
		release := FreeKeyRelease(settings.FreeUseAiQualityTier, defaultTier)
		return obj.UserTierSourceServer, &release
	}
	return obj.UserTierSourceNone, nil
}

// SetInstitutionFreeUseAiQualityTier sets or clears (nil) the tier an institution's
// free-use key is released up to. Permission check: head of the institution or admin.
func SetInstitutionFreeUseAiQualityTier(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID, tier *string) error {
	if err := canAccessInstitution(ctx, userID, OpUpdate, &institutionID); err != nil {
		return err
	}
	if tier != nil && obj.AiModelTierPriority(*tier) < 0 {
		return obj.ErrValidation("unknown AI quality tier")
	}
	return queries().UpdateInstitutionFreeUseAiQualityTier(ctx, sqlc.UpdateInstitutionFreeUseAiQualityTierParams{
		ID:                   institutionID,
		FreeUseAiQualityTier: stringPtrToNullString(tier),
	})
}
