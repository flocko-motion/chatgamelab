package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"
)

// UsersMe is a convenience endpoint that returns the current user's info
// For updates, use /api/users/{id}
var UsersMe = handler.NewEndpoint(
	"/api/users/me",
	handler.AuthRequired,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		return request.User, nil
	},
)
