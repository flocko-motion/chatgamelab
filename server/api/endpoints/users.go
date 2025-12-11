package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
)

var UsersList = handler.NewEndpoint(
	"/api/users",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		users, err := db.GetAllUsers(request.Ctx)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get users: " + err.Error()}
		}
		return users, nil
	},
)
