package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/game/stream"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for sessions
type SessionActionRequest struct {
	Message string `json:"message"`
}

type SessionResponse struct {
	*obj.GameSession
	Messages []obj.GameSessionMessage `json:"messages,omitempty"`
}

type CreateSessionRequest struct {
	ShareID uuid.UUID `json:"shareId"`
	Model   string    `json:"model"`
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

	log.Printf("GetSession: %s", sessionID)

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
			log.Printf("GetLatestGameSessionMessage error: %v", err)
		}
	case "all":
		// TODO: implement GetAllGameSessionMessages
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

	log.Printf("PostSessionAction: %s", sessionID)

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

	// Create player action message
	action := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypePlayer,
		Message:       req.Message,
	}

	// Execute the action and get streaming response
	response, httpErr := game.DoSessionAction(r.Context(), session, action)
	if httpErr != nil {
		httpx.WriteError(w, httpErr.StatusCode, httpErr.Message)
		return
	}

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

	log.Printf("GetGameSessions: %s", gameID)

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
//	@Description	Creates a new session for a game and returns the first message (image bytes omitted)
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Game ID (UUID)"
//	@Param			request	body		CreateSessionRequest	true	"Create session request"
//	@Success		200		{object}	obj.GameSessionMessage
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

	log.Printf("CreateGameSession: %s", gameID)

	var req CreateSessionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	_, firstMessage, httpErr := game.CreateSession(r.Context(), user.ID, gameID, req.ShareID, req.Model)
	if httpErr != nil {
		httpx.WriteError(w, httpErr.StatusCode, httpErr.Message)
		return
	}

	firstMessage.Image = nil
	httpx.WriteJSON(w, http.StatusOK, firstMessage)
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
		httpx.WriteError(w, http.StatusForbidden, "Forbidden: admin access required")
		return
	}

	log.Println("restart request - exiting server")
	go func() {
		// Give time for response to be sent
		<-r.Context().Done()
		log.Println("shutting down server")
		// Use a channel or signal instead of os.Exit for graceful shutdown
	}()

	httpx.WriteJSON(w, http.StatusOK, "Server will restart shortly")
}
