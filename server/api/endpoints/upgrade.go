package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
	"log"
	"os"
	"time"
)

var Upgrade = handler.NewEndpoint(
	"/api/upgrade",
	true,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		log.Println("upgrade docker request - exiting server")
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("shutting down server")
			os.Exit(0)
		}()
		return "Server will shutdown in 10 seconds, upgrade to new version, if available, and then restart. This usually takes 1-3 minutes.", nil
	},
)
