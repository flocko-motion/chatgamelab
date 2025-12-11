package endpoints

import (
	"fmt"
	"cgl/obj"
	"cgl/api"
)

var Status = api.NewEndpoint(
	"/api/status",
	true,
	"application/json",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api status called")
		return "running", nil
	},
)
