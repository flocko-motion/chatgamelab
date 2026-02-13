package routes

import (
	"net/http"
	"strings"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

type CreateGameRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Public      *bool  `json:"public,omitempty"`
}

// GetGames godoc
//
//	@Summary		List games
//	@Description	Returns a list of games. If authenticated, includes user's private games.
//	@Tags			games
//	@Produce		json
//	@Param			search	query		string	false	"Search games by name (case-insensitive)"
//	@Param			sortBy	query		string	false	"Sort field (name, createdAt, modifiedAt)"
//	@Param			sortDir	query		string	false	"Sort direction (asc, desc)"
//	@Param			filter	query		string	false	"Filter type (all, own, public, organization, favorites)"
//	@Success		200		{array}		obj.Game
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/games [get]
func GetGames(w http.ResponseWriter, r *http.Request) {
	user := httpx.MaybeUserFromRequest(r)

	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	searchQuery := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sortBy")
	sortDir := r.URL.Query().Get("sortDir")
	filter := r.URL.Query().Get("filter")

	filters := &db.GetGamesFilters{
		Search:    searchQuery,
		SortField: sortBy,
		SortDir:   sortDir,
		Filter:    filter,
	}
	games, err := db.GetGames(r.Context(), userID, filters)
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

	// Parse request
	var req CreateGameRequest
	if err := httpx.ReadJSONOrYAML(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate that at least name is provided
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Missing required field: name")
		return
	}

	// Validate name length
	if len(strings.TrimSpace(req.Name)) > 70 {
		httpx.WriteAppError(w, obj.ErrNameTooLong("Game name must be 70 characters or less"))
		return
	}

	// Build game object from request
	game := obj.Game{
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Public != nil {
		game.Public = *req.Public
	}

	if err := db.CreateGame(r.Context(), user.ID, &game); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create game: "+err.Error())
		return
	}
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

	var updatedGame obj.Game
	if err := httpx.ReadJSON(r, &updatedGame); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	updatedGame.ID = gameID

	// Validate name length
	if len(strings.TrimSpace(updatedGame.Name)) > 70 {
		httpx.WriteAppError(w, obj.ErrNameTooLong("Game name must be 70 characters or less"))
		return
	}

	if err := db.UpdateGame(r.Context(), user.ID, &updatedGame); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
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

	deleted, err := db.GetGameByID(r.Context(), &user.ID, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	if err := db.DeleteGame(r.Context(), user.ID, gameID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete game: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, deleted)
}

// CloneGame godoc
//
//	@Summary		Clone game
//	@Description	Creates a copy of a game for the authenticated user
//	@Tags			games
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID) to clone"
//	@Success		200	{object}	obj.Game
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	httpx.ErrorResponse	"Game not found"
//	@Failure		500	{object}	httpx.ErrorResponse	"Failed to clone"
//	@Security		BearerAuth
//	@Router			/games/{id}/clone [post]
func CloneGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)

	// Get the source game (allow cloning public games or own games)
	sourceGame, err := db.GetGameByID(r.Context(), nil, gameID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	// Check if user can clone: must be public or owned by user
	isOwner := sourceGame.Meta.CreatedBy.Valid && sourceGame.Meta.CreatedBy.UUID == user.ID
	if !sourceGame.Public && !isOwner {
		httpx.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	// Determine the original creator: use existing OriginallyCreatedBy if set, otherwise use the source game's creator
	var originalCreator *uuid.UUID
	if sourceGame.OriginallyCreatedBy != nil {
		originalCreator = sourceGame.OriginallyCreatedBy
	} else if sourceGame.Meta.CreatedBy.Valid {
		originalCreator = &sourceGame.Meta.CreatedBy.UUID
	}

	// Create a new game based on the source
	clonedGame := obj.Game{
		Name:                   sourceGame.Name + " (Copy)",
		Description:            sourceGame.Description,
		Public:                 false, // Cloned games start as private
		ImageStyle:             sourceGame.ImageStyle,
		SystemMessageScenario:  sourceGame.SystemMessageScenario,
		SystemMessageGameStart: sourceGame.SystemMessageGameStart,
		StatusFields:           sourceGame.StatusFields,
		Theme:                  sourceGame.Theme,
		FirstMessage:           sourceGame.FirstMessage,
		FirstStatus:            sourceGame.FirstStatus,
		CSS:                    sourceGame.CSS,
		OriginallyCreatedBy:    originalCreator,
	}

	if err := db.CreateGame(r.Context(), user.ID, &clonedGame); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to clone game: "+err.Error())
		return
	}
	// Increment the clone count on the source game
	if err := db.IncrementGameCloneCount(r.Context(), gameID); err != nil {
		log.Warn("failed to increment clone count", "error", err)
	}

	created, err := db.GetGameByID(r.Context(), &user.ID, clonedGame.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load cloned game: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, created)
}
