package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"

	"github.com/google/uuid"
)

type GameNewRequest struct {
	Name string `json:"name"`
}

type GameNewResponse struct {
	ID uuid.UUID `json:"id"`
}

var GamesNew = handler.NewEndpoint(
	"/api/games/new",
	false,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		var req GameNewRequest
		if err := json.NewDecoder(request.R.Body).Decode(&req); err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}

		newGame := obj.Game{
			Name:         req.Name,
			StatusFields: `[{"name":"Gold","value":"100"}]`,
		}
		if err := db.CreateGame(request.Ctx, request.User.ID, &newGame); err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create game: " + err.Error()}
		}

		return GameNewResponse{ID: newGame.ID}, nil
	},
)
