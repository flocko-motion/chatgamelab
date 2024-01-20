package api

import (
	"encoding/json"
	"log"
	"path"
	"strconv"
	"webapp-server/obj"
	"webapp-server/router"
)

var Game = router.NewEndpoint(
	"/api/game/",
	false,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		log.Printf("Handling game request for user %s", request.User.Name)

		// new game?
		if path.Base(request.R.URL.Path) == "new" {
			log.Printf("Creating new game..")
			type GameNewRequest struct {
				Title string `json:"title"`
			}
			var gameRequest GameNewRequest
			if err := json.NewDecoder(request.R.Body).Decode(&gameRequest); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
			}
			newGame := obj.Game{
				Title: gameRequest.Title,
				StatusFields: []obj.StatusField{
					{Name: "Gold", Value: "100"},
				},
			}
			if err := request.User.CreateGame(&newGame); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create game: " + err.Error()}
			}
			type GameNewResponse struct {
				GameId uint `json:"id"`
			}
			log.Printf("Created new game with id %d", newGame.ID)
			return GameNewResponse{
				GameId: newGame.ID,
			}, nil

		}

		gameId, err := strconv.ParseUint(path.Base(request.R.URL.Path), 10, 32)
		log.Printf("gameId: %d, method: %s", gameId, request.R.Method)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}
		switch request.R.Method {
		case "DELETE":
			log.Printf("Deleting game %d", gameId)
			return nil, request.User.DeleteGame(uint(gameId))
		case "GET":
			log.Printf("Getting game %d", gameId)
			return request.User.GetGame(uint(gameId))

		case "POST":
			log.Printf("Updating game %d", gameId)
			var updatedGame obj.Game // Replace GameType with your actual game struct type
			err := json.NewDecoder(request.R.Body).Decode(&updatedGame)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
			}

			err = request.User.UpdateGame(updatedGame)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
			}

			return request.User.GetGame(uint(gameId))

		default:
			return nil, &obj.HTTPError{StatusCode: 405, Message: "Method Not Allowed"}
		}
	},
)
