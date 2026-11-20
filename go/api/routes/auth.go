// package: routes / authentication HTTP handlers
// type:    logic
// job:     handles registration, name-availability checks, logout, and participant login
// limits:  does not validate tokens (-> api/httpx); no route registration (-> router.go)
package routes

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"cgl/api/httpx"
	"cgl/constants"
	"cgl/db"
	"cgl/log"
)

// CheckNameAvailability godoc
//
//	@Summary		Check if name is available
//	@Description	Checks if a username is available for registration
//	@Tags			auth
//	@Produce		json
//	@Param			name	query		string	true	"Name to check"
//	@Success		200		{object}	map[string]bool
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Router			/auth/check-name [get]
func CheckNameAvailability(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Name parameter is required")
		return
	}

	if utf8.RuneCountInString(name) > 24 {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": false})
		return
	}

	// Check for profanity
	if constants.IsProfane(name) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": false, "profane": true})
		return
	}

	taken, err := db.IsNameTaken(r.Context(), name)
	if err != nil {
		log.Error("failed to check name availability", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to check name availability")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}
