package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"
	"net/http"
)

type UserAddRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

var UserAdd = handler.NewEndpoint(
	"/api/user/add",
	true, // public for dev use
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		var req UserAddRequest
		if err := json.NewDecoder(request.R.Body).Decode(&req); err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: "Invalid request body"}
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
