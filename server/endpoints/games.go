package endpoints

import (
	"cgl/db"
	"cgl/obj"
	"cgl/api"
)

var Games = api.NewEndpoint(
	"/api/games",
	false,
	"application/json",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		games, err := db.GetGames(request.Ctx, &request.User.ID, nil)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get games: " + err.Error()}
		}
		return games, nil
	},
)
