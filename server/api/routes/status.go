package routes

import (
	"net/http"
	"time"

	"cgl/api/httpx"
	"cgl/functional"
)

var serverStartTime = time.Now()

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
