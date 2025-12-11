package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"
	"log"
	"path"

	"github.com/google/uuid"
)

var Game = handler.NewEndpoint(
	"/api/game/",
	false,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		log.Printf("Handling game request for user %s", request.User.Name)

		// new game?
		if path.Base(request.R.URL.Path) == "new" {
			log.Printf("Creating new game..")
			type GameNewRequest struct {
				Name string `json:"name"`
			}
			var gameRequest GameNewRequest
			if err := json.NewDecoder(request.R.Body).Decode(&gameRequest); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
			}
			newGame := obj.Game{
				Name:         gameRequest.Name,
				StatusFields: `[{"name":"Gold","value":"100"}]`,
			}
			if err := db.CreateGame(request.Ctx, request.User.ID, &newGame); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create game: " + err.Error()}
			}
			type GameNewResponse struct {
				GameId uuid.UUID `json:"id"`
			}
			log.Printf("Created new game with id %s", newGame.ID)
			return GameNewResponse{
				GameId: newGame.ID,
			}, nil
		}

		gameIDStr := path.Base(request.R.URL.Path)
		gameID, err := uuid.Parse(gameIDStr)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid game ID"}
		}
		log.Printf("gameId: %s, method: %s", gameID, request.R.Method)

		switch request.R.Method {
		case "DELETE":
			log.Printf("Deleting game %s", gameID)
			if err := db.DeleteGame(request.Ctx, request.User.ID, gameID); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to delete game: " + err.Error()}
			}
			return nil, nil

		case "GET":
			log.Printf("Getting game %s", gameID)
			game, err := db.GetGameByID(request.Ctx, &request.User.ID, gameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
			}
			return game, nil

		case "POST":
			log.Printf("Updating game %s", gameID)
			var updatedGame obj.Game
			if err := json.NewDecoder(request.R.Body).Decode(&updatedGame); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
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

		default:
			return nil, &obj.HTTPError{StatusCode: 405, Message: "Method Not Allowed"}
		}
	},
)
