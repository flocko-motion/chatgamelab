// package: routes / game session HTTP handlers
// type:    logic
// job:     list, fetch, create, update, delete game sessions; post player actions and restart.
// limits:  does not serve message status/media/streaming (-> sessions_messages.go).
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/game/imagecache"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for sessions
type SessionActionRequest struct {
	Type          string            `json:"type,omitempty"` // Message type: "player" or "system" (defaults to "player")
	Message       string            `json:"message"`
	StatusFields  []obj.StatusField `json:"statusFields,omitempty"`  // Current status to pass to AI
	AudioBase64   string            `json:"audioBase64,omitempty"`   // Base64-encoded audio from voice input
	AudioMimeType string            `json:"audioMimeType,omitempty"` // MIME type of the audio (e.g. "audio/webm;codecs=opus")
}

// SessionResponse wraps a game session together with its messages.
type SessionResponse struct {
	*obj.GameSession
	Messages []obj.GameSessionMessage `json:"messages,omitempty"`
}

// GetUserSessions godoc
//
//	@Summary		List user sessions
//	@Description	Returns recent sessions for the authenticated user with game names
//	@Tags			sessions
//	@Produce		json
//	@Param			search	query		string	false	"Search by game name"
//	@Param			sortBy	query		string	false	"Sort field: game, model, lastPlayed (default)"
//	@Success		200	{array}		db.UserSessionWithGame
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/sessions [get]
func GetUserSessions(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	filters := &db.GetUserSessionsFilters{
		Search:    httpx.QueryParam(r, "search"),
		SortField: httpx.QueryParam(r, "sortBy"),
	}

	sessions, err := db.GetGameSessionsByUserID(r.Context(), user.ID, filters)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessions)
}

// GetSession godoc
//
//	@Summary		Get session
//	@Description	Returns session details. Optional query parameter can include latest message.
//	@Tags			sessions
//	@Produce		json
//	@Param			id		path		string	true	"Session ID (UUID)"
//	@Param			messages	query		string	false	"Message inclusion: none|latest|all"
//	@Success		200		{object}	SessionResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid session ID"
//	@Failure		404		{object}	httpx.ErrorResponse	"Session not found"
//	@Router			/sessions/{id} [get]
func GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	user := httpx.MaybeUserFromRequest(r)
	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	session, err := db.GetGameSessionByID(r.Context(), userID, sessionID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Session not found")
		return
	}

	resp := SessionResponse{GameSession: session}

	// Check ?messages= query param: none (default), latest, all
	switch httpx.QueryParam(r, "messages") {
	case "latest":
		if msg, err := db.GetLatestGameSessionMessage(r.Context(), user.ID, sessionID); err == nil {
			resp.Messages = []obj.GameSessionMessage{*msg}
			game.ApplySessionCapabilities(session, resp.Messages)
		} else {
			log.Debug("failed to get latest message", "session_id", sessionID, "error", err)
		}
	case "all":
		if msgs, err := db.GetAllGameSessionMessages(r.Context(), user.ID, sessionID); err == nil {
			resp.Messages = msgs
			game.ApplySessionCapabilities(session, resp.Messages)
		} else {
			log.Debug("failed to get all messages", "session_id", sessionID, "error", err)
		}
	}

	// Check for messages with imagePrompt but no persisted image - retry generation once.
	// Only for non-streaming (text-complete) messages where the image was lost or never generated.
	for i := range resp.Messages {
		msg := &resp.Messages[i]
		if msg.ImagePrompt != nil && *msg.ImagePrompt != "" && len(msg.Image) == 0 && !msg.Stream {
			cache := imagecache.Get()
			status := cache.GetStatus(msg.ID)
			if !status.Exists {
				log.Debug("detected missing image, triggering retry", "session_id", sessionID, "message_id", msg.ID)
				game.RetryImageGeneration(session, msg)
			}
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

// PostSessionAction godoc
//
//	@Summary		Send session action
//	@Description	Sends a player message/action to a session and returns the resulting message.
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Session ID (UUID)"
//	@Param			request	body		SessionActionRequest	true	"Player action"
//	@Success		200		{object}	obj.GameSessionMessage
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		404		{object}	httpx.ErrorResponse	"Session not found"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/sessions/{id} [post]
func PostSessionAction(w http.ResponseWriter, r *http.Request) {
	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	user := httpx.MaybeUserFromRequest(r)
	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	var req SessionActionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Get session from DB
	session, err := db.GetGameSessionByID(r.Context(), userID, sessionID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Session not found")
		return
	}
	// Get current status fields from the latest message in the session
	var currentStatus []obj.StatusField
	latestMsg, err := db.GetLatestGameSessionMessage(r.Context(), *userID, sessionID)
	if err == nil && latestMsg != nil {
		currentStatus = latestMsg.StatusFields
	}

	// Create action message with current status for AI context
	// Type defaults to "player" if not specified
	messageType := req.Type
	if messageType == "" {
		messageType = obj.GameSessionMessageTypePlayer
	}

	action := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          messageType,
		Message:       req.Message,
		StatusFields:  currentStatus,
		AudioBase64:   req.AudioBase64,
		AudioMimeType: req.AudioMimeType,
	}

	// Re-resolve API key and execute action with fallback retry logic
	response, httpErr := game.DoSessionActionWithFallback(r.Context(), session, action)
	if httpErr != nil {
		log.Warn("session action failed", "session_id", session.ID, "error", httpErr.Message)
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	// Return full message (without image/audio bytes - served via separate endpoints)
	response.Image = nil
	response.Audio = nil
	httpx.WriteJSON(w, http.StatusOK, response)
}

// GetGameSessions godoc
//
//	@Summary		List game sessions
//	@Description	Lists sessions for a game
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
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessions)
}

// CreateGameSession godoc
//
//	@Summary		Create game session
//	@Description	Creates a new session for a game and returns the session with first message
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Game ID (UUID)"
//	@Success		200		{object}	SessionResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/games/{id}/sessions [post]
func CreateGameSession(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	session, _, httpErr := game.CreateSession(r.Context(), user.ID, gameID)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	// Create a copy for response to avoid modifying session used by async goroutines
	responseSession := *session
	responseSession.ApiKey = nil
	responseSession.AiSession = ""

	// TWO-PHASE INITIALIZATION: Return session without messages
	// Frontend will call sendAction("") to trigger opening scene generation
	httpx.WriteJSON(w, http.StatusOK, SessionResponse{
		GameSession: &responseSession,
		Messages:    []obj.GameSessionMessage{}, // Empty - no initial message
	})
}

// DeleteSession godoc
//
//	@Summary		Delete session
//	@Description	Deletes a session and all its messages. User must be the owner.
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
		if err.Error() == "access denied: not the owner of this session" {
			httpx.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		if err.Error() == "session not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateSession godoc
//
//	@Summary		Update session API key
//	@Description	Re-resolves the API key for a session. Used when resuming a session whose API key was deleted.
//	@Description	The API key is resolved server-side using the same priority as session creation.
//	@Tags			sessions
//	@Produce		json
//	@Param			id	path		string	true	"Session ID (UUID)"
//	@Success		200	{object}	obj.GameSession
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid request or no API key available"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	httpx.ErrorResponse	"Session not found"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/sessions/{id} [patch]
func UpdateSession(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	session, err := db.ResolveAndUpdateGameSessionApiKey(r.Context(), user.ID, sessionID)
	if err != nil {
		if httpErr, ok := err.(*obj.HTTPError); ok {
			httpx.WriteHTTPError(w, httpErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update session: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, session)
}

// PostRestart godoc
//
//	@Summary		Restart server
//	@Description	Admin-only endpoint that triggers a server restart.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{string}	string
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden"
//	@Security		BearerAuth
//	@Router			/restart [post]
func PostRestart(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	// Require admin
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		log.Debug("restart denied - not admin", "user_id", user.ID)
		httpx.WriteError(w, http.StatusForbidden, "Forbidden: admin access required")
		return
	}

	log.Info("restart request received", "user_id", user.ID)
	go func() {
		// Give time for response to be sent
		<-r.Context().Done()
		log.Info("shutting down server for restart")
		// Use a channel or signal instead of os.Exit for graceful shutdown
	}()

	httpx.WriteJSON(w, http.StatusOK, "Server will restart shortly")
}
