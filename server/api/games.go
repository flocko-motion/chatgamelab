package api

import (
	"webapp-server/obj"
	"webapp-server/router"
)

var Games = router.NewEndpointJson(
	"/api/games",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		games, err := request.User.GetGames()
		return games, err
	},
)
