package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
	"fmt"
)

// Report endpoint - temporarily disabled pending session refactor
var Report = handler.NewEndpoint(
	"/api/report",
	true,
	"text/csv",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api report called")
		if request.GetParam("secret") != "ccccccngiibddiecnllviecihbgflufetrrjhdthcfib" {
			return nil, &obj.HTTPError{StatusCode: 403, Message: "Forbidden"}
		}
		// TODO: Reimplement using new GameSession types
		return nil, &obj.HTTPErrorNotImplemented
	},
)
