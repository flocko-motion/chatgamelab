package api

import (
	"log"
	"os"
	"time"
	"webapp-server/obj"
	"webapp-server/router"
)

var Upgrade = router.NewEndpoint(
	"/api/upgrade",
	true,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		log.Println("upgrade docker request - exiting server")
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("shutting down server")
			os.Exit(0)
		}()
		return "Server will shutdown in 10 seconds, upgrade to new version, if available, and then restart. This usually takes 1-3 minutes.", nil
	},
)
