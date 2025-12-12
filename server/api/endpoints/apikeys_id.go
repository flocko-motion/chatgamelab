package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"
	"net/http"
)

var ApiKeysId = handler.NewEndpoint(
	"/api/apikeys/{id:uuid}",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		apiKeyID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid API key ID"}
		}

		switch request.R.Method {
		case http.MethodGet:
			shares, err := db.GetApiKeyShares(request.Ctx, request.User.ID, apiKeyID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get shares: " + err.Error()}
			}
			return shares, nil

		case http.MethodPatch:
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(request.R.Body).Decode(&req); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request: " + err.Error()}
			}

			if err := db.UpdateUserApiKeyName(request.Ctx, request.User.ID, apiKeyID, req.Name); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update API key: " + err.Error()}
			}
			return map[string]string{"status": "updated"}, nil

		case http.MethodDelete:
			if err := db.DeleteUserApiKey(request.Ctx, request.User.ID, apiKeyID); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to delete API key: " + err.Error()}
			}
			return map[string]string{"status": "deleted"}, nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
