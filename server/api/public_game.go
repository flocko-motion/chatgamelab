package api

import (
	"log"
	"path"
	"webapp-server/db"
	"webapp-server/obj"
	"webapp-server/router"
)

var PublicGame = router.NewEndpoint(
	"/api/public/game/",
	true,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		gameToken := path.Base(request.R.URL.Path)
		log.Printf("gameToken: %s, method: %s", gameToken, request.R.Method)
		game, err := db.GetGameByToken(request.Ctx, gameToken)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
		}
		return game, nil
	},
)
