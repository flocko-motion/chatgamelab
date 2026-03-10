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
		// ── Personal / org share ────────────────────────────────────────
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

		// If the selected API key share belongs to an institution, mark this as an org share
		apiKeyShare, err := db.GetApiKeyShareByID(r.Context(), user.ID, sponsorShareID)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid API key share")
			return
		}
		if apiKeyShare.Institution != nil {
			institutionID = &apiKeyShare.Institution.ID
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
	// - Workshop members with sharing permission (for workshop shares)
	// - Head/staff of workshop's institution (for workshop shares)
	isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
	isCreator := share.CreatedBy != nil && *share.CreatedBy == user.ID
	allowed := isOwner || isCreator

	if !allowed && share.WorkshopID != nil {
		userObj, err := db.GetUserByID(r.Context(), user.ID)
		if err == nil && userObj.Role != nil {
			// Head/staff of the workshop's institution
			if userObj.Role.Institution != nil &&
				share.InstitutionID != nil && *share.InstitutionID == userObj.Role.Institution.ID &&
				(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
				allowed = true
			}
			// Workshop member (participant in the same workshop)
			if !allowed && userObj.Role.Workshop != nil && *share.WorkshopID == userObj.Role.Workshop.ID {
				allowed = true
			}
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

// UpdateGameShareRequest is the request body for updating a share.
type UpdateGameShareRequest struct {
	MaxSessions *int `json:"maxSessions"` // null = unlimited
}

// UpdateGameShare godoc
//
//	@Summary		Update a game share
//	@Description	Updates settings (max sessions) on an existing share link.
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Game ID (UUID)"
//	@Param			shareId	path		string					true	"Share ID (UUID)"
//	@Param			request	body		UpdateGameShareRequest	true	"Updated settings"
//	@Success		200		{object}	GameShareResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/shares/{shareId} [patch]
func UpdateGameShare(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateGameShareRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

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

	// Same permission check as delete
	isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
	isCreator := share.CreatedBy != nil && *share.CreatedBy == user.ID
	allowed := isOwner || isCreator

	if !allowed && share.WorkshopID != nil {
		userObj, err := db.GetUserByID(r.Context(), user.ID)
		if err == nil && userObj.Role != nil {
			if userObj.Role.Institution != nil &&
				share.InstitutionID != nil && *share.InstitutionID == userObj.Role.Institution.ID &&
				(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
				allowed = true
			}
			if !allowed && userObj.Role.Workshop != nil && *share.WorkshopID == userObj.Role.Workshop.ID {
				allowed = true
			}
		}
	}

	if !allowed {
		httpx.WriteError(w, http.StatusForbidden, "Not authorized to update this share")
		return
	}

	updated, err := db.UpdateGameShareRemaining(r.Context(), shareID, req.MaxSessions)
	if err != nil {
		log.Warn("failed to update game share", "share_id", shareID, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update share")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toGameShareResponse(*updated))
}

// ── Private Share Status ─────────────────────────────────────────────────

// EnrichedGameShare extends a game share with resolved names and a source label.
type EnrichedGameShare struct {
	obj.GameShare
	ShareURL     string `json:"shareUrl"`
	Source       string `json:"source"`                 // "workshop", "organization", "personal"
	WorkshopName string `json:"workshopName,omitempty"` // set for workshop shares
	GameName     string `json:"gameName,omitempty"`     // set when returned from game-shares endpoint
}

// PrivateShareStatus returns all shares the requesting user has access to.
type PrivateShareStatus struct {
	Shares []EnrichedGameShare `json:"shares"`
}

// GetPrivateShareStatus godoc
//
//	@Summary		Get private share status
//	@Description	Returns all share links for a game that the requesting user has access to.
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

	// Load user for role/institution/workshop info
	userObj, err := db.GetUserByID(r.Context(), user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load user")
		return
	}

	seen := map[uuid.UUID]bool{}
	var result []EnrichedGameShare

	// 1. Personal/org shares (created by this user, not in a workshop)
	personalShares, _ := db.GetGameSharesByGameIDAndCreator(r.Context(), gameID, user.ID)
	for _, s := range personalShares {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		source := "personal"
		if s.InstitutionID != nil {
			source = "organization"
		}
		result = append(result, EnrichedGameShare{
			GameShare: s,
			ShareURL:  "/play/" + s.Token,
			Source:    source,
		})
	}

	// 2. Org shares (if user is head/staff with an institution)
	if userObj.Role != nil && userObj.Role.Institution != nil &&
		(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff || userObj.Role.Role == obj.RoleAdmin) {
		orgShares, _ := db.GetGameSharesByGameIDAndInstitution(r.Context(), gameID, userObj.Role.Institution.ID)
		for _, s := range orgShares {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			result = append(result, EnrichedGameShare{
				GameShare: s,
				ShareURL:  "/play/" + s.Token,
				Source:    "organization",
			})
		}
	}

	// 3. Workshop shares
	if userObj.Role != nil && userObj.Role.Workshop != nil {
		// User is currently in a workshop (participant or head/staff in workshop mode)
		wsShares, _ := db.GetGameSharesByGameIDAndWorkshop(r.Context(), gameID, userObj.Role.Workshop.ID)
		for _, s := range wsShares {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			result = append(result, EnrichedGameShare{
				GameShare:    s,
				ShareURL:     "/play/" + s.Token,
				Source:       "workshop",
				WorkshopName: userObj.Role.Workshop.Name,
			})
		}
	}
	// Head/staff outside workshop mode: show workshop shares from all workshops in their institution
	if userObj.Role != nil && userObj.Role.Workshop == nil && userObj.Role.Institution != nil &&
		(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
		workshops, _ := db.ListWorkshopsForInstitution(r.Context(), userObj.Role.Institution.ID)
		for _, ws := range workshops {
			wsShares, _ := db.GetGameSharesByGameIDAndWorkshop(r.Context(), gameID, ws.ID)
			for _, s := range wsShares {
				if seen[s.ID] {
					continue
				}
				seen[s.ID] = true
				result = append(result, EnrichedGameShare{
					GameShare:    s,
					ShareURL:     "/play/" + s.Token,
					Source:       "workshop",
					WorkshopName: ws.Name,
				})
			}
		}
	}

	if result == nil {
		result = []EnrichedGameShare{}
	}

	httpx.WriteJSON(w, http.StatusOK, PrivateShareStatus{Shares: result})
}
