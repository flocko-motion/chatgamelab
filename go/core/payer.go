package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/obj"

	"engine"

	"github.com/google/uuid"
)

// Who pays for a session: the first of these that applies and whose key
// works (the engine checks it, session/session.go), at most maxPayers:
//
//  1. the key of the workshop the player is in, at the workshop's tier;
//  2. the share sponsoring the game, for whoever plays it;
//  3. their organisation's free-use key, for its members, not its guests;
//  4. their own default key, at their own tier;
//  5. the site's free-use key.
//
// The workshop's and the organisation's keys are their organisation's own
// shares (SetWorkshopKey, SetOrganisationFreeUseKey), so whoever may use
// them may play with them. Without a tier of its own a payer plays at the
// site's default.

// maxPayers caps the candidates a session tries.
const maxPayers = 3

// Payer is a key that may pay for the caller's session.
type Payer struct {
	ShareID  *uuid.UUID // the share it pays by; nil: the site's free use, by the key itself
	KeyID    uuid.UUID
	KeyName  string
	Platform string
	Tier     string // high, medium or low, never empty
	Source   string // obj.ApiKeyType*

	chosen bool // the core picked it: only then its key is read
}

// Secret is the payer's key, read fresh on every call so a rotated key
// applies at once. Only a payer the core picked has one: a Payer made
// elsewhere reads nothing.
func (p Payer) Secret(ctx context.Context) (string, error) {
	if !p.chosen {
		return "", forbidden("this key is not yours to use")
	}
	key, err := store.Queries().GetApiKeyByID(ctx, p.KeyID)
	if err != nil {
		return "", fmt.Errorf("the API key is gone: %w", err)
	}
	return key.Key, nil
}

// RecordCheck records whether the payer's key worked: on a refusal with its
// problem, so its owner sees why.
func (p Payer) RecordCheck(ctx context.Context, worked bool, problem string) {
	if !p.chosen {
		return
	}
	code := sql.NullString{String: problem, Valid: !worked && problem != ""}
	_ = store.Queries().UpdateApiKeyLastUsageSuccess(ctx, db.UpdateApiKeyLastUsageSuccessParams{
		ID: p.KeyID, LastUsageSuccess: sql.NullBool{Bool: worked, Valid: true}, LastErrorCode: code,
	})
}

// PayerStep is one step of looking for payers, for telemetry.
type PayerStep struct {
	Source string
	Result string // "selected", "skipped", "filtered"
	Detail string
	Payer  *Payer
}

// Payers are the keys that may pay for the caller's session of game, in the
// order above; with platform, only its keys (resuming a session). The steps
// say why each source did or did not count.
func Payers(ctx context.Context, game uuid.UUID, platform string) ([]Payer, []PayerStep, error) {
	q := store.Queries()
	settings, err := q.GetSystemSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	defaultTier := string(engine.TierMedium)
	if settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}
	tierOf := func(t sql.NullString) string {
		if t.Valid && t.String != "" {
			return t.String
		}
		return defaultTier
	}

	var payers []Payer
	var steps []PayerStep
	seen := map[uuid.UUID]bool{}
	add := func(source string, p Payer) {
		p.Source = source
		switch {
		case seen[p.KeyID]:
			steps = append(steps, PayerStep{Source: source, Result: "skipped", Detail: "duplicate key", Payer: &p})
		case len(payers) >= maxPayers:
			steps = append(steps, PayerStep{Source: source, Result: "skipped", Detail: "max candidates reached", Payer: &p})
		case !acceptsKeys(p.Platform):
			steps = append(steps, PayerStep{Source: source, Result: "filtered", Detail: "platform not supported yet", Payer: &p})
		case platform != "" && p.Platform != platform:
			steps = append(steps, PayerStep{Source: source, Result: "filtered", Detail: fmt.Sprintf("platform %s != %s", p.Platform, platform), Payer: &p})
		default:
			p.chosen = true
			seen[p.KeyID] = true
			payers = append(payers, p)
			steps = append(steps, PayerStep{Source: source, Result: "selected", Payer: &p})
		}
	}
	skip := func(source, why string) {
		steps = append(steps, PayerStep{Source: source, Result: "skipped", Detail: why})
	}
	byShare := func(source string, share uuid.NullUUID, tier string) {
		if !share.Valid {
			skip(source, "none configured")
			return
		}
		s, err := q.V2PayerShare(ctx, share.UUID)
		if err != nil {
			skip(source, "share gone")
			return
		}
		add(source, Payer{ShareID: &s.ID, KeyID: s.KeyID, KeyName: s.Name, Platform: s.Platform, Tier: tier})
	}

	inWorkshop, freeUse, own := payerGrants(ctx)
	c := from(ctx)

	// 1. the workshop's key
	if w := c.CurrentWorkshop(); w != nil && inWorkshop {
		ws, err := q.GetWorkshopByID(ctx, *w)
		if err == nil {
			byShare(obj.ApiKeyTypeWorkshop, ws.DefaultApiKeyShareID, tierOf(ws.AiQualityTier))
		} else {
			skip(obj.ApiKeyTypeWorkshop, "workshop gone")
		}
	} else {
		skip(obj.ApiKeyTypeWorkshop, "not in a workshop")
	}

	// 2. the game's sponsor
	sponsor, err := q.V2GameSponsor(ctx, game)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	byShare(obj.ApiKeyTypeSponsor, sponsor, defaultTier)

	// 3. the organisation's free use
	if org := c.Organisation(); org != nil && freeUse {
		inst, err := q.GetInstitutionByID(ctx, *org)
		if err == nil {
			byShare(obj.ApiKeyTypeInstitutionFreeUse, inst.FreeUseApiKeyShareID, tierOf(inst.FreeUseAiQualityTier))
		} else {
			skip(obj.ApiKeyTypeInstitutionFreeUse, "organisation gone")
		}
	} else {
		skip(obj.ApiKeyTypeInstitutionFreeUse, "no organisation's free use for this caller")
	}

	// 4. the player's own default key
	if me := c.Account(); me != nil && own {
		shares, err := q.V2OwnDefaultKeyShare(ctx, *me)
		if err != nil {
			return nil, nil, err
		}
		user, err := q.GetUserByID(ctx, *me)
		if err != nil {
			return nil, nil, err
		}
		if len(shares) > 0 {
			byShare(obj.ApiKeyTypePersonal, uuid.NullUUID{UUID: shares[0], Valid: true}, tierOf(user.AiQualityTier))
		} else {
			skip(obj.ApiKeyTypePersonal, "no default key")
		}
	} else {
		skip(obj.ApiKeyTypePersonal, "no own keys")
	}

	// 5. the site's free use
	if settings.FreeUseApiKeyID.Valid {
		k, err := q.V2PayerKey(ctx, settings.FreeUseApiKeyID.UUID)
		if err == nil {
			add(obj.ApiKeyTypeSystemFreeUse, Payer{KeyID: k.ID, KeyName: k.Name, Platform: k.Platform, Tier: tierOf(settings.FreeUseAiQualityTier)})
		} else {
			skip(obj.ApiKeyTypeSystemFreeUse, "key gone")
		}
	} else {
		skip(obj.ApiKeyTypeSystemFreeUse, "none configured")
	}
	return payers, steps, nil
}
