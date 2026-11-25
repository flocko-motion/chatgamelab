package wire

import (
	"cgl/core"
	"cgl/obj"
)

// SiteSettingsOf is the site's settings as v1 sent them.
func SiteSettingsOf(s *core.SiteSettings) obj.SystemSettings {
	v := obj.SystemSettings{
		ID: s.ID, CreatedAt: &s.CreatedAt, ModifiedAt: &s.ModifiedAt,
		DefaultAiQualityTier: s.DefaultTier, FreeUseAiQualityTier: s.FreeUseTier,
		PromptConstraintU13: s.Rules.U13, PromptConstraintU13p: s.Rules.U13p, PromptConstraintU18: s.Rules.U18,
		SessionRetentionHours: s.SessionRetentionHours,
		Mail: obj.MailSettings{
			Host: s.Mail.Host, Port: s.Mail.Port, User: s.Mail.User, From: s.Mail.From,
			PasswordSet: s.Mail.PasswordSet, ConsentCopyEmail: s.Mail.ConsentCopyEmail,
		},
	}
	d := core.DefaultSiteRules()
	v.PromptConstraintDefaults = obj.SiteRuleTexts{U13: *d.U13, U13p: *d.U13p, U18: *d.U18}
	if k := s.FreeUseKey; k != nil {
		v.FreeUseApiKeyID = &k.ID
		v.FreeUseApiKeyName, v.FreeUseApiKeyPlatform, v.FreeUseApiKeyWorking = k.Name, k.Platform, k.Working
	}
	return v
}
