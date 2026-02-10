package routes

import (
	"io"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
)

// GetGameYAML godoc
//
//	@Summary		Export game as YAML
//	@Description	Exports a game's configuration as YAML format
//	@Tags			games
//	@Produce		application/x-yaml
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	obj.Game
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	httpx.ErrorResponse	"Game not found"
//	@Security		BearerAuth
//	@Router			/games/{id}/yaml [get]
func GetGameYAML(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("exporting game as YAML", "game_id", gameID, "user_id", user.ID)

	game, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteYAML(w, http.StatusOK, game)
}

// UpdateGameYAML godoc
//
//	@Summary		Import game from YAML
//	@Description	Updates a game's configuration from YAML format
//	@Tags			games
//	@Accept			application/x-yaml
//	@Produce		json
//	@Param			id		path	string	true	"Game ID (UUID)"
//	@Param			yaml	body	string	true	"Game YAML content"
//	@Success		200		{object}	obj.Game
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse	"Failed to update"
//	@Security		BearerAuth
//	@Router			/games/{id}/yaml [put]
func UpdateGameYAML(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		log.Debug("UpdateGameYAML: invalid game ID", "error", err)
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("updating game from YAML", "game_id", gameID, "user_id", user.ID)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("UpdateGameYAML: failed to read body", "error", err)
		httpx.WriteError(w, http.StatusBadRequest, "Failed to read body: "+err.Error())
		return
	}

	log.Debug("UpdateGameYAML: body read", "body_length", len(body), "body_preview", string(body[:min(200, len(body))]))

	if err := db.UpdateGameYaml(r.Context(), user.ID, gameID, string(body)); err != nil {
		log.Error("UpdateGameYAML: db.UpdateGameYaml failed", "error", err, "game_id", gameID, "user_id", user.ID)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update game: "+err.Error())
		return
	}

	updated, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updated)
}
