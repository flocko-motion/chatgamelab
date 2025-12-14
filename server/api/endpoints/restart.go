package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
	"log"
	"os"
	"time"
)

var Restart = handler.NewEndpoint(
	"/api/restart",
	handler.AuthRequired,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		if httpErr := request.RequireAdmin(); httpErr != nil {
			return nil, httpErr
		}
		log.Println("restart request - exiting server")
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("shutting down server")
			os.Exit(0)
		}()
		return "Server will shutdown in 10 seconds, upgrade to new version, if available, and then restart. This usually takes 1-3 minutes.", nil
	},
)
