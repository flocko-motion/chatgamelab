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
// obj.WorkshopKeySource* constants, or "" when there is none anywhere.
func workshopKeySource(ctx context.Context, ws db.Workshop) string {
	if ws.DefaultApiKeyShareID.Valid {
		return obj.WorkshopKeySourceWorkshop
	}
	if inst, err := queries().GetInstitutionByID(ctx, ws.InstitutionID); err == nil && inst.FreeUseApiKeyShareID.Valid {
		return obj.WorkshopKeySourceOrganization
	}
	if settings, err := GetSystemSettings(ctx); err == nil && settings.FreeUseApiKeyID != nil {
		return obj.WorkshopKeySourceServer
	}
	return ""
}

// GetWorkshopOrganizationKeyShare returns the organisation's free-use key that a
// workshop without a key of its own falls back to, with the organisation's tier.
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

// publicPageApiKeyID returns the API key the public page's share links pay with,
// following the workshop's fallback; uuid.Nil when there is none. working is
// false once the key has been proven not to work.
func publicPageApiKeyID(ctx context.Context, ws *db.Workshop) (keyID uuid.UUID, working bool) {
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
