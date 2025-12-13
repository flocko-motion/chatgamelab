package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
)

var GamesList = handler.NewEndpoint(
	"/api/games",
	handler.AuthOptional, // public games visible, private games if logged in
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		games, err := db.GetGames(request.Ctx, &request.User.ID, nil)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get games: " + err.Error()}
		}
		return games, nil
	},
)
