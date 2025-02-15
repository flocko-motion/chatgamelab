package api

import (
	"fmt"
	"webapp-server/db"
	"webapp-server/obj"
	"webapp-server/router"
)

var Report = router.NewEndpoint(
	"/api/report",
	true,
	"text/csv",
	func(request router.Request) (interface{}, *obj.HTTPError) {
		fmt.Println("api report called")
		if request.GetParam("secret") != "ccccccngiibddiecnllviecihbgflufetrrjhdthcfib" {
			return nil, &obj.HTTPError{StatusCode: 403, Message: "Forbidden"}
		}
		report, err := db.GetSessionUsageReport()
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
		}
		return report, nil
	},
)
