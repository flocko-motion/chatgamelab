package core

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// The site is run by the admin: its youth-protection rules per age group,
// its AI quality tiers and its free-use key, which pays last for whoever
// has no other key. The free-use key is one of the admin's own: no admin
// spends someone else's key on the whole site. The site's rules are
// readable to everyone signed in, as the fallback below an organisation's.

// SiteRules are the site's youth-protection rules per age group; nil: none
// set.
type SiteRules struct {
	U13  *string // 13-17 without parental consent
	U13p *string // 13-17 with parental consent
	U18  *string // 18 and over
}

// ReadSiteRules are the site's rules: {Read, SiteRules, Everywhere}.
func ReadSiteRules(ctx context.Context) (SiteRules, error) {
	if !granted(ctx, rules.Read, rules.SiteRules, rules.Everywhere) {
		return SiteRules{}, forbidden("you may not read the site's rules")
	}
	s, err := store.Queries().V2GetSystemSettings(ctx)
	if err != nil {
		return SiteRules{}, err
	}
	return SiteRules{U13: nullString(s.PromptConstraintU13), U13p: nullString(s.PromptConstraintU13p), U18: nullString(s.PromptConstraintU18)}, nil
}

// SiteSettings are the site's settings as the admin sees them.
type SiteSettings struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	ModifiedAt  time.Time
	DefaultTier string  // the tier wherever nothing else sets one
	FreeUseTier *string // the free-use key's tier; nil: the default
	Rules       SiteRules
	FreeUseKey  *FreeUseKey
}

// FreeUseKey is the key that pays last, and whether it worked when last used.
type FreeUseKey struct {
	ID       uuid.UUID
	Name     string
	Platform string
	Working  *bool // nil: not used yet
}

// ReadSiteSettings are the site's settings: {Read, SiteSettings,
// Everywhere}, the admin.
func ReadSiteSettings(ctx context.Context) (*SiteSettings, error) {
	if !granted(ctx, rules.Read, rules.SiteSettings, rules.Everywhere) {
		return nil, forbidden("only the admin reads the site's settings")
	}
	return siteSettings(ctx)
}

func siteSettings(ctx context.Context) (*SiteSettings, error) {
	q := store.Queries()
	s, err := q.V2GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	settings := &SiteSettings{
		ID: s.ID, CreatedAt: s.CreatedAt, ModifiedAt: s.ModifiedAt,
		DefaultTier: s.DefaultAiQualityTier, FreeUseTier: nullString(s.FreeUseAiQualityTier),
		Rules: SiteRules{U13: nullString(s.PromptConstraintU13), U13p: nullString(s.PromptConstraintU13p), U18: nullString(s.PromptConstraintU18)},
	}
	if s.FreeUseApiKeyID.Valid {
		key := &FreeUseKey{ID: s.FreeUseApiKeyID.UUID}
		if k, err := q.V2GetApiKeyByID(ctx, key.ID); err == nil {
			key.Name, key.Platform = k.Name, k.Platform
			if k.LastUsageSuccess.Valid {
				key.Working = &k.LastUsageSuccess.Bool
			}
		}
		settings.FreeUseKey = key
	}
	return settings, nil
}

// SiteSettingsChange is what changes in the site's settings: a nil field
// stays, an empty one clears it. The default tier cannot be cleared.
type SiteSettingsChange struct {
	DefaultTier *string
	FreeUseTier *string
	U13         *string
	U13p        *string
	U18         *string
}

// EditSiteSettings changes the site's settings, all or none: {Edit,
// SiteSettings, Everywhere}, the admin. It answers them as stored.
func EditSiteSettings(ctx context.Context, c SiteSettingsChange) (*SiteSettings, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return nil, forbidden("only the admin changes the site's settings")
	}
	if c.DefaultTier != nil && *c.DefaultTier == "" {
		return nil, refuse("the default AI quality tier cannot be empty")
	}
	err := store.InTx(ctx, func(q *db.Queries) error {
		if c.DefaultTier != nil {
			if err := q.V2UpdateDefaultAiQualityTier(ctx, *c.DefaultTier); err != nil {
				return err
			}
		}
		for _, set := range []struct {
			value *string
			write func(context.Context, sql.NullString) error
		}{
			{c.FreeUseTier, q.V2UpdateFreeUseAiQualityTier},
			{c.U13, q.V2UpdatePromptConstraintU13},
			{c.U13p, q.V2UpdatePromptConstraintU13p},
			{c.U18, q.V2UpdatePromptConstraintU18},
		} {
			if set.value == nil {
				continue
			}
			if err := set.write(ctx, sql.NullString{String: *set.value, Valid: *set.value != ""}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Info("site settings changed", "by", account(ctx).UUID)
	return siteSettings(ctx)
}

// SetFreeUseKey sets the key that pays last, or clears it (nil): {Edit,
// SiteSettings, Everywhere}, the admin, with a key of their own. Someone
// else's key does not exist for them (ErrNotFound).
func SetFreeUseKey(ctx context.Context, key *uuid.UUID) (*SiteSettings, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return nil, forbidden("only the admin sets the site's free-use key")
	}
	if key != nil {
		k, err := store.Queries().V2GetApiKeyByID(ctx, *key)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && k.UserID != account(ctx).UUID) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	if err := store.Queries().V2UpdateSystemSettingsFreeUseApiKey(ctx, nullUUID(key)); err != nil {
		return nil, err
	}
	log.Info("site free-use key set", "key_id", key, "by", account(ctx).UUID)
	return siteSettings(ctx)
}

// MayRestart refuses restarting the server, unless the caller runs the
// site: {Manage, Site, Everywhere}, the admin.
func MayRestart(ctx context.Context) error {
	if !granted(ctx, rules.Manage, rules.Site, rules.Everywhere) {
		return forbidden("only the admin restarts the server")
	}
	log.Info("restart requested", "by", account(ctx).UUID)
	return nil
}

// LatestBackup is the newest backup run recorded, nil for none. It is
// public, for nobody in particular: the liveness probe grades it.
func LatestBackup(ctx context.Context) (*obj.BackupLog, error) {
	row, err := store.Queries().V2GetLatestBackup(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &obj.BackupLog{ID: row.ID, StartedAt: row.StartedAt, FinishedAt: row.FinishedAt, Success: row.Success}, nil
}
