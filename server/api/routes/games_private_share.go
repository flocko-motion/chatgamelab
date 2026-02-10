package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// ── Private Share Management ────────────────────────────────────────────────

// PrivateShareStatus is the response for the private share status endpoint.
type PrivateShareStatus struct {
	Enabled                       bool       `json:"enabled"`
	ShareURL                      string     `json:"shareUrl,omitempty"`
	Token                         string     `json:"token,omitempty"`
	Remaining                     *int       `json:"remaining"` // null = unlimited
	PrivateSponsoredApiKeyShareID *uuid.UUID `json:"privateSponsoredApiKeyShareId,omitempty"`
}

// PrivateShareRequest is the request body for enabling/configuring a private share.
type PrivateShareRequest struct {
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"` // required to enable
	MaxSessions       *int       `json:"maxSessions"`       // null = unlimited
}

// GetPrivateShareStatus godoc
//
//	@Summary		Get private share status
//	@Description	Returns the current private share configuration for a game.
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	PrivateShareStatus
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/private-share [get]
func GetPrivateShareStatus(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)
	g, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	enabled := g.PrivateShareHash != nil && g.PrivateSponsoredApiKeyShareID != nil
	status := PrivateShareStatus{
		Enabled:                       enabled,
		Token:                         stringDeref(g.PrivateShareHash),
		Remaining:                     g.PrivateShareRemaining,
		PrivateSponsoredApiKeyShareID: g.PrivateSponsoredApiKeyShareID,
	}
	if enabled {
		status.ShareURL = "/play/" + *g.PrivateShareHash
	}

	httpx.WriteJSON(w, http.StatusOK, status)
}

// EnablePrivateShare godoc
//
//	@Summary		Enable or configure private share
//	@Description	Enables private sharing for a game with a sponsored API key and optional session limit.
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Game ID (UUID)"
//	@Param			request	body		PrivateShareRequest	true	"Share configuration"
//	@Success		200		{object}	PrivateShareStatus
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/private-share [post]
func EnablePrivateShare(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	var req PrivateShareRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.SponsorKeyShareID == nil {
		httpx.WriteError(w, http.StatusBadRequest, "sponsorKeyShareId is required")
		return
	}

	g, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	// Update the game with private share config
	g.PrivateSponsoredApiKeyShareID = req.SponsorKeyShareID
	g.PrivateShareRemaining = req.MaxSessions

	if err := db.UpdateGame(r.Context(), user.ID, g); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update game: "+err.Error())
		return
	}

	// Reload to get the generated hash
	g, _ = db.GetGameByID(r.Context(), &user.ID, gameID)

	enabled := g.PrivateShareHash != nil && g.PrivateSponsoredApiKeyShareID != nil
	status := PrivateShareStatus{
		Enabled:                       enabled,
		Token:                         stringDeref(g.PrivateShareHash),
		Remaining:                     g.PrivateShareRemaining,
		PrivateSponsoredApiKeyShareID: g.PrivateSponsoredApiKeyShareID,
	}
	if enabled {
		status.ShareURL = "/play/" + *g.PrivateShareHash
	}

	httpx.WriteJSON(w, http.StatusOK, status)
}

// RevokePrivateShare godoc
//
//	@Summary		Revoke private share
//	@Description	Disables private sharing by clearing the share token and sponsor key.
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	PrivateShareStatus
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/private-share [delete]
func RevokePrivateShare(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	g, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	// Clean up guest users, sessions, and messages created via this share link
	guestCount, _ := db.CountGuestUsersByGameID(r.Context(), gameID)
	if guestCount > 0 {
		log.Info("revoking private share: cleaning up guest data", "game_id", gameID, "guest_users", guestCount)
		if err := db.DeleteGuestDataByGameID(r.Context(), gameID); err != nil {
			log.Warn("failed to clean up guest data on share revoke", "game_id", gameID, "error", err)
			// Non-fatal — continue with revoke even if cleanup fails
		} else {
			log.Info("revoking private share: removed guest data", "game_id", gameID, "guest_users_removed", guestCount)
		}
	}

	// Clear all private share fields — hash is removed, a new one will be generated on next enable
	g.PrivateShareHash = nil
	g.PrivateSponsoredApiKeyShareID = nil
	g.PrivateShareRemaining = nil

	if err := db.UpdateGame(r.Context(), user.ID, g); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to revoke share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, PrivateShareStatus{Enabled: false})
}

func stringDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
