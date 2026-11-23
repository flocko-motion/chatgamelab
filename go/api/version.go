// package: api / schemaf endpoints
// type:    logic
// job:     reports the release the server runs
// limits:  the version is the running image's tag (schemaf's api.Updates); the schema level is schemaf's (/api/status)
package api

import (
	"context"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GetVersionEndpoint reports the release the server runs.
//
// Returns the version of the running release ("dev" where none runs, as in
// development). Frontend and server ship together, so a page compares it
// with the version it was loaded with to see that the server was upgraded.
type GetVersionEndpoint struct{}

func (GetVersionEndpoint) Method() string             { return "GET" }
func (GetVersionEndpoint) Path() string               { return "/api/version" }
func (GetVersionEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetVersionEndpoint) Handle(_ context.Context, _ GetVersionReq) (GetVersionResp, error) {
	return GetVersionResp{Version: RunningVersion()}, nil
}

// RunningVersion is the release the server runs: its image's tag, "dev"
// where none runs, as in development.
func RunningVersion() string {
	if state, err := schemafapi.Updates(); err == nil && state.Running != "" {
		return state.Running
	}
	return "dev"
}

// GetVersionReq takes no parameters.
type GetVersionReq struct{}

// GetVersionResp is the release the server runs.
type GetVersionResp struct {
	Version string `json:"version"`
}
