package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
)

// GetFavouriteGames godoc
//
//	@Summary		Get user's favourite games
//	@Description	Returns the list of games the authenticated user has marked as favourites
//	@Tags			games
//	@Produce		json
//	@Success		200	{array}		obj.Game
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/favourites [get]
func GetFavouriteGames(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	games, err := db.GetFavouriteGames(r.Context(), user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get favourite games: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, games)
}

// AddFavouriteGame godoc
//
//	@Summary		Add game to favourites
//	@Description	Adds a game to the authenticated user's favourites
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	map[string]bool
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/favourite [post]
func AddFavouriteGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	if err := db.AddFavouriteGame(r.Context(), user.ID, gameID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to add favourite: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"favourite": true})
}

// RemoveFavouriteGame godoc
//
//	@Summary		Remove game from favourites
//	@Description	Removes a game from the authenticated user's favourites
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	map[string]bool
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/favourite [delete]
func RemoveFavouriteGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	if err := db.RemoveFavouriteGame(r.Context(), user.ID, gameID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to remove favourite: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"favourite": false})
}
