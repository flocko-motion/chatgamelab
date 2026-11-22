package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

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

	// the key's last recorded check (RecordCheck): nil never checked
	lastWorked  *bool
	lastProblem string
}

// lastChecked sets the key's last recorded check on the payer.
func (p Payer) lastChecked(worked sql.NullBool, problem sql.NullString) Payer {
	if worked.Valid {
		p.lastWorked = &worked.Bool
	}
	p.lastProblem = problem.String
	return p
}

// Secret is the payer's key, read fresh on every call so a rotated key
// applies at once. Only a payer the core picked has one: a Payer made
// elsewhere reads nothing.
func (p Payer) Secret(ctx context.Context) (string, error) {
	if !p.chosen {
		return "", forbidden("this key is not yours to use")
	}
	key, err := store.Queries().V2GetApiKeyByID(ctx, p.KeyID)
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
	_ = store.Queries().V2UpdateApiKeyLastUsageSuccess(ctx, db.V2UpdateApiKeyLastUsageSuccessParams{
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
	c, err := loadCascade(ctx)
	if err != nil {
		return nil, nil, err
	}
	return c.payers(ctx, game, platform)
}

// source is one step of the cascade as loaded: a payer, or why there is none.
type source struct {
	name  string // obj.ApiKeyType*
	payer *Payer
	skip  string
}

// cascade is the caller's side of the payers, loaded once: every source but
// the game's sponsor, which payers adds for each game. A game list reads the
// cascade once for all its games.
type cascade struct {
	defaultTier string
	before      []source // the workshop's key
	after       []source // the organisation's free use, the own key, the site's free use
}

// loadCascade loads the caller's sources of payers, as Payers orders them.
func loadCascade(ctx context.Context) (*cascade, error) {
	q := store.Queries()
	settings, err := q.V2GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	c := &cascade{defaultTier: string(engine.TierMedium)}
	if settings.DefaultAiQualityTier != "" {
		c.defaultTier = settings.DefaultAiQualityTier
	}
	tierOf := func(t sql.NullString) string {
		if t.Valid && t.String != "" {
			return t.String
		}
		return c.defaultTier
	}
	inWorkshop, freeUse, own := payerGrants(ctx)
	caller := from(ctx)

	// 1. the workshop's key
	workshop := source{name: obj.ApiKeyTypeWorkshop, skip: "not in a workshop"}
	if w := caller.CurrentWorkshop(); w != nil && inWorkshop {
		if ws, err := q.V2GetWorkshopByID(ctx, *w); err == nil {
			workshop = c.byShare(ctx, obj.ApiKeyTypeWorkshop, ws.DefaultApiKeyShareID, tierOf(ws.AiQualityTier))
		} else {
			workshop.skip = "workshop gone"
		}
	}
	c.before = []source{workshop}

	// 3. the organisation's free use
	orgFree := source{name: obj.ApiKeyTypeInstitutionFreeUse, skip: "no organisation's free use for this caller"}
	if org := caller.Organisation(); org != nil && freeUse {
		if inst, err := q.V2GetInstitutionByID(ctx, *org); err == nil {
			orgFree = c.byShare(ctx, obj.ApiKeyTypeInstitutionFreeUse, inst.FreeUseApiKeyShareID, tierOf(inst.FreeUseAiQualityTier))
		} else {
			orgFree.skip = "organisation gone"
		}
	}

	// 4. the player's own default key
	ownKey := source{name: obj.ApiKeyTypePersonal, skip: "no own keys"}
	if me := caller.Account(); me != nil && own {
		shares, err := q.V2OwnDefaultKeyShare(ctx, *me)
		if err != nil {
			return nil, err
		}
		user, err := q.V2GetUserByID(ctx, *me)
		if err != nil {
			return nil, err
		}
		if len(shares) > 0 {
			ownKey = c.byShare(ctx, obj.ApiKeyTypePersonal, uuid.NullUUID{UUID: shares[0], Valid: true}, tierOf(user.AiQualityTier))
		} else {
			ownKey.skip = "no default key"
		}
	}

	// 5. the site's free use
	site := source{name: obj.ApiKeyTypeSystemFreeUse, skip: "none configured"}
	if settings.FreeUseApiKeyID.Valid {
		if k, err := q.V2PayerKey(ctx, settings.FreeUseApiKeyID.UUID); err == nil {
			p := Payer{KeyID: k.ID, KeyName: k.Name, Platform: k.Platform, Tier: tierOf(settings.FreeUseAiQualityTier)}.
				lastChecked(k.LastUsageSuccess, k.LastErrorCode)
			site = source{name: obj.ApiKeyTypeSystemFreeUse, payer: &p}
		} else {
			site.skip = "key gone"
		}
	}
	c.after = []source{orgFree, ownKey, site}
	return c, nil
}

// byShare is a source paying by a share, at tier.
func (c *cascade) byShare(ctx context.Context, name string, share uuid.NullUUID, tier string) source {
	if !share.Valid {
		return source{name: name, skip: "none configured"}
	}
	s, err := store.Queries().V2PayerShare(ctx, share.UUID)
	if err != nil {
		return source{name: name, skip: "share gone"}
	}
	p := Payer{ShareID: &s.ID, KeyID: s.KeyID, KeyName: s.Name, Platform: s.Platform, Tier: tier}.
		lastChecked(s.LastUsageSuccess, s.LastErrorCode)
	return source{name: name, payer: &p}
}

// payers are the cascade's payers for a session of game, its sponsor second,
// at most maxPayers; with platform, only its keys.
func (c *cascade) payers(ctx context.Context, game uuid.UUID, platform string) ([]Payer, []PayerStep, error) {
	// 2. the game's sponsor
	sponsor, err := store.Queries().V2GameSponsor(ctx, game)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	sources := append(append(slices.Clone(c.before), c.byShare(ctx, obj.ApiKeyTypeSponsor, sponsor, c.defaultTier)), c.after...)

	var payers []Payer
	var steps []PayerStep
	seen := map[uuid.UUID]bool{}
	for _, src := range sources {
		if src.payer == nil {
			steps = append(steps, PayerStep{Source: src.name, Result: "skipped", Detail: src.skip})
			continue
		}
		p := *src.payer
		p.Source = src.name
		switch {
		case seen[p.KeyID]:
			steps = append(steps, PayerStep{Source: src.name, Result: "skipped", Detail: "duplicate key", Payer: &p})
		case len(payers) >= maxPayers:
			steps = append(steps, PayerStep{Source: src.name, Result: "skipped", Detail: "max candidates reached", Payer: &p})
		case !acceptsKeys(p.Platform):
			steps = append(steps, PayerStep{Source: src.name, Result: "filtered", Detail: "platform not supported yet", Payer: &p})
		case platform != "" && p.Platform != platform:
			steps = append(steps, PayerStep{Source: src.name, Result: "filtered", Detail: fmt.Sprintf("platform %s != %s", p.Platform, platform), Payer: &p})
		default:
			p.chosen = true
			seen[p.KeyID] = true
			payers = append(payers, p)
			steps = append(steps, PayerStep{Source: src.name, Result: "selected", Payer: &p})
		}
	}
	return payers, steps, nil
}
