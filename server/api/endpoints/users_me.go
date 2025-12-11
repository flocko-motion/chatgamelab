package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
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

var UsersMe = handler.NewEndpoint(
	"/api/users/me",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		// GET: return user info
		if request.R.Method == "GET" {
			return request.User, nil
		}

		// POST: update user info
		var postUser userDetail
		if httpErr := request.BodyJSON(&postUser); httpErr != nil {
			return nil, httpErr
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
			var err error
			request.User, err = db.GetUserByID(request.Ctx, request.User.ID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get updated user"}
			}
		}

		return request.User, nil
	},
)
