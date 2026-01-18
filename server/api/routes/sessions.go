package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/game/stream"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for sessions
type SessionActionRequest struct {
	Message      string            `json:"message"`
	StatusFields []obj.StatusField `json:"statusFields,omitempty"` // Current status to pass to AI
}

type SessionResponse struct {
	*obj.GameSession
	Messages []obj.GameSessionMessage `json:"messages,omitempty"`
}

type CreateSessionRequest struct {
	ShareID uuid.UUID `json:"shareId"`
	Model   string    `json:"model"`
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

	log.Debug("getting user sessions", "user_id", user.ID, "search", filters.Search, "sortBy", filters.SortField)

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

	log.Debug("getting session", "session_id", sessionID, "user_id", userID)

	session, err := db.GetGameSessionByID(r.Context(), userID, sessionID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Session not found")
		return
	}

	resp := SessionResponse{GameSession: session}

	// Check ?messages= query param: none (default), latest, all
	switch httpx.QueryParam(r, "messages") {
	case "latest":
		if msg, err := db.GetLatestGameSessionMessage(r.Context(), sessionID); err == nil {
			resp.Messages = []obj.GameSessionMessage{*msg}
		} else {
			log.Debug("failed to get latest message", "session_id", sessionID, "error", err)
		}
	case "all":
		if msgs, err := db.GetAllGameSessionMessages(r.Context(), sessionID); err == nil {
			resp.Messages = msgs
		} else {
			log.Debug("failed to get all messages", "session_id", sessionID, "error", err)
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

	log.Debug("session action request", "session_id", sessionID, "user_id", userID)

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

	// Create player action message with current status for AI context
	action := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypePlayer,
		Message:       req.Message,
		StatusFields:  req.StatusFields, // Pass current status to AI
	}

	// Execute the action and get streaming response
	log.Debug("executing session action", "session_id", session.ID, "message_length", len(req.Message))
	response, httpErr := game.DoSessionAction(r.Context(), session, action)
	if httpErr != nil {
		log.Debug("session action failed", "session_id", session.ID, "error", httpErr.Message)
		httpx.WriteHTTPError(w, httpErr)
		return
	}
	log.Debug("session action completed", "session_id", session.ID, "response_id", response.ID)

	// Return full message (without image bytes)
	response.Image = nil
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

	log.Debug("getting game sessions", "game_id", gameID)

	// TODO: we need to consider user permissions here!
	sessions, err := db.GetGameSessionsByGameID(r.Context(), gameID)
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
//	@Param			request	body		CreateSessionRequest	true	"Create session request"
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

	log.Debug("creating game session", "game_id", gameID, "user_id", user.ID)

	var req CreateSessionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	log.Debug("creating session with model", "game_id", gameID, "share_id", req.ShareID, "model", req.Model)
	session, firstMessage, httpErr := game.CreateSession(r.Context(), user.ID, gameID, req.ShareID, req.Model)
	if httpErr != nil {
		log.Debug("session creation failed", "game_id", gameID, "error", httpErr.Message)
		httpx.WriteHTTPError(w, httpErr)
		return
	}
	log.Debug("session created", "game_id", gameID, "session_id", session.ID, "message_id", firstMessage.ID)

	// Create a copy for response to avoid modifying session used by async goroutines
	responseSession := *session
	responseSession.ApiKey = nil
	responseSession.AiSession = ""

	responseMessage := *firstMessage
	responseMessage.Image = nil

	httpx.WriteJSON(w, http.StatusOK, SessionResponse{
		GameSession: &responseSession,
		Messages:    []obj.GameSessionMessage{responseMessage},
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

	log.Debug("deleting session", "session_id", sessionID, "user_id", user.ID)

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

	log.Debug("session deleted", "session_id", sessionID)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetMessageImage godoc
//
//	@Summary		Get message image
//	@Description	Returns the image for a message as PNG
//	@Tags			messages
//	@Produce		image/png
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{file}		binary	"PNG image"
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message or image not found"
//	@Router			/messages/{id}/image [get]
func GetMessageImage(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	msg, err := db.GetGameSessionMessageByID(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	if len(msg.Image) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "Image not found")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	w.Write(msg.Image)
}

// GetMessageStream godoc
//
//	@Summary		Stream message updates (SSE)
//	@Description	Server-Sent Events endpoint for streaming message chunks.
//	@Tags			messages
//	@Produce		text/event-stream
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{string}	string	"SSE stream"
//	@Failure		400	{string}	string	"Invalid message ID"
//	@Failure		404	{string}	string	"Stream not found"
//	@Router			/messages/{id}/stream [get]
func GetMessageStream(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// Lookup the stream
	registry := stream.Get()
	s := registry.Lookup(messageID)
	if s == nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream chunks to client until both text and image are done
	textDone := false
	imageDone := false

	for chunk := range s.Chunks {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if chunk.TextDone {
			textDone = true
		}
		if chunk.ImageDone {
			imageDone = true
		}
		if chunk.Error != "" {
			break
		}
		if textDone && imageDone {
			break
		}
	}

	// Cleanup
	registry.Remove(messageID)
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
