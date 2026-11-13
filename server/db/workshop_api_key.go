// package: db / database access and repository layer
// type:    data
// job:     the API key a workshop pays with: its own, else its organisation's free-use key, else the server's.
// limits:  does not walk a session's whole priority chain (-> game/resolve_api_key.go) or set keys (-> workshop.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// workshopKeySource names the key a workshop's plays are paid with, as one of the
// obj.WorkshopKeySource* constants, or "" when there is none anywhere. limit is the
// tier a free-use key is released up to, nil for the workshop's own key or none.
func workshopKeySource(ctx context.Context, ws db.Workshop) (source string, limit *string) {
	if ws.DefaultApiKeyShareID.Valid {
		return obj.WorkshopKeySourceWorkshop, nil
	}
	settings, settingsErr := GetSystemSettings(ctx)
	defaultTier := DefaultAiQualityTier(settings)
	if inst, err := queries().GetInstitutionByID(ctx, ws.InstitutionID); err == nil && inst.FreeUseApiKeyShareID.Valid {
		release := FreeKeyRelease(nullStringToPtr(inst.FreeUseAiQualityTier), defaultTier)
		return obj.WorkshopKeySourceOrganization, &release
	}
	if settingsErr == nil && settings.FreeUseApiKeyID != nil {
		release := FreeKeyRelease(settings.FreeUseAiQualityTier, defaultTier)
		return obj.WorkshopKeySourceServer, &release
	}
	return "", nil
}

// GetWorkshopOrganizationKeyShare returns the organisation's free-use key that a
// workshop without a key of its own falls back to, with the tier it is released up to.
// Its participants play with it through the workshop, so it is loaded without the
// per-user check that keeps them from the organisation's key everywhere else.
func GetWorkshopOrganizationKeyShare(ctx context.Context, workshopID uuid.UUID) (*obj.ApiKeyShare, *string) {
	ws, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil || ws.DefaultApiKeyShareID.Valid {
		return nil, nil
	}
	inst, err := queries().GetInstitutionByID(ctx, ws.InstitutionID)
	if err != nil || !inst.FreeUseApiKeyShareID.Valid {
		return nil, nil
	}
	row, err := queries().GetApiKeyShareByID(ctx, inst.FreeUseApiKeyShareID.UUID)
	if err != nil {
		return nil, nil
	}
	var tier *string
	if inst.FreeUseAiQualityTier.Valid {
		tier = &inst.FreeUseAiQualityTier.String
	}
	return apiKeyShareRowToObj(row), tier
}

// workshopApiKeyID returns the API key a workshop's share links pay with — its
// own, else its organisation's free-use key, else the server's; uuid.Nil when
// there is none. Used by the public page and by workshop share links. working is
// false once the key has been proven not to work.
func workshopApiKeyID(ctx context.Context, ws *db.Workshop) (keyID uuid.UUID, working bool) {
	shareIDs := []uuid.NullUUID{ws.DefaultApiKeyShareID}
	if inst, err := queries().GetInstitutionByID(ctx, ws.InstitutionID); err == nil {
		shareIDs = append(shareIDs, inst.FreeUseApiKeyShareID)
	}
	for _, id := range shareIDs {
		if !id.Valid {
			continue
		}
		row, err := queries().GetApiKeyShareByID(ctx, id.UUID)
		if err != nil {
			continue
		}
		return row.ApiKeyID, !row.KeyLastUsageSuccess.Valid || row.KeyLastUsageSuccess.Bool
	}

	settings, err := GetSystemSettings(ctx)
	if err != nil || settings.FreeUseApiKeyID == nil {
		return uuid.Nil, false
	}
	key, err := queries().GetApiKeyByID(ctx, *settings.FreeUseApiKeyID)
	if err != nil {
		return uuid.Nil, false
	}
	return key.ID, !key.LastUsageSuccess.Valid || key.LastUsageSuccess.Bool
}
