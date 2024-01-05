package api

import (
	"encoding/json"
	"io"
	"webapp-server/obj"
	"webapp-server/router"
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

var Login = router.NewEndpointJson(
	"/api/login",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}

		// get user details from request body
		var user userDetail
		// Read the entire body
		bodyBytes, err := io.ReadAll(request.R.Body)
		if err == nil {
			err = json.Unmarshal(bodyBytes, &user)
		}
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad request"}
		}

		if user.Name != request.User.Name || user.Email != request.User.Email {
			request.User.Update(user.Name, user.Email)
		}

		return request.User.Export(), nil
	},
)
