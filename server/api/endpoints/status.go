package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
	"fmt"
)

var Status = handler.NewEndpoint(
	"/api/status",
	true,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api status called")
		return "running", nil
	},
)
