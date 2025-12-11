package endpoints

import (
	"log"
	"path"
	"cgl/db"
	"cgl/obj"
	"cgl/api"
)

var PublicGame = api.NewEndpoint(
	"/api/public/game/",
	true,
	"application/json",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		gameToken := path.Base(request.R.URL.Path)
		log.Printf("gameToken: %s, method: %s", gameToken, request.R.Method)
		game, err := db.GetGameByToken(request.Ctx, gameToken)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
		}
		return game, nil
	},
)
