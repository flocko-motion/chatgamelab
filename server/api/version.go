package api

import (
	"webapp-server/obj"
	"webapp-server/router"
)

// Set by main package init
var (
	GitCommit = "dev"
	BuildTime = "unknown"
)

var Version = router.NewEndpoint(
	"/api/version",
	true,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		return map[string]string{
			"version":   GitCommit,
			"buildTime": BuildTime,
		}, nil
	},
)
