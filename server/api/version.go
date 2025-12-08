package api

import (
	"webapp-server/obj"
	"webapp-server/router"
)

var Version = router.NewEndpoint(
	"/api/version",
	true,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		return map[string]string{
			"version": "dev",
		}, nil
	},
)
