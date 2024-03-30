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
		gameHash := path.Base(request.R.URL.Path)
		log.Printf("gameHash: %s, method: %s", gameHash, request.R.Method)
		return db.GetGameByPublicHash(gameHash)
	},
)
