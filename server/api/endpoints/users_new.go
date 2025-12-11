package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"net/http"
)

type UsersNewRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

var UsersNew = handler.NewEndpoint(
	"/api/users/new",
	true, // public for dev use - endpoint is only existing in --dev mode of server
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		var req UsersNewRequest
		if httpErr := request.BodyJSON(&req); httpErr != nil {
			return nil, httpErr
		}

		if req.Name == "" {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: "Name is required"}
		}

		user, err := db.CreateUser(request.Ctx, req.Name, req.Email, "")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to create user: " + err.Error()}
		}

		return user, nil
	},
)
