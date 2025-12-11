package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
)

// Set by main package / CLI
var (
	GitCommit = "dev"
	BuildTime = "unknown"
	DevMode   = false
)

var Version = handler.NewEndpoint(
	"/api/version",
	true,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		return map[string]string{
			"version":   GitCommit,
			"buildTime": BuildTime,
		}, nil
	},
)
