package routes

import (
	"io"
	"net/http"
	"strings"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

type CreateGameRequest struct {
	Name string `json:"name"`
}

// GetGames godoc
//
//	@Summary		List games
//	@Description	Returns a list of games. If authenticated, includes user's private games.
//	@Tags			games
//	@Produce		json
//	@Success		200	{array}		obj.Game
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/games [get]
func GetGames(w http.ResponseWriter, r *http.Request) {
	user := httpx.MaybeUserFromRequest(r)

	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	log.Debug("listing games", "user_id", userID)
	games, err := db.GetGames(r.Context(), userID, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get games: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, games)
}

// CreateGame godoc
//
//	@Summary		Create game
//	@Description	Creates a new game from JSON or YAML. Accepts either a simple name or full game object.
//	@Tags			games
//	@Accept			json,application/x-yaml
//	@Produce		json
//	@Param			request	body		CreateGameRequest	true	"Create game request (JSON or YAML)"
//	@Success		200		{object}	obj.Game
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/new [post]
func CreateGame(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	// Try to parse as full game object first
	var game obj.Game
	if err := httpx.ReadJSONOrYAML(r, &game); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate that at least name is provided
	if strings.TrimSpace(game.Name) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Missing required field: name")
		return
	}

	log.Debug("creating game", "user_id", user.ID, "name", game.Name)
	if err := db.CreateGame(r.Context(), user.ID, &game); err != nil {
		log.Debug("game creation failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create game: "+err.Error())
		return
	}
	log.Debug("game created", "game_id", game.ID)

	created, err := db.GetGameByID(r.Context(), &user.ID, game.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load created game: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, created)
}

// GetGameByID godoc
//
//	@Summary		Get game by ID
//	@Description	Returns a single game by its ID
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	obj.Game
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Game not found"
//	@Router			/games/{id} [get]
func GetGameByID(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.MaybeUserFromRequest(r)
	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	log.Debug("getting game by ID", "game_id", gameID, "user_id", userID)

	game, err := db.GetGameByID(r.Context(), userID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, game)
}

// UpdateGame godoc
//
//	@Summary		Update game
//	@Description	Updates a game's properties
//	@Tags			games
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string		true	"Game ID (UUID)"
//	@Param			game	body		obj.Game	true	"Game data"
//	@Success		200		{object}	obj.Game
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	httpx.ErrorResponse	"Game not found"
//	@Security		BearerAuth
//	@Router			/games/{id} [post]
func UpdateGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("updating game", "game_id", gameID, "user_id", user.ID)

	var updatedGame obj.Game
	if err := httpx.ReadJSON(r, &updatedGame); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	updatedGame.ID = gameID

	if err := db.UpdateGame(r.Context(), user.ID, &updatedGame); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update game: "+err.Error())
		return
	}

	game, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, game)
}

// DeleteGame godoc
//
//	@Summary		Delete game
//	@Description	Deletes a game by ID
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{object}	obj.Game
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse	"Failed to delete"
//	@Security		BearerAuth
//	@Router			/games/{id} [delete]
func DeleteGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("deleting game", "game_id", gameID, "user_id", user.ID)

	deleted, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	if err := db.DeleteGame(r.Context(), user.ID, gameID); err != nil {
		log.Debug("game deletion failed", "game_id", gameID, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete game: "+err.Error())
		return
	}
	log.Debug("game deleted", "game_id", gameID)

	httpx.WriteJSON(w, http.StatusOK, deleted)
}

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
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	log.Debug("updating game from YAML", "game_id", gameID, "user_id", user.ID)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Failed to read body: "+err.Error())
		return
	}

	if err := db.UpdateGameYaml(r.Context(), user.ID, gameID, string(body)); err != nil {
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
