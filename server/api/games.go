package api

import (
	"webapp-server/router"
)

var Games = router.NewEndpointJson(
	"/api/games",
	false,
	func(request router.Request) (interface{}, *router.HTTPError) {
		if request.User == nil {
			return nil, &router.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		games, err := request.User.GetGames()
		if err != nil {
			return nil, nil
		}
		return games, nil
	},
)
