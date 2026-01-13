package routes

import (
	"net/http"

	"cgl/api/httpx"
)

// Version info (set via main package at startup)
var (
	GitCommit = "dev"
	Version   = "dev"
	BuildTime = "unknown"
)

type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
}

// GetVersion godoc
//
//	@Summary		Get server version
//	@Description	Returns the server version and build time
//	@Tags			status
//	@Produce		json
//	@Success		200	{object}	VersionResponse
//	@Router			/version [get]
func GetVersion(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, VersionResponse{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	})
}
