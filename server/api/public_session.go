package api

import (
	"webapp-server/obj"
	"webapp-server/router"
)

var PublicSession = router.NewEndpoint(
	"/api/public/session/",
	true,
	"application/json",
	func(request router.Request) (out interface{}, httpErr *obj.HTTPError) {
		return handleSessionRequest(request, true)
	},
)
