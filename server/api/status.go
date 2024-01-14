package api

import (
	"fmt"
	"webapp-server/obj"
	"webapp-server/router"
)

var Status = router.NewEndpointJson(
	"/api/status",
	true,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api status called")
		return "running", nil
	},
)
