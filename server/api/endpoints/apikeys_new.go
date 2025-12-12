package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

type CreateApiKeyRequest struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
}

type CreateApiKeyResponse struct {
	ID uuid.UUID `json:"id"`
}

// ApiKeysNew handles:
// POST /api/apikeys/new - Create new API key + self-share
var ApiKeysNew = handler.NewEndpoint(
	"/api/apikeys/new",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		if request.R.Method != http.MethodPost {
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}

		var req CreateApiKeyRequest
		if httpErr := request.BodyJSON(&req); httpErr != nil {
			return nil, httpErr
		}

		if req.Platform == "" {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Platform is required"}
		}
		if req.Key == "" {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Key is required"}
		}

		id, err := db.CreateUserApiKey(request.Ctx, request.User.ID, req.Name, req.Platform, req.Key)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create API key: " + err.Error()}
		}

		return CreateApiKeyResponse{ID: *id}, nil
	},
)
