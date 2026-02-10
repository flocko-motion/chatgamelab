package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game"
	"cgl/log"
	"cgl/obj"
)

// ── Guest Play Endpoints ────────────────────────────────────────────────────
// These endpoints allow anonymous users to play a game via a private share token.
// No authentication required — the token in the URL is the capability.

// GuestGameInfo is the public game info returned for the welcome screen.
type GuestGameInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Remaining   *int   `json:"remaining"` // null = unlimited, 0 = exhausted
}

// PlayGuestGetGameInfo godoc
//
//	@Summary		Get game info via share token
//	@Description	Returns basic game info (name, description) for the welcome screen. No session is created.
//	@Tags			play
//	@Produce		json
//	@Param			token	path		string	true	"Private share token"
//	@Success		200		{object}	GuestGameInfo
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid token"
//	@Failure		404		{object}	httpx.ErrorResponse	"Game not found"
//	@Router			/play/{token}/info [get]
func PlayGuestGetGameInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Missing share token")
		return
	}

	gameObj, httpErr := game.ValidatePrivateShareToken(r.Context(), token)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, GuestGameInfo{
		Name:        gameObj.Name,
		Description: gameObj.Description,
		Remaining:   gameObj.PrivateShareRemaining,
	})
}

// PlayGuestCreateSession godoc
//
//	@Summary		Create guest session via share token
//	@Description	Creates a new anonymous game session using a private share link.
//	@Description	No authentication required — the share token grants access.
//	@Tags			play
//	@Produce		json
//	@Param			token	path		string	true	"Private share token"
//	@Success		200		{object}	GuestSessionResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid or expired token"
//	@Failure		403		{object}	httpx.ErrorResponse	"Share limit reached"
//	@Failure		404		{object}	httpx.ErrorResponse	"Game not found"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/play/{token} [post]
func PlayGuestCreateSession(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Missing share token")
		return
	}

	// Parse optional request body for language preference
	var req struct {
		Language string `json:"language"`
	}
	// Body is optional — ignore parse errors (empty body is fine)
	_ = httpx.ReadJSON(r, &req)

	log.Debug("guest play: creating session", "token_prefix", token[:min(8, len(token))], "language", req.Language)

	session, firstMessage, httpErr := game.CreateGuestSession(r.Context(), token, req.Language)
	if httpErr != nil {
		log.Debug("guest play: session creation failed", "error", httpErr.Message)
		httpx.WriteHTTPError(w, httpErr)
		return
	}
	log.Debug("guest play: session created", "session_id", session.ID, "message_id", firstMessage.ID)

	// Strip sensitive fields from response
	responseSession := *session
	responseSession.ApiKey = nil
	responseSession.AiSession = ""

	responseMessage := *firstMessage
	responseMessage.Image = nil

	httpx.WriteJSON(w, http.StatusOK, GuestSessionResponse{
		GameSession: &responseSession,
		Messages:    []SessionMessageResponse{toSessionMessageResponse(responseMessage)},
	})
}

// PlayGuestSendAction godoc
//
//	@Summary		Send guest action via share token
//	@Description	Sends a player message to an anonymous session. Validates token→game→session chain.
//	@Tags			play
//	@Accept			json
//	@Produce		json
//	@Param			token	path		string				true	"Private share token"
//	@Param			id		path		string				true	"Session ID (UUID)"
//	@Param			request	body		SessionActionRequest	true	"Player action"
//	@Success		200		{object}	obj.GameSessionMessage
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		403		{object}	httpx.ErrorResponse	"Token/session mismatch"
//	@Failure		404		{object}	httpx.ErrorResponse	"Session not found"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/play/{token}/sessions/{id} [post]
func PlayGuestSendAction(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil || token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate token → game, then verify session belongs to that game
	gameObj, httpErr := game.ValidatePrivateShareToken(r.Context(), token)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	session, err := db.GetGameSessionByIDForGuest(r.Context(), sessionID, gameObj.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Session not found")
		return
	}

	var req SessionActionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Get current status from latest message
	var currentStatus []obj.StatusField
	latestMsg, err := db.GetLatestGuestSessionMessage(r.Context(), sessionID)
	if err == nil && latestMsg != nil {
		currentStatus = latestMsg.StatusFields
	}

	action := obj.GameSessionMessage{
		GameSessionID: session.ID,
		Type:          obj.GameSessionMessageTypePlayer,
		Message:       req.Message,
		StatusFields:  currentStatus,
	}

	// Re-resolve API key from the private share
	if httpErr := game.ResolveGuestSessionApiKey(r.Context(), session, gameObj); httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	response, httpErr := game.DoSessionAction(r.Context(), session, action)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	response.Image = nil
	httpx.WriteJSON(w, http.StatusOK, response)
}

// PlayGuestGetSession godoc
//
//	@Summary		Get guest session via share token
//	@Description	Returns session details for recovery after reload. Validates token→game→session chain.
//	@Tags			play
//	@Produce		json
//	@Param			token	path		string	true	"Private share token"
//	@Param			id		path		string	true	"Session ID (UUID)"
//	@Param			messages	query	string	false	"Message inclusion: none|latest|all"
//	@Success		200		{object}	SessionResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Router			/play/{token}/sessions/{id} [get]
func PlayGuestGetSession(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	sessionID, err := httpx.PathParamUUID(r, "id")
	if err != nil || token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate token → game
	gameObj, httpErr := game.ValidatePrivateShareToken(r.Context(), token)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	// Load session and verify it belongs to this game
	session, err := db.GetGameSessionByIDForGuest(r.Context(), sessionID, gameObj.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Session not found")
		return
	}

	resp := SessionResponse{GameSession: session}

	switch httpx.QueryParam(r, "messages") {
	case "latest":
		if msg, err := db.GetLatestGuestSessionMessage(r.Context(), sessionID); err == nil {
			resp.Messages = []obj.GameSessionMessage{*msg}
		}
	case "all":
		if msgs, err := db.GetAllGuestSessionMessages(r.Context(), sessionID); err == nil {
			resp.Messages = msgs
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

// ── Response types ──────────────────────────────────────────────────────────

// SessionMessageResponse is a message without the image bytes (sent as URL).
type SessionMessageResponse struct {
	obj.GameSessionMessage
}

func toSessionMessageResponse(msg obj.GameSessionMessage) SessionMessageResponse {
	msg.Image = nil
	return SessionMessageResponse{msg}
}

// GuestSessionResponse is the response for guest session creation.
type GuestSessionResponse struct {
	*obj.GameSession
	Messages []SessionMessageResponse `json:"messages,omitempty"`
}
