// package: routes / version HTTP handler
// type:    logic
// job:     handles the endpoint reporting build version, git commit, build time, and DB level
// limits:  no route registration (-> router.go); build vars injected by main at startup
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
)

// Version info (set via main package at startup)
var (
	GitCommit = "dev"
	Version   = "dev"
	BuildTime = "unknown"
)

// VersionResponse reports the server build version, git commit, build time, and DB schema level.
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
	DbLevel   int    `json:"dbLevel"`
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
	dbLevel, err := db.GetCurrentSchemaVersion()
	if err != nil {
		log.Warn("failed to get schema version", "error", err)
	}

	httpx.WriteJSON(w, http.StatusOK, VersionResponse{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		DbLevel:   dbLevel,
	})
}
