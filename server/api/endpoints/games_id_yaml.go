package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
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
			yamlContent, httpErr := request.BodyRaw()
			if httpErr != nil {
				return nil, httpErr
			}

			if err := db.UpdateGameYaml(request.Ctx, request.User.ID, gameID, yamlContent); err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to update game: " + err.Error()}
			}

			return "OK", nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
