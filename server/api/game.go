package api

import (
	"path"
	"webapp-server/router"
)

var Game = router.NewEndpointJson(
	"/api/game/",
	false,
	func(request router.Request) (interface{}, *router.HTTPError) {
		if request.User == nil {
			return nil, &router.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		gameId := path.Base(request.R.URL.Path)
		game, err := request.User.GetGame(gameId)
		if err != nil {
			if err.Error() == "unauthorized" {
				return nil, &router.HTTPError{StatusCode: 401, Message: "Unauthorized"}
			} else {
				return nil, &router.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
			}
		}
		return game, nil
	},
)
