package api

import (
	"fmt"
	"webapp-server/obj"
	"webapp-server/router"
)

var Status = router.NewEndpoint(
	"/api/status",
	true,
	"application/json",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api status called")
		return "running", nil
	},
)
