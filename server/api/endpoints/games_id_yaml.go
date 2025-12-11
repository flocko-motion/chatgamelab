package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/functional"
	"cgl/obj"
	"log"
	"net/http"
)

var GamesIdYaml = handler.NewEndpoint(
	"/api/games/{id:uuid}/yaml",
	false,
	"application/x-yaml",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		gameID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid game ID"}
		}
		log.Printf("GamesIdYaml: %s %s", request.R.Method, gameID)

		switch request.R.Method {
		case "GET":
			game, err := db.GetGameByID(request.Ctx, &request.User.ID, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "Game not found"}
			}

			return game, nil

		case "PUT":
			// Get existing game first
			existing, err := db.GetGameByID(request.Ctx, &request.User.ID, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "Game not found"}
			}

			// Unmarshal YAML into a separate object
			var incoming obj.Game
			if httpErr := request.BodyYAML(&incoming); httpErr != nil {
				return nil, httpErr
			}

			// Selectively copy allowed fields
			existing.Name = incoming.Name
			existing.Description = incoming.Description
			existing.SystemMessageScenario = incoming.SystemMessageScenario
			existing.SystemMessageGameStart = incoming.SystemMessageGameStart
			existing.ImageStyle = incoming.ImageStyle

			// Normalize status fields JSON
			existing.StatusFields = functional.NormalizeJson(incoming.StatusFields, &[]obj.StatusField{})

			// TODO: this is too primitive - we need to validate the contents of the fields as well
			existing.CSS = functional.NormalizeJson(incoming.CSS, &obj.CSS{})

			if err := db.UpdateGame(request.Ctx, request.User.ID, existing); err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to update game: " + err.Error()}
			}

			return "OK", nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
