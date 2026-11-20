// package: routes / game session HTTP handlers
// type:    logic
// job:     lists and deletes a player's sessions, and issues the play token the playback-engine acts with
// limits:  playing a session is the engine's, mounted in engine.go; storage is db/game_sessions.go
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
)

// GetUserSessions godoc
//
//	@Summary		List user sessions
//	@Description	Returns the authenticated player's recent sessions with their games' names
//	@Tags			sessions
//	@Produce		json
//	@Param			search	query		string	false	"Search by game name"
//	@Param			sortBy	query		string	false	"Sort field: game, model (AI platform), lastPlayed (default)"
//	@Success		200		{array}		db.UserSessionWithGame
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/sessions [get]
func GetUserSessions(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	sessions, err := db.GetGameSessionsByUserID(r.Context(), user.ID, db.GetUserSessionsFilters{
		Search:    httpx.QueryParam(r, "search"),
		SortField: httpx.QueryParam(r, "sortBy"),
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessions)
}

// GetGameSessions godoc
//
//	@Summary		List game sessions
//	@Description	Lists the authenticated player's own sessions of a game
//	@Tags			sessions
//	@Produce		json
//	@Param			id	path		string	true	"Game ID (UUID)"
//	@Success		200	{array}		obj.GameSession
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid game ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/sessions [get]
func GetGameSessions(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	user := httpx.UserFromRequest(r)
	sessions, err := db.GetGameSessionsByGameID(r.Context(), user.ID, gameID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessions)
}

// DeleteSession godoc
//
//	@Summary		Delete session
//	@Description	Deletes a session and everything it stored. The player must own it.
//	@Tags			sessions
//	@Produce		json
//	@Param			id	path		string	true	"Session ID (UUID)"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid session ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	httpx.ErrorResponse	"Session not found"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/sessions/{id} [delete]
func DeleteSession(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	if err := db.DeleteGameSession(r.Context(), user.ID, sessionID); err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}
	Engine.Forget(sessionID.String())

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
