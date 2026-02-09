package routes

import (
	"encoding/json"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// GetSystemSettings godoc
//
//	@Summary		Get system settings
//	@Description	Returns the global system settings
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	obj.SystemSettings
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/system/settings [get]
func GetSystemSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := db.GetSystemSettings(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get system settings: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, settings)
}

// UpdateSystemSettingsRequest is the request body for updating system settings
type UpdateSystemSettingsRequest struct {
	DefaultAiModel string `json:"defaultAiModel"`
}

// UpdateSystemSettings godoc
//
//	@Summary		Update system settings
//	@Description	Updates the global system settings (admin only)
//	@Tags			system
//	@Accept			json
//	@Produce		json
//	@Param			request	body		UpdateSystemSettingsRequest	true	"Settings to update"
//	@Success		200		{object}	obj.SystemSettings
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/system/settings [patch]
func UpdateSystemSettings(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	// Require admin
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		httpx.WriteError(w, http.StatusForbidden, "Forbidden: admin access required")
		return
	}

	var req UpdateSystemSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DefaultAiModel == "" {
		httpx.WriteError(w, http.StatusBadRequest, "defaultAiModel is required")
		return
	}

	// Update the default AI model
	if err := db.UpdateDefaultAiModel(r.Context(), req.DefaultAiModel); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update settings: "+err.Error())
		return
	}

	// Return updated settings
	settings, err := db.GetSystemSettings(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated settings: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, settings)
}

// SetFreeUseApiKeyRequest is the request body for setting the system free-use API key
type SetFreeUseApiKeyRequest struct {
	ApiKeyID *uuid.UUID `json:"apiKeyId"`
}

// SetSystemFreeUseApiKey godoc
//
//	@Summary		Set system free-use API key
//	@Description	Sets or clears the global free-use API key (admin only).
//	@Description	The admin's own API key will be used directly.
//	@Description	Pass null apiKeyId to clear.
//	@Tags			system
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SetFreeUseApiKeyRequest	true	"API Key ID to set (null to clear)"
//	@Success		200		{object}	obj.SystemSettings
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/system/settings/free-use-key [patch]
func SetSystemFreeUseApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	// Require admin
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		httpx.WriteError(w, http.StatusForbidden, "Forbidden: admin access required")
		return
	}

	var req SetFreeUseApiKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := db.UpdateSystemSettingsFreeUseApiKey(r.Context(), req.ApiKeyID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update free-use key: "+err.Error())
		return
	}

	updatedSettings, err := db.GetSystemSettings(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated settings: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updatedSettings)
}
