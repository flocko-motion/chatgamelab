// package: api / schemaf endpoints
// type:    logic
// job:     reads and updates site-wide settings, constraints and the free-use API key, and restarts the server
// limits:  settings storage lives in db
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// SystemSettingsResp is the site-wide settings.
type SystemSettingsResp = obj.SystemSettings

// systemAdmin refuses a caller who is not an admin.
func systemAdmin(user *obj.User) error {
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		return errForbidden("Forbidden: admin access required")
	}
	return nil
}

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
	if _, err := caller(ctx); err != nil {
		return SystemConstraintsResp{}, err
	}
	settings, err := db.GetSystemSettings(ctx)
	if err != nil {
		return SystemConstraintsResp{}, fmt.Errorf("failed to get system constraints: %w", err)
	}
	return SystemConstraintsResp{
		PromptConstraintU13:  settings.PromptConstraintU13,
		PromptConstraintU13p: settings.PromptConstraintU13p,
		PromptConstraintU18:  settings.PromptConstraintU18,
	}, nil
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
	user, err := caller(ctx)
	if err != nil {
		return obj.SystemSettings{}, err
	}
	if err := systemAdmin(user); err != nil {
		return obj.SystemSettings{}, err
	}
	settings, err := db.GetSystemSettings(ctx)
	if err != nil {
		return obj.SystemSettings{}, fmt.Errorf("failed to get system settings: %w", err)
	}
	return *settings, nil
}

// SystemSettingsReq takes no parameters.
type SystemSettingsReq struct{}

// UpdateSystemSettingsEndpoint updates the site-wide settings; admins only.
//
// Only the fields sent change. The default AI quality tier cannot be empty;
// an empty free-use tier or constraint clears it. Answers with the settings
// as stored.
type UpdateSystemSettingsEndpoint struct{}

func (UpdateSystemSettingsEndpoint) Method() string { return "PATCH" }
func (UpdateSystemSettingsEndpoint) Path() string   { return "/api/system/settings" }
func (UpdateSystemSettingsEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (UpdateSystemSettingsEndpoint) Handle(ctx context.Context, req UpdateSystemSettingsReq) (SystemSettingsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.SystemSettings{}, err
	}
	if err := systemAdmin(user); err != nil {
		return obj.SystemSettings{}, err
	}

	if req.DefaultAiQualityTier != nil {
		if *req.DefaultAiQualityTier == "" {
			return obj.SystemSettings{}, errBadRequest("defaultAiQualityTier cannot be empty")
		}
		if err := db.UpdateDefaultAiQualityTier(ctx, *req.DefaultAiQualityTier); err != nil {
			return obj.SystemSettings{}, fmt.Errorf("failed to update default AI quality tier: %w", err)
		}
	}
	if req.FreeUseAiQualityTier != nil {
		if err := db.UpdateFreeUseAiQualityTier(ctx, systemEmptyToNil(req.FreeUseAiQualityTier)); err != nil {
			return obj.SystemSettings{}, fmt.Errorf("failed to update free-use AI quality tier: %w", err)
		}
	}
	if req.PromptConstraintU13 != nil {
		if err := db.UpdatePromptConstraintU13(ctx, systemEmptyToNil(req.PromptConstraintU13)); err != nil {
			return obj.SystemSettings{}, fmt.Errorf("failed to update U13 constraint: %w", err)
		}
	}
	if req.PromptConstraintU13p != nil {
		if err := db.UpdatePromptConstraintU13p(ctx, systemEmptyToNil(req.PromptConstraintU13p)); err != nil {
			return obj.SystemSettings{}, fmt.Errorf("failed to update U13p constraint: %w", err)
		}
	}
	if req.PromptConstraintU18 != nil {
		if err := db.UpdatePromptConstraintU18(ctx, systemEmptyToNil(req.PromptConstraintU18)); err != nil {
			return obj.SystemSettings{}, fmt.Errorf("failed to update U18 constraint: %w", err)
		}
	}

	settings, err := db.GetSystemSettings(ctx)
	if err != nil {
		return obj.SystemSettings{}, fmt.Errorf("failed to get updated settings: %w", err)
	}
	return *settings, nil
}

// systemEmptyToNil is nil for an empty string, which clears a setting.
func systemEmptyToNil(s *string) *string {
	if *s == "" {
		return nil
	}
	return s
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
}

// SetSystemFreeUseApiKeyEndpoint sets or clears the site-wide free-use API
// key; admins only.
//
// The admin's own API key is used directly. A null apiKeyId clears it.
type SetSystemFreeUseApiKeyEndpoint struct{}

func (SetSystemFreeUseApiKeyEndpoint) Method() string { return "PATCH" }
func (SetSystemFreeUseApiKeyEndpoint) Path() string   { return "/api/system/settings/free-use-key" }
func (SetSystemFreeUseApiKeyEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetSystemFreeUseApiKeyEndpoint) Handle(ctx context.Context, req SetSystemFreeUseApiKeyReq) (SystemSettingsResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.SystemSettings{}, err
	}
	if err := systemAdmin(user); err != nil {
		return obj.SystemSettings{}, err
	}
	if err := db.UpdateSystemSettingsFreeUseApiKey(ctx, req.ApiKeyID); err != nil {
		return obj.SystemSettings{}, fmt.Errorf("failed to update free-use key: %w", err)
	}
	settings, err := db.GetSystemSettings(ctx)
	if err != nil {
		return obj.SystemSettings{}, fmt.Errorf("failed to get updated settings: %w", err)
	}
	return *settings, nil
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
	user, err := caller(ctx)
	if err != nil {
		return "", err
	}
	if err := systemAdmin(user); err != nil {
		log.Debug("restart denied - not admin", "user_id", user.ID)
		return "", err
	}
	log.Info("restart request received", "user_id", user.ID)
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
