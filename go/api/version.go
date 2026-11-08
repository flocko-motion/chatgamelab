// package: api / schemaf endpoints
// type:    logic
// job:     reports the server's build: version, git commit and build time
// limits:  build values come from -ldflags; the schema level is schemaf's (/api/status)
package api

import (
	"context"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// Build values, set with -ldflags "-X cgl/api.Version=..." at build time.
var (
	Version   = "dev"
	GitCommit = "dev"
	BuildTime = "unknown"
)

// GetVersionEndpoint reports the server's build.
//
// Returns the version, git commit and build time the server was built with.
type GetVersionEndpoint struct{}

func (GetVersionEndpoint) Method() string             { return "GET" }
func (GetVersionEndpoint) Path() string               { return "/api/version" }
func (GetVersionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetVersionEndpoint) Handle(_ context.Context, _ GetVersionReq) (GetVersionResp, error) {
	return GetVersionResp{Version: Version, GitCommit: GitCommit, BuildTime: BuildTime}, nil
}

// GetVersionReq takes no parameters.
type GetVersionReq struct{}

// GetVersionResp is the server's build.
type GetVersionResp struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
}
