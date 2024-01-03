package api

import (
	"time"
	"webapp-server/router"
)

var External = router.NewEndpointJson(
	"/api/external",
	false,
	func(request router.Request) (interface{}, *router.HTTPError) {
		res := struct {
			Message string `json:"message"`
		}{}

		dateTimeString := time.Now().Format("2006-01-02 15:04:05")

		res.Message = dateTimeString + " Hello from a private endpoint! You need to be authenticated to see this."
		return res, nil
	},
)
