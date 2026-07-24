// package: routes / server status HTTP handler
// type:    logic
// job:     handles the public health/status endpoint reporting server state and uptime
// limits:  no route registration (-> router.go); build info (-> version.go)
package routes

import (
	"net/http"
	"time"

	"cgl/api/httpx"
	"cgl/functional"
)

var serverStartTime = time.Now()

// StatusResponse reports the server's health status and uptime.
type StatusResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// GetStatus godoc
//
//	@Summary		Get server status
//	@Description	Returns the current server status and uptime
//	@Tags			status
//	@Produce		json
//	@Success		200	{object}	StatusResponse
//	@Router			/status [get]
func GetStatus(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, StatusResponse{
		Status: "running",
		Uptime: functional.HumanizeDuration(time.Since(serverStartTime)),
	})
}
