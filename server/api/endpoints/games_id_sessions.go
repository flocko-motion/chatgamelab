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

type CreateSessionRequest struct {
	ShareID uuid.UUID `json:"shareId"`
	Model   string    `json:"model"`
}

type CreateSessionResponse struct {
	SessionID uuid.UUID `json:"sessionId"`
}

var GamesIdSessions = handler.NewEndpoint(
	"/api/games/{id:uuid}/sessions",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		gameID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid game ID"}
		}
		log.Printf("GamesIdSessions: %s %s", request.R.Method, gameID)

		switch request.R.Method {
		case "POST":
			var req CreateSessionRequest
			if httpErr := request.BodyJSON(&req); httpErr != nil {
				return nil, httpErr
			}

			session, httpErr := game.CreateSession(request.Ctx, request.User.ID, gameID, req.ShareID, req.Model)
			if httpErr != nil {
				return nil, httpErr
			}

			return CreateSessionResponse{SessionID: session.ID}, nil

		case "GET":
			// TODO: we need to consider user permissions here!
			sessions, err := db.GetGameSessionsByGameID(request.Ctx, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get sessions: " + err.Error()}
			}

			return sessions, nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
