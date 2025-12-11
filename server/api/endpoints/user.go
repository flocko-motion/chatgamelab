package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"
	"io"
	"net/http"
)

type userDetail struct {
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Nickname   string `json:"nickname"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	Locale     string `json:"locale"`
	Email      string `json:"email"`
}

var User = handler.NewEndpoint(
	"/api/user",
	false,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}

		// GET: return user info
		if request.R.Method == "GET" {
			return request.User, nil
		}

		// POST: update user info
		var postUser userDetail
		bodyBytes, err := io.ReadAll(request.R.Body)
		if err == nil {
			err = json.Unmarshal(bodyBytes, &postUser)
		}
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad request"}
		}

		// Check if name or email changed
		emailChanged := (request.User.Email == nil && postUser.Email != "") ||
			(request.User.Email != nil && postUser.Email != *request.User.Email)
		nameChanged := postUser.Name != request.User.Name

		if nameChanged || emailChanged {
			var email *string
			if postUser.Email != "" {
				email = &postUser.Email
			}
			if err := db.UpdateUserDetails(request.Ctx, request.User.ID, postUser.Name, email); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update user"}
			}
			// Refresh user data
			request.User, err = db.GetUserByID(request.Ctx, request.User.ID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get updated user"}
			}
		}

		// TODO: API key management moved to separate endpoints
		return request.User, nil
	},
)

// API key management endpoints - to be implemented
var UserApiKeys = handler.NewEndpoint(
	"/api/user/apikeys",
	false,
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		// TODO: Implement API key CRUD using db.GetUserApiKeys, db.CreateUserApiKey, db.DeleteUserApiKey
		return nil, &obj.HTTPError{StatusCode: http.StatusNotImplemented, Message: "API keys endpoint not yet implemented"}
	},
)
