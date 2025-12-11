package endpoints

import (
	"cgl/obj"
	"cgl/api"
)

// Set by main package / CLI
var (
	GitCommit = "dev"
	BuildTime = "unknown"
	DevMode   = false
)

var Version = api.NewEndpoint(
	"/api/version",
	true,
	"application/json",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		return map[string]string{
			"version":   GitCommit,
			"buildTime": BuildTime,
		}, nil
	},
)
