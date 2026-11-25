// package: api / schemaf endpoints
// type:    logic
// job:     reads and updates site-wide settings, constraints and the free-use API key, and restarts the server
// limits:  the secure core decides and stores (core/site.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SystemSettingsResp is the site-wide settings.
type SystemSettingsResp = obj.SystemSettings

// GetSystemConstraintsEndpoint returns the three site-wide age-based prompt
// constraint texts.
//
// Any signed-in user may read them, so organization settings can show the
// fallback that applies when an organization has none of its own.
type GetSystemConstraintsEndpoint struct{}

func (GetSystemConstraintsEndpoint) Method() string { return "GET" }
func (GetSystemConstraintsEndpoint) Path() string   { return "/api/system/constraints" }
func (GetSystemConstraintsEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (GetSystemConstraintsEndpoint) Handle(ctx context.Context, _ SystemConstraintsReq) (SystemConstraintsResp, error) {
	r, err := core.ReadSiteRules(ctx)
	return SystemConstraintsResp{PromptConstraintU13: r.U13, PromptConstraintU13p: r.U13p, PromptConstraintU18: r.U18}, err
}

// SystemConstraintsReq takes no parameters.
type SystemConstraintsReq struct{}

// SystemConstraintsResp is the site-wide age-based prompt constraint texts.
type SystemConstraintsResp struct {
	PromptConstraintU13  *string `json:"promptConstraintU13,omitempty"`
	PromptConstraintU13p *string `json:"promptConstraintU13p,omitempty"`
	PromptConstraintU18  *string `json:"promptConstraintU18,omitempty"`
}

// GetSystemSettingsEndpoint returns the site-wide settings to an admin.
type GetSystemSettingsEndpoint struct{}

func (GetSystemSettingsEndpoint) Method() string             { return "GET" }
func (GetSystemSettingsEndpoint) Path() string               { return "/api/system/settings" }
func (GetSystemSettingsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetSystemSettingsEndpoint) Handle(ctx context.Context, _ SystemSettingsReq) (SystemSettingsResp, error) {
	return siteSettingsView(core.ReadSiteSettings(ctx))
}

// SystemSettingsReq takes no parameters.
type SystemSettingsReq struct{}

// UpdateSystemSettingsEndpoint updates the site-wide settings; admins only.
//
// Only the fields sent change. The default AI quality tier cannot be empty;
// an empty free-use tier clears it; a constraint emptied, or set to its
// default, is the default again. The session retention is whole hours, 0 or
// more. The mail server changes as a whole, but for its password: absent, it
// stays; empty, it is cleared. Answers with the settings as they apply; the
// password never.
type UpdateSystemSettingsEndpoint struct{}

func (UpdateSystemSettingsEndpoint) Method() string { return "PATCH" }
func (UpdateSystemSettingsEndpoint) Path() string   { return "/api/system/settings" }
func (UpdateSystemSettingsEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (UpdateSystemSettingsEndpoint) Handle(ctx context.Context, req UpdateSystemSettingsReq) (SystemSettingsResp, error) {
	return siteSettingsView(core.EditSiteSettings(ctx, core.SiteSettingsChange{
		DefaultTier: req.DefaultAiQualityTier, FreeUseTier: req.FreeUseAiQualityTier,
		U13: req.PromptConstraintU13, U13p: req.PromptConstraintU13p, U18: req.PromptConstraintU18,
		SessionRetentionHours: req.SessionRetentionHours,
		Mail:                  mailChange(req.Mail),
	}))
}

// mailChange is the mail server the request sets; nil: it stays.
func mailChange(m *MailSettingsReq) *core.MailChange {
	if m == nil {
		return nil
	}
	return &core.MailChange{
		Host: m.Host, Port: m.Port, User: m.User, From: m.From,
		ConsentCopyEmail: m.ConsentCopyEmail, Password: m.Password,
	}
}

// siteSettingsView is the site's settings as v1 sent them.
func siteSettingsView(s *core.SiteSettings, err error) (SystemSettingsResp, error) {
	if err != nil {
		return SystemSettingsResp{}, err
	}
	return wire.SiteSettingsOf(s), nil
}

// UpdateSystemSettingsReq is the settings to change; absent fields stay.
type UpdateSystemSettingsReq struct {
	DefaultAiQualityTier *string `json:"defaultAiQualityTier"`
	FreeUseAiQualityTier *string `json:"freeUseAiQualityTier"`
	// PromptConstraintU13 applies to players aged 13-17 without parental consent.
	PromptConstraintU13 *string `json:"promptConstraintU13"`
	// PromptConstraintU13p applies to players aged 13-17 with parental consent.
	PromptConstraintU13p *string `json:"promptConstraintU13p"`
	// PromptConstraintU18 applies to players aged 18 and over.
	PromptConstraintU18 *string `json:"promptConstraintU18"`
	// SessionRetentionHours is how long a session not played is kept, in
	// hours; 0 keeps every session.
	SessionRetentionHours *int `json:"sessionRetentionHours"`
	// Mail is the mail server; absent: it stays.
	Mail *MailSettingsReq `json:"mail"`
}

// MailSettingsReq is the mail server as it is to be.
type MailSettingsReq struct {
	Host string `json:"host"` // empty: mails are only logged
	Port int    `json:"port"` // 0: 587
	User string `json:"user"`
	From string `json:"from"` // the sender; empty: the user
	// ConsentCopyEmail receives a copy of every organisation's consent;
	// empty: the site's support address.
	ConsentCopyEmail string `json:"consentCopyEmail"`
	// Password is the login's password: absent, it stays; empty, it is
	// cleared.
	Password *string `json:"password"`
}

// SetSystemFreeUseApiKeyEndpoint sets or clears the site-wide free-use API
// key; admins only.
//
// Only a key of the admin's own; someone else's is not found. A null
// apiKeyId clears it.
type SetSystemFreeUseApiKeyEndpoint struct{}

func (SetSystemFreeUseApiKeyEndpoint) Method() string { return "PATCH" }
func (SetSystemFreeUseApiKeyEndpoint) Path() string   { return "/api/system/settings/free-use-key" }
func (SetSystemFreeUseApiKeyEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetSystemFreeUseApiKeyEndpoint) Handle(ctx context.Context, req SetSystemFreeUseApiKeyReq) (SystemSettingsResp, error) {
	return siteSettingsView(core.SetFreeUseKey(ctx, req.ApiKeyID))
}

// SetSystemFreeUseApiKeyReq is the API key to use for free play, or null to clear it.
type SetSystemFreeUseApiKeyReq struct {
	ApiKeyID *uuid.UUID `json:"apiKeyId"`
}

// RestartEndpoint has the server restart; admins only.
type RestartEndpoint struct{}

func (RestartEndpoint) Method() string             { return "POST" }
func (RestartEndpoint) Path() string               { return "/api/restart" }
func (RestartEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (RestartEndpoint) Handle(ctx context.Context, _ RestartReq) (RestartResp, error) {
	if err := core.MayRestart(ctx); err != nil {
		return "", err
	}
	go func() {
		// Waits for the response to be sent.
		<-ctx.Done()
		log.Info("shutting down server for restart")
	}()
	return "Server will restart shortly", nil
}

// RestartReq takes no parameters.
type RestartReq struct{}

// RestartResp is a confirmation for the admin.
type RestartResp = string
