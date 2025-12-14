package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"log"
)

var GamesId = handler.NewEndpoint(
	"/api/games/{id}",
	handler.AuthOptional,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		gameID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid game ID"}
		}
		log.Printf("GamesById: %s %s", request.R.Method, gameID)

		switch request.R.Method {
		case "GET":
			game, err := db.GetGameByID(request.Ctx, &request.User.ID, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
			}
			return game, nil

		case "POST":
			var updatedGame obj.Game
			if httpErr := request.BodyJSON(&updatedGame); httpErr != nil {
				return nil, httpErr
			}
			updatedGame.ID = gameID

			if err := db.UpdateGame(request.Ctx, request.User.ID, &updatedGame); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update game: " + err.Error()}
			}

			game, err := db.GetGameByID(request.Ctx, &request.User.ID, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
			}
			return game, nil

		case "DELETE":
			if err := db.DeleteGame(request.Ctx, request.User.ID, gameID); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to delete game: " + err.Error()}
			}
			return nil, nil

		default:
			return nil, &obj.HTTPError{StatusCode: 405, Message: "Method Not Allowed"}
		}
	},
)
