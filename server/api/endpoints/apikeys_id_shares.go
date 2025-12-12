package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type CreateShareRequest struct {
	UserID        *uuid.UUID `json:"userId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
	AllowPublic   bool       `json:"allowPublicSponsoredPlays"`
}

type CreateShareResponse struct {
	ID uuid.UUID `json:"id"`
}

var ApiKeysIdShares = handler.NewEndpoint(
	"/api/apikeys/{id:uuid}/shares",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		apiKeyID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid API key ID"}
		}

		switch request.R.Method {
		case http.MethodPost:
			var req CreateShareRequest
			if err := json.NewDecoder(request.R.Body).Decode(&req); err != nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request: " + err.Error()}
			}

			if req.UserID == nil && req.WorkshopID == nil && req.InstitutionID == nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "At least one of userId, workshopId, or institutionId is required"}
			}

			shareID, err := db.CreateApiKeyShare(request.Ctx, request.User.ID, apiKeyID, req.UserID, req.WorkshopID, req.InstitutionID, req.AllowPublic)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to create share: " + err.Error()}
			}

			return CreateShareResponse{ID: *shareID}, nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
