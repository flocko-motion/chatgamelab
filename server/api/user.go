package api

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"webapp-server/obj"
	"webapp-server/router"
)

type userDetail struct {
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Nickname          string `json:"nickname"`
	Name              string `json:"name"`
	Picture           string `json:"picture"`
	Locale            string `json:"locale"`
	Email             string `json:"email"`
	OpenaiKeyPersonal string `json:"openaiKeyPersonal"`
	OpenaiKeyPublish  string `json:"openaiKeyPublish"`
}

var User = router.NewEndpointJson(
	"/api/user",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}

		// get user details from request body
		var postUser userDetail
		// Read the entire body
		bodyBytes, err := io.ReadAll(request.R.Body)
		if err == nil {
			err = json.Unmarshal(bodyBytes, &postUser)
		}
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad request"}
		}

		if postUser.Name != request.User.Name || postUser.Email != request.User.Email {
			request.User.Update(postUser.Name, postUser.Email)
		}

		if postUser.OpenaiKeyPublish == "" || isOpenaiApiKey(postUser.OpenaiKeyPublish) {
			request.User.UpdateApiKeyPublish(postUser.OpenaiKeyPublish)
		}
		if postUser.OpenaiKeyPersonal == "" || isOpenaiApiKey(postUser.OpenaiKeyPersonal) {
			request.User.UpdateApiKeyPersonal(postUser.OpenaiKeyPersonal)
		}

		return request.User.Export(), nil
	},
)

func isOpenaiApiKey(key string) bool {
	// This is a basic regex pattern for demonstration purposes.
	// Adjust the regex according to the actual key format you expect.
	pattern := `^sk-[A-Za-z0-9]{48}$`
	matched, err := regexp.MatchString(pattern, key)
	if err != nil {
		fmt.Println("Error in regex pattern:", err)
		return false
	}
	return matched
}
