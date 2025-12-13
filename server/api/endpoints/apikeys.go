package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"net/http"
)

// ApiKeys handles:
// GET /api/apikeys - List all shares accessible to current user
var ApiKeys = handler.NewEndpoint(
	"/api/apikeys",
	handler.AuthRequired,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		if request.R.Method != http.MethodGet {
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}

		keys, err := db.GetApiKeySharesByUser(request.Ctx, request.User.ID)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get API keys: " + err.Error()}
		}
		return keys, nil
	},
)
