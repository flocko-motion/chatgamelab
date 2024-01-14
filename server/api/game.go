package api

import (
	"encoding/json"
	"path"
	"strconv"
	"webapp-server/obj"
	"webapp-server/router"
)

var Game = router.NewEndpointJson(
	"/api/game/",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}

		// new game?
		if path.Base(request.R.URL.Path) == "new" {
			type GameNewRequest struct {
				Title string `json:"title"`
			}
			var gameRequest GameNewRequest
			if err := json.NewDecoder(request.R.Body).Decode(&gameRequest); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
			}
			newGame := obj.Game{
				Title: gameRequest.Title,
			}
			if err := request.User.CreateGame(&newGame); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
			}
			type GameNewResponse struct {
				GameId uint `json:"id"`
			}
			return GameNewResponse{
				GameId: newGame.ID,
			}, nil

		}

		gameId, err := strconv.ParseUint(path.Base(request.R.URL.Path), 10, 32)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}
		switch request.R.Method {
		case "DELETE":
			return nil, request.User.DeleteGame(uint(gameId))
		case "GET":
			return request.User.GetGame(uint(gameId))

		case "POST":
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
