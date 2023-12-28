package api

import (
	"webapp-server/router"
)

var External = router.NewEndpointJson(
	"/api/external",
	false,
	func(request router.Request) (interface{}, *router.HTTPError) {
		res := struct {
			Message string `json:"message"`
		}{}
		res.Message = "XOXO Hello from a private endpoint! You need to be authenticated to see this."
		return res, nil
	},
)
