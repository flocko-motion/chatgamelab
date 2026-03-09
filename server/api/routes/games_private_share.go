package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// ── Game Share Management ─────────────────────────────────────────────────

// GameShareResponse wraps a single game share with its share URL.
type GameShareResponse struct {
	obj.GameShare
	ShareURL string `json:"shareUrl"`
}

func toGameShareResponse(gs obj.GameShare) GameShareResponse {
	return GameShareResponse{
		GameShare: gs,
		ShareURL:  "/play/" + gs.Token,
	}
}

// CreateGameShareRequest is the request body for creating a share.
type CreateGameShareRequest struct {
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"` // required for personal shares; ignored for workshop shares
	MaxSessions       *int       `json:"maxSessions"`       // null = unlimited
	WorkshopID        *uuid.UUID `json:"workshopId"`        // set to create a workshop share (uses workshop default key)
}

// ListGameShares godoc
//
//	@Summary		List game shares
//	@Description	Returns all share links for a game.
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{array}		GameShareResponse
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/shares [get]
func ListGameShares(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)
	_, err = db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	shares, _ := db.GetGameSharesByGameID(r.Context(), gameID)

	result := make([]GameShareResponse, len(shares))
	for i, s := range shares {
		result[i] = toGameShareResponse(s)
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}

// CreateGameShare godoc
//
//	@Summary		Create a game share link
//	@Description	Creates a share link for a game. For workshop shares, the workshop default key is used automatically.
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Game ID (UUID)"
//	@Param			request	body		CreateGameShareRequest	true	"Share configuration"
//	@Success		200		{object}	GameShareResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/shares [post]
func CreateGameShare(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	var req CreateGameShareRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	game, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	var sponsorShareID uuid.UUID
	var institutionID, workshopID *uuid.UUID

	if req.WorkshopID != nil {
		// ── Workshop share ──────────────────────────────────────────────
		// Validate workshop access and get workshop default key
		workshop, err := db.GetWorkshopByID(r.Context(), user.ID, *req.WorkshopID)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Workshop not found")
			return
		}

		// Permission: head/staff can always share; participants need AllowGameSharing
		userObj, err := db.GetUserByID(r.Context(), user.ID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to load user")
			return
		}
		canEditAll := userObj.Role != nil && (userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff || userObj.Role.Role == obj.RoleAdmin)
		if !canEditAll {
			if !workshop.AllowGameSharing {
				httpx.WriteError(w, http.StatusForbidden, "Game sharing is not enabled for this workshop")
				return
			}
		}

		// Game must belong to this workshop
		if game.WorkshopID == nil || *game.WorkshopID != *req.WorkshopID {
			httpx.WriteError(w, http.StatusBadRequest, "Game does not belong to this workshop")
			return
		}

		// Reuse existing workshop share if one exists
		existing, err := db.GetWorkshopGameShare(r.Context(), gameID, *req.WorkshopID)
		if err == nil {
			httpx.WriteJSON(w, http.StatusOK, toGameShareResponse(*existing))
			return
		}

		// Workshop must have a default API key
		if workshop.DefaultApiKeyShareID == nil {
			httpx.WriteError(w, http.StatusBadRequest, "Workshop has no default API key configured")
			return
		}

		sponsorShareID = *workshop.DefaultApiKeyShareID
		workshopID = req.WorkshopID
		if workshop.Institution != nil {
			institutionID = &workshop.Institution.ID
		}
	} else {
		// ── Personal / public share ─────────────────────────────────────
		if req.SponsorKeyShareID == nil {
			httpx.WriteError(w, http.StatusBadRequest, "sponsorKeyShareId is required")
			return
		}
		sponsorShareID = *req.SponsorKeyShareID

		// Permission: owner can share own game; anyone can share a public game
		isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
		if !isOwner && !game.Public {
			httpx.WriteError(w, http.StatusForbidden, "Only the owner can share a non-public game")
			return
		}
	}

	// Create the game share
	gameShare, err := db.CreateGameShare(r.Context(), user.ID, gameID, sponsorShareID, institutionID, workshopID, req.MaxSessions)
	if err != nil {
		log.Warn("failed to create game share", "game_id", gameID, "error", err)
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create share link: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toGameShareResponse(*gameShare))
}

// DeleteGameShareByID godoc
//
//	@Summary		Delete a specific game share
//	@Description	Revokes a specific share link and cleans up guest data.
//	@Tags			games
//	@Produce		json
//	@Param			id			path	string	true	"Game ID (UUID)"
//	@Param			shareId		path	string	true	"Share ID (UUID)"
//	@Success		204
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/shares/{shareId} [delete]
func DeleteGameShareByID(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	shareID, err := httpx.PathParamUUID(r, "shareId")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	user := httpx.UserFromRequest(r)

	game, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	share, err := db.GetGameShareByID(r.Context(), shareID)
	if err != nil || share.GameID != gameID {
		httpx.WriteError(w, http.StatusNotFound, "Share not found")
		return
	}

	// Permission check: who can revoke?
	// - Share creator
	// - Game owner
	// - Head/staff of workshop's institution (for workshop shares)
	isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
	isCreator := share.CreatedBy != nil && *share.CreatedBy == user.ID
	allowed := isOwner || isCreator

	if !allowed && share.WorkshopID != nil {
		// Check if user is head/staff of the workshop's institution
		userObj, err := db.GetUserByID(r.Context(), user.ID)
		if err == nil && userObj.Role != nil && userObj.Role.Institution != nil &&
			share.InstitutionID != nil && *share.InstitutionID == userObj.Role.Institution.ID &&
			(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
			allowed = true
		}
	}

	if !allowed {
		httpx.WriteError(w, http.StatusForbidden, "Not authorized to revoke this share")
		return
	}

	if err := db.DeleteGameShare(r.Context(), shareID); err != nil {
		log.Warn("failed to delete game share", "game_id", gameID, "share_id", shareID, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete share")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Legacy endpoints (backwards compatibility) ──────────────────────────

// PrivateShareRequest is the legacy request body for enabling/configuring a private share.
type PrivateShareRequest struct {
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"` // required to enable
	MaxSessions       *int       `json:"maxSessions"`       // null = unlimited
}

// PrivateShareStatus is the legacy response for the private share status endpoint.
type PrivateShareStatus struct {
	Enabled   bool            `json:"enabled"`
	ShareURL  string          `json:"shareUrl,omitempty"`
	Token     string          `json:"token,omitempty"`
	Remaining *int            `json:"remaining"` // null = unlimited
	ShareID   *uuid.UUID      `json:"shareId,omitempty"`
	Shares    []obj.GameShare `json:"shares,omitempty"`
}

// GetPrivateShareStatus godoc
//
//	@Summary		Get private share status (legacy)
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
	_, err = db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	// Filter shares by context: workshop shares or personal shares only
	var shares []obj.GameShare
	workshopIDStr := r.URL.Query().Get("workshopId")
	if workshopIDStr != "" {
		wid, err := uuid.Parse(workshopIDStr)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid workshopId")
			return
		}
		shares, _ = db.GetGameSharesByGameIDAndWorkshop(r.Context(), gameID, wid)
	} else {
		// Personal context: only return shares created by this user
		shares, _ = db.GetGameSharesByGameIDAndCreator(r.Context(), gameID, user.ID)
	}
	enabled := len(shares) > 0

	status := PrivateShareStatus{
		Enabled: enabled,
		Shares:  shares,
	}
	if enabled {
		// Use the first share for backwards-compatible single-share fields
		status.Token = shares[0].Token
		status.Remaining = shares[0].Remaining
		status.ShareID = &shares[0].ID
		status.ShareURL = "/play/" + shares[0].Token
	}

	httpx.WriteJSON(w, http.StatusOK, status)
}

// EnablePrivateShare godoc
//
//	@Summary		Enable private share (legacy)
//	@Description	Creates a share link. Delegates to CreateGameShare internally.
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Game ID (UUID)"
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

	_, err = db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	gameShare, err := db.CreateGameShare(r.Context(), user.ID, gameID, *req.SponsorKeyShareID, nil, nil, req.MaxSessions)
	if err != nil {
		log.Warn("failed to create game share", "game_id", gameID, "error", err)
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to set up private share: "+err.Error())
		return
	}

	status := PrivateShareStatus{
		Enabled:   true,
		Token:     gameShare.Token,
		Remaining: gameShare.Remaining,
		ShareID:   &gameShare.ID,
		ShareURL:  "/play/" + gameShare.Token,
	}

	httpx.WriteJSON(w, http.StatusOK, status)
}

// RevokePrivateShare godoc
//
//	@Summary		Revoke private share (legacy)
//	@Description	Deletes all game shares for the game.
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

	_, err = db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	shares, _ := db.GetGameSharesByGameID(r.Context(), gameID)
	for _, s := range shares {
		if err := db.DeleteGameShare(r.Context(), s.ID); err != nil {
			log.Warn("failed to delete game share on revoke", "game_id", gameID, "share_id", s.ID, "error", err)
		}
	}

	httpx.WriteJSON(w, http.StatusOK, PrivateShareStatus{Enabled: false})
}
