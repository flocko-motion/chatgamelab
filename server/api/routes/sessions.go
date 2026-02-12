package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/game/imagecache"
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
		if msg, err := db.GetLatestGameSessionMessage(r.Context(), user.ID, sessionID); err == nil {
			resp.Messages = []obj.GameSessionMessage{*msg}
		} else {
			log.Debug("failed to get latest message", "session_id", sessionID, "error", err)
		}
	case "all":
		if msgs, err := db.GetAllGameSessionMessages(r.Context(), user.ID, sessionID); err == nil {
			resp.Messages = msgs
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
	log.Debug("[TRACE] session loaded from DB for action", "session_id", session.ID, "ai_session", session.AiSession, "platform", session.AiPlatform)

	// Get current status fields from the latest message in the session
	var currentStatus []obj.StatusField
	latestMsg, err := db.GetLatestGameSessionMessage(r.Context(), *userID, sessionID)
	if err == nil && latestMsg != nil {
		currentStatus = latestMsg.StatusFields
	}

	// Create player action message with current status for AI context
	action := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypePlayer,
		Message:       req.Message,
		StatusFields:  currentStatus,
	}

	// Re-resolve API key and execute action with fallback retry logic
	log.Debug("executing session action", "session_id", session.ID, "message_length", len(req.Message))
	response, httpErr := game.DoSessionActionWithFallback(r.Context(), session, action)
	if httpErr != nil {
		log.Warn("session action failed", "session_id", session.ID, "error", httpErr.Message)
		httpx.WriteHTTPError(w, httpErr)
		return
	}
	log.Debug("session action completed", "session_id", session.ID, "response_id", response.ID)

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
	log.Debug("getting game sessions", "game_id", gameID, "user_id", user.ID)

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

	log.Debug("creating game session", "game_id", gameID, "user_id", user.ID)

	session, firstMessage, httpErr := game.CreateSession(r.Context(), user.ID, gameID)
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
	responseMessage.Audio = nil

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

	log.Debug("updating session API key", "session_id", sessionID, "user_id", user.ID)

	session, err := db.ResolveAndUpdateGameSessionApiKey(r.Context(), user.ID, sessionID)
	if err != nil {
		if httpErr, ok := err.(*obj.HTTPError); ok {
			httpx.WriteHTTPError(w, httpErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update session: "+err.Error())
		return
	}

	log.Debug("session updated", "session_id", sessionID)
	httpx.WriteJSON(w, http.StatusOK, session)
}

// MessageStatusResponse is the unified response for polling message completion.
// Frontend polls this to catch up after SSE drops, on reload, or for image progress.
type MessageStatusResponse struct {
	Text         string            `json:"text"`                   // Current full text of the message
	TextDone     bool              `json:"textDone"`               // True when text streaming is complete (Stream=false in DB)
	ImageStatus  string            `json:"imageStatus"`            // "none" | "generating" | "complete" | "error"
	ImageHash    string            `json:"imageHash,omitempty"`    // Hash for cache-busting image URL
	ImageError   string            `json:"imageError,omitempty"`   // Machine-readable image error code
	StatusFields []obj.StatusField `json:"statusFields,omitempty"` // Current status fields
	Error        string            `json:"error,omitempty"`        // Fatal error message (AI failure)
	ErrorCode    string            `json:"errorCode,omitempty"`    // Machine-readable error code
}

// GetMessageStatus godoc
//
//	@Summary		Get message completion status
//	@Description	Returns the current state of a message: text, image status, errors.
//	@Description	Frontend polls this as a safety net when SSE drops, on reload, or for image progress.
//	@Description	No authentication required - message UUIDs are random and unguessable.
//	@Tags			messages
//	@Produce		json
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{object}	MessageStatusResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message not found"
//	@Router			/messages/{id}/status [get]
func GetMessageStatus(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Get message from DB (no auth - relies on UUID unguessability, same as image endpoint)
	msg, err := db.GetGameSessionMessageByIDPublic(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	// Determine text completion: if stream registry has an active stream, text is still in progress
	registry := stream.Get()
	textStreaming := registry.Lookup(messageID) != nil

	resp := MessageStatusResponse{
		Text:         msg.Message,
		TextDone:     !textStreaming,
		StatusFields: msg.StatusFields,
	}

	// Determine image status
	if msg.ImagePrompt == nil || *msg.ImagePrompt == "" {
		resp.ImageStatus = "none"
	} else {
		// Check image cache first (in-progress generation)
		cache := imagecache.Get()
		imgStatus := cache.GetStatus(messageID)

		if imgStatus.Exists {
			if imgStatus.HasError {
				resp.ImageStatus = "error"
				resp.ImageError = imgStatus.ErrorCode
			} else if imgStatus.IsComplete {
				resp.ImageStatus = "complete"
				resp.ImageHash = imgStatus.Hash
			} else {
				resp.ImageStatus = "generating"
				resp.ImageHash = imgStatus.Hash
			}
		} else if len(msg.Image) > 0 {
			// Image already persisted to DB
			resp.ImageStatus = "complete"
			resp.ImageHash = "persisted"
		} else if textStreaming {
			// Stream still active, image generation hasn't started yet
			resp.ImageStatus = "generating"
		} else {
			// Stream finished but no image - generation failed silently or was skipped
			resp.ImageStatus = "none"
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

// ImageStatusResponse is the response for the image status endpoint
type ImageStatusResponse struct {
	Hash                     string `json:"hash"`
	IsComplete               bool   `json:"isComplete"`
	HasError                 bool   `json:"hasError,omitempty"`
	ErrorCode                string `json:"errorCode,omitempty"`
	ErrorMsg                 string `json:"errorMsg,omitempty"`
	Exists                   bool   `json:"exists"`
	IsOrganisationUnverified bool   `json:"isOrganisationUnverified,omitempty"`
}

// GetMessageImageStatus godoc
//
//	@Summary		Get image generation status
//	@Description	Returns the current hash and completion status of an image being generated.
//	@Description	Frontend can poll this to detect when new partial/final images are available.
//	@Tags			messages
//	@Produce		json
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{object}	ImageStatusResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Router			/messages/{id}/image/status [get]
func GetMessageImageStatus(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Check cache first for in-progress images
	cache := imagecache.Get()
	status := cache.GetStatus(messageID)

	if status.Exists {
		resp := ImageStatusResponse{
			Hash:       status.Hash,
			IsComplete: status.IsComplete,
			HasError:   status.HasError,
			ErrorCode:  status.ErrorCode,
			ErrorMsg:   status.ErrorMsg,
			Exists:     true,
		}

		// If there's an org verification error, also set the flag
		if status.ErrorCode == obj.ErrCodeOrgVerificationRequired {
			resp.IsOrganisationUnverified = true
		}

		httpx.WriteJSON(w, http.StatusOK, resp)
		return
	}

	// Check if image exists in DB (already completed)
	msg, err := db.GetGameSessionMessageImageByID(r.Context(), messageID)
	if err == nil && len(msg.Image) > 0 {
		httpx.WriteJSON(w, http.StatusOK, ImageStatusResponse{
			Hash:       "persisted",
			IsComplete: true,
			Exists:     true,
		})
		return
	}

	// No image in cache or DB
	httpx.WriteJSON(w, http.StatusOK, ImageStatusResponse{
		Exists: false,
	})
}

// GetMessageImage godoc
//
//	@Summary		Get message image
//	@Description	Returns the generated image for a game session message.
//	@Description	Checks in-memory cache first (for partial/WIP images), then database.
//	@Description	No authentication required - message UUIDs are random and unguessable.
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

	// Check cache first for in-progress/partial images
	cache := imagecache.Get()
	if imageData, exists := cache.GetImage(messageID); exists {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache") // Don't cache partial images
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
		return
	}

	// Fall back to database for persisted images
	msg, err := db.GetGameSessionMessageImageByID(r.Context(), messageID)
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

// GetMessageAudio godoc
//
//	@Summary		Get message audio
//	@Description	Returns the audio narration for a message (MP3 format).
//	@Description	No authentication required - message UUIDs are random and unguessable.
//	@Tags			messages
//	@Produce		audio/mpeg
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{file}		binary
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message or audio not found"
//	@Router			/messages/{id}/audio [get]
func GetMessageAudio(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	audio, err := db.GetGameSessionMessageAudioByID(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	if len(audio) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "Audio not found")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.WriteHeader(http.StatusOK)
	w.Write(audio)
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
	// Use origin from request for CORS with credentials support
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream chunks to client until text, image, and audio are all done
	log.Debug("[SSE] client connected", "message_id", messageID)
	textDone := false
	imageDone := false
	audioDone := false
	chunkCount := 0

	for chunk := range s.Chunks {
		chunkCount++
		data, _ := json.Marshal(chunk)
		log.Debug("[SSE] sending chunk to client", "message_id", messageID, "chunk_num", chunkCount, "text_len", len(chunk.Text), "textDone", chunk.TextDone, "imageDone", chunk.ImageDone, "audioDone", chunk.AudioDone)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if chunk.TextDone {
			textDone = true
		}
		if chunk.ImageDone {
			imageDone = true
		}
		if chunk.AudioDone {
			audioDone = true
		}
		if chunk.Error != "" {
			break
		}
		// Stream is complete when all active channels are done
		if textDone && imageDone && audioDone {
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
