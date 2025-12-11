package endpoints

import (
	"log"
	"os"
	"time"
	"cgl/obj"
	"cgl/api"
)

var Upgrade = api.NewEndpoint(
	"/api/upgrade",
	true,
	"application/json",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		log.Println("upgrade docker request - exiting server")
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("shutting down server")
			os.Exit(0)
		}()
		return "Server will shutdown in 10 seconds, upgrade to new version, if available, and then restart. This usually takes 1-3 minutes.", nil
	},
)
