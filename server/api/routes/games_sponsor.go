package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// SponsorGameRequest represents the request body for sponsoring a game
type SponsorGameRequest struct {
	ShareID uuid.UUID `json:"shareId"`
}

// SetGameSponsor godoc
//
//	@Summary		Sponsor a game
//	@Description	Sets a public sponsorship on a game using an API key share
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Game ID (UUID)"
//	@Param			request	body		SponsorGameRequest	true	"Sponsor request"
//	@Success		200		{object}	obj.Game
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	httpx.ErrorResponse	"Game not found"
//	@Security		BearerAuth
//	@Router			/games/{id}/sponsor [put]
func SetGameSponsor(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	var req SponsorGameRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.ShareID == uuid.Nil {
		httpx.WriteError(w, http.StatusBadRequest, "Missing required field: shareId")
		return
	}

	log.Debug("setting game sponsor", "game_id", gameID, "user_id", user.ID, "share_id", req.ShareID)

	if err := db.SetGamePublicSponsorship(r.Context(), user.ID, gameID, req.ShareID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to set sponsor: "+err.Error())
		return
	}

	g, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, g)
}

// RemoveGameSponsor godoc
//
//	@Summary		Remove game sponsorship
//	@Description	Removes the public sponsorship from a game
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	obj.Game
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Security		BearerAuth
//	@Router			/games/{id}/sponsor [delete]
func RemoveGameSponsor(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("removing game sponsor", "game_id", gameID, "user_id", user.ID)

	if err := db.ClearGamePublicSponsorship(r.Context(), user.ID, gameID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to remove sponsor: "+err.Error())
		return
	}

	g, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, g)
}

// GetAvailableKeys godoc
//
//	@Summary		Get available API keys for a game
//	@Description	Returns a prioritized list of API keys available to the user for playing this game
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{array}		obj.AvailableKey
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	httpx.ErrorResponse	"Game not found"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/available-keys [get]
func GetAvailableKeys(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("getting available keys for game", "game_id", gameID, "user_id", user.ID)

	keys, err := db.GetAvailableKeysForGame(r.Context(), user.ID, gameID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get available keys: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, keys)
}

// GetApiKeyStatus godoc
//
//	@Summary		Check API key availability
//	@Description	Checks whether an API key can be reso        lved for the current user and game.
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	map[string]bool
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Security		BearerAuth
//	@Router			/games/{id}/api-key-status [get]
func GetApiKeyStatus(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)
	available := game.IsApiKeyAvailable(r.Context(), user.ID, gameID)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"available": available})
}
