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
		gameId, err := strconv.ParseUint(path.Base(request.R.URL.Path), 10, 32)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}
		switch request.R.Method {
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
