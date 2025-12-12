package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"strings"

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
		contentType := request.R.Header.Get("Content-Type")

		// Create a minimal game first
		newGame := obj.Game{
			Name:         "New Game",
			StatusFields: `[{"name":"Gold","value":"100"}]`,
		}

		if err := db.CreateGame(request.Ctx, request.User.ID, &newGame); err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create game: " + err.Error()}
		}

		if strings.Contains(contentType, "application/x-yaml") {
			// Update with YAML content from "import game" form
			body, httpErr := request.BodyRaw()
			if httpErr != nil {
				return nil, httpErr
			}

			if err := db.UpdateGameYaml(request.Ctx, request.User.ID, newGame.ID, body); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Failed to apply YAML: " + err.Error()}
			}
		} else {
			// Update with JSON content from "new game" form
			var req GameNewRequest
			if httpErr := request.BodyJSON(&req); httpErr != nil {
				return nil, httpErr
			}

			if req.Name != "" {
				newGame.Name = req.Name
				if err := db.UpdateGame(request.Ctx, request.User.ID, &newGame); err != nil {
					return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update game: " + err.Error()}
				}
			}
		}

		return GameNewResponse{ID: newGame.ID}, nil
	},
)
