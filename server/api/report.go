package api

import (
	"fmt"
	"webapp-server/obj"
	"webapp-server/router"
)

// Report endpoint - temporarily disabled pending session refactor
var Report = router.NewEndpoint(
	"/api/report",
	true,
	"text/csv",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api report called")
		if request.GetParam("secret") != "ccccccngiibddiecnllviecihbgflufetrrjhdthcfib" {
			return nil, &obj.HTTPError{StatusCode: 403, Message: "Forbidden"}
		}
		// TODO: Reimplement using new GameSession types
		return nil, &obj.HTTPErrorNotImplemented
	},
)
