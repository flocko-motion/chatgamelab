package endpoints

import (
	"cgl/obj"
	"cgl/api"
)

var PublicSession = api.NewEndpoint(
	"/api/public/session/",
	true,
	"application/json",
	func(request api.Request) (out interface{}, httpErr *obj.HTTPError) {
		return handleSessionRequest(request, true)
	},
)
