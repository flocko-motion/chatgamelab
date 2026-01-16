package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
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
