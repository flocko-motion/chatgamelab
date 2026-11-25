package core

import (
	"context"
	"database/sql"
	"errors"
	"math"
	netmail "net/mail"
	"strings"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
	"cgl/mail"
	"cgl/obj"

	"github.com/google/uuid"
)

// The site is run by the admin: its youth-protection rules per age group,
// its AI quality tiers, its free-use key, which pays last for whoever has no
// other key, how long it keeps a session not played, and the mail server
// mails go through, whose password no one reads back. The free-use key is
// one of the admin's own: no admin spends someone else's key on the whole
// site. The site's rules are
// readable to everyone signed in, as the fallback below an organisation's.

// SiteRules are the site's youth-protection rules per age group.
type SiteRules struct {
	U13  *string // 13-17 without parental consent
	U13p *string // 13-17 with parental consent
	U18  *string // 18 and over
}

// The site's default rules (spec, Jugendschutz: the site's defaults): a level
// the admin never set, or gave back its default, plays under these, so the
// site rule is never empty. U13p and U18 are the same by default.
const (
	defaultSiteRuleU13   = "Erlaube spürbare Konflikte und stärkere Spannung; greife bei realistischer, detaillierter oder eskalierender Gewalt ein und verhindere Verherrlichung und Überforderung."
	defaultSiteRuleAdult = "Erlaube realistische, auch deutlich intensivere Inhalte; unterbinde exzessive oder verherrlichte Gewalt sowie extreme, entwürdigende oder diskriminierende Darstellungen konsequent."
)

// DefaultSiteRules are the site's default rules, a copy.
func DefaultSiteRules() SiteRules {
	u13, u13p, u18 := defaultSiteRuleU13, defaultSiteRuleAdult, defaultSiteRuleAdult
	return SiteRules{U13: &u13, U13p: &u13p, U18: &u18}
}

// siteRulesOf are the site's rules as they apply: what the admin set, else
// the default.
func siteRulesOf(s db.V2GetSystemSettingsRow) SiteRules {
	d := DefaultSiteRules()
	or := func(stored sql.NullString, def *string) *string {
		if stored.Valid && stored.String != "" {
			return &stored.String
		}
		return def
	}
	return SiteRules{U13: or(s.PromptConstraintU13, d.U13), U13p: or(s.PromptConstraintU13p, d.U13p), U18: or(s.PromptConstraintU18, d.U18)}
}

// ReadSiteRules are the site's rules as they apply: {Read, SiteRules, Everywhere}.
func ReadSiteRules(ctx context.Context) (SiteRules, error) {
	if !granted(ctx, rules.Read, rules.SiteRules, rules.Everywhere) {
		return SiteRules{}, forbidden("you may not read the site's rules")
	}
	s, err := store.Queries().V2GetSystemSettings(ctx)
	if err != nil {
		return SiteRules{}, err
	}
	return siteRulesOf(s), nil
}

// SiteSettings are the site's settings as the admin sees them.
type SiteSettings struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	ModifiedAt  time.Time
	DefaultTier string    // the tier wherever nothing else sets one
	FreeUseTier *string   // the free-use key's tier; nil: the default
	Rules       SiteRules // as they apply
	FreeUseKey  *FreeUseKey
	// SessionRetentionHours is how long a session not played is kept; 0
	// keeps every session.
	SessionRetentionHours int
	Mail                  MailSettings
}

// MailSettings are the mail server mails go through, as the admin sees
// them: its password only as whether one is set.
type MailSettings struct {
	Host        string // empty: mails are only logged
	Port        int
	User        string
	From        string // the sender; empty: the user
	PasswordSet bool
	// ConsentCopyEmail receives a copy of every organisation's consent;
	// empty: the site's support address.
	ConsentCopyEmail string
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
		Rules:                 siteRulesOf(s),
		SessionRetentionHours: int(s.SessionRetentionHours),
		Mail: MailSettings{
			Host: s.SmtpHost, Port: int(s.SmtpPort), User: s.SmtpUser, From: s.SmtpFrom,
			PasswordSet: s.SmtpPasswordSet, ConsentCopyEmail: s.ConsentCopyEmail,
		},
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
// stays, an empty one clears it. The default tier cannot be cleared; a rule
// cleared, or set to its default, is the default again. The session
// retention is whole hours, 0 or more.
type SiteSettingsChange struct {
	DefaultTier *string
	FreeUseTier *string
	U13         *string
	U13p        *string
	U18         *string

	SessionRetentionHours *int

	Mail *MailChange
}

// MailChange is the mail server as it is to be, all its fields at once but
// the password: a nil password stays, an empty one clears it.
type MailChange struct {
	Host             string
	Port             int // 0: the submission port, 587
	User             string
	From             string
	ConsentCopyEmail string
	Password         *string
}

// validate refuses a mail server no one can send through, and returns it
// trimmed.
func (m MailChange) validate() (MailChange, error) {
	m.Host, m.User = strings.TrimSpace(m.Host), strings.TrimSpace(m.User)
	m.From, m.ConsentCopyEmail = strings.TrimSpace(m.From), strings.TrimSpace(m.ConsentCopyEmail)
	if m.Port == 0 {
		m.Port = 587
	}
	if m.Port < 1 || m.Port > 65535 {
		return m, refuse("the mail server's port is 1 to 65535")
	}
	if strings.ContainsAny(m.Host, " /:") {
		return m, refuse("the mail server is a host name, without port or scheme")
	}
	if m.From != "" {
		if _, err := netmail.ParseAddress(m.From); err != nil {
			return m, refuse("the sender is an email address, with a name if you like")
		}
	}
	if m.ConsentCopyEmail != "" {
		if a, err := netmail.ParseAddress(m.ConsentCopyEmail); err != nil || a.Address != m.ConsentCopyEmail {
			return m, refuse("the address for copies of consents is an email address")
		}
	}
	return m, nil
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
	if h := c.SessionRetentionHours; h != nil && (*h < 0 || *h > math.MaxInt32) {
		return nil, refuse("the session retention is whole hours, 0 or more")
	}
	if c.Mail != nil {
		m, err := c.Mail.validate()
		if err != nil {
			return nil, err
		}
		c.Mail = &m
	}
	err := store.InTx(ctx, func(q *db.Queries) error {
		if c.DefaultTier != nil {
			if err := q.V2UpdateDefaultAiQualityTier(ctx, *c.DefaultTier); err != nil {
				return err
			}
		}
		if c.SessionRetentionHours != nil {
			if err := q.V2UpdateSessionRetentionHours(ctx, int32(*c.SessionRetentionHours)); err != nil {
				return err
			}
		}
		if m := c.Mail; m != nil {
			if err := q.V2UpdateMailSettings(ctx, db.V2UpdateMailSettingsParams{
				Host: m.Host, Port: int32(m.Port), Username: m.User, Sender: m.From, ConsentCopyEmail: m.ConsentCopyEmail,
			}); err != nil {
				return err
			}
			if m.Password != nil {
				if err := q.V2UpdateMailPassword(ctx, *m.Password); err != nil {
					return err
				}
			}
		}
		d := DefaultSiteRules()
		for _, set := range []struct {
			value *string
			def   *string // a value equal to it stores nothing: the default applies
			write func(context.Context, sql.NullString) error
		}{
			{c.FreeUseTier, nil, q.V2UpdateFreeUseAiQualityTier},
			{c.U13, d.U13, q.V2UpdatePromptConstraintU13},
			{c.U13p, d.U13p, q.V2UpdatePromptConstraintU13p},
			{c.U18, d.U18, q.V2UpdatePromptConstraintU18},
		} {
			if set.value == nil {
				continue
			}
			value := strings.TrimSpace(*set.value)
			keep := value != "" && (set.def == nil || value != *set.def)
			if err := set.write(ctx, sql.NullString{String: value, Valid: keep}); err != nil {
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

// MayManageUpdates refuses seeing the site's available versions and
// requesting an upgrade, unless the caller runs the site: {Manage, Site,
// Everywhere}, the admin. schemaf's update endpoints ask it on every request.
func MayManageUpdates(ctx context.Context) error {
	if !granted(ctx, rules.Manage, rules.Site, rules.Everywhere) {
		return forbidden("only the admin manages updates")
	}
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

// mailServer is the mail server mails go through, password included, and the
// address that receives copies of consents (empty: the support address's).
// Never shown to anyone: for sending only.
func mailServer(ctx context.Context) (mail.Server, string, error) {
	m, err := store.Queries().V2MailServer(ctx)
	if err != nil {
		return mail.Server{}, "", err
	}
	return mail.Server{
		Host: m.SmtpHost, Port: int(m.SmtpPort), User: m.SmtpUser, Password: m.SmtpPassword, From: m.SmtpFrom,
	}, m.ConsentCopyEmail, nil
}
