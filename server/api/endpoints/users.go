package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
)

var Users = handler.NewEndpoint(
	"/api/users",
	false, // Requires authentication
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		users, err := db.GetAllUsers(request.Ctx)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get users: " + err.Error()}
		}
		return users, nil
	},
)
