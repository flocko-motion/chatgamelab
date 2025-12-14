package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/game"
	"cgl/obj"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// POST /api/sessions/{id} - send action
type SessionActionRequest struct {
	Message string `json:"message"`
}

// GET /api/sessions/{id}?messages=none|latest|all
type SessionResponse struct {
	*obj.GameSession
	Messages []obj.GameSessionMessage `json:"messages,omitempty"`
}

var Session = handler.NewEndpoint(
	"/api/sessions/{id:uuid}",
	handler.AuthOptional,
	"application/json",
	func(request handler.Request) (out interface{}, httpErr *obj.HTTPError) {
		return handleSessionRequest(request, false)
	},
)

func handleSessionRequest(request handler.Request, public bool) (out interface{}, httpErr *obj.HTTPError) {
	sessionID, err := request.GetPathParamUUID("id")
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: "Invalid session ID"}
	}
	log.Printf("Session: %s %s", request.R.Method, sessionID)

	var userID *uuid.UUID
	if request.User != nil {
		userID = &request.User.ID
	}

	switch request.R.Method {
	case http.MethodPost:
		var req SessionActionRequest
		if httpErr := request.BodyJSON(&req); httpErr != nil {
			return nil, httpErr
		}

		// Get session from DB
		session, err := db.GetGameSessionByID(request.Ctx, userID, sessionID)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "Session not found"}
		}

		// Create player action message
		action := obj.GameSessionMessage{
			GameSessionID: session.ID,
			Type:          obj.GameSessionMessageTypePlayer,
			Message:       req.Message,
		}

		// Execute the action and get streaming response
		response, httpErr := game.DoSessionAction(request.Ctx, session, action)
		if httpErr != nil {
			return nil, httpErr
		}

		// Return full message (without image bytes)
		response.Image = nil
		return response, nil

	case http.MethodGet:
		// Get session info
		session, err := db.GetGameSessionByID(request.Ctx, userID, sessionID)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "Session not found"}
		}

		resp := SessionResponse{GameSession: session}

		// Check ?messages= query param: none (default), latest, all
		switch request.R.URL.Query().Get("messages") {
		case "latest":
			if msg, err := db.GetLatestGameSessionMessage(request.Ctx, sessionID); err == nil {
				resp.Messages = []obj.GameSessionMessage{*msg}
			} else {
				log.Printf("GetLatestGameSessionMessage error: %v", err)
			}
		case "all":
			// TODO: implement GetAllGameSessionMessages
		}

		return resp, nil

	default:
		return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
	}
}
