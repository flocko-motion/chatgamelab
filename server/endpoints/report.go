package endpoints

import (
	"fmt"
	"cgl/obj"
	"cgl/api"
)

// Report endpoint - temporarily disabled pending session refactor
var Report = api.NewEndpoint(
	"/api/report",
	true,
	"text/csv",
	func(request api.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api report called")
		if request.GetParam("secret") != "ccccccngiibddiecnllviecihbgflufetrrjhdthcfib" {
			return nil, &obj.HTTPError{StatusCode: 403, Message: "Forbidden"}
		}
		// TODO: Reimplement using new GameSession types
		return nil, &obj.HTTPErrorNotImplemented
	},
)
