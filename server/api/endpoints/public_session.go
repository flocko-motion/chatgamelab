package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
)

var PublicSession = handler.NewEndpoint(
	"/api/public/session/",
	true,
	"application/json",
	func(request handler.Request) (out interface{}, httpErr *obj.HTTPError) {
		return handleSessionRequest(request, true)
	},
)
