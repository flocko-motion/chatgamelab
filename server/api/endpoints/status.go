package endpoints

import (
	"cgl/api/handler"
	"cgl/functional"
	"cgl/obj"
	"time"
)

var startTime = time.Now()

type StatusResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

var Status = handler.NewEndpoint(
	"/api/status",
	handler.AuthNone,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		return StatusResponse{
			Status: "running",
			Uptime: functional.HumanizeDuration(time.Since(startTime)),
		}, nil
	},
)
