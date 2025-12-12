package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

type ShareRequest struct {
	UserID        *uuid.UUID `json:"userId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
	AllowPublic   bool       `json:"allowPublicSponsoredPlays"`
}

type ShareResponse struct {
	ID uuid.UUID `json:"id"`
}

type UpdateApiKeyRequest struct {
	Name *string `json:"name,omitempty"`
}

// FYI: ApiKeys are always delivered to the user as ApiKeyShares - so it's a wrapped format.
// When adding a new ApiKey, the system automatically creates a share for the owning user. The owner
// can then add shares for other users, institutions, workshops, etc. to also use the key. The underlying key itself
// always stays on the server, only the shares are made visible to the user(s).
//
// ApiKeysId handles:
// GET    /api/apikeys/{id} - Get share info + linked shares (if owner)
// POST   /api/apikeys/{id} - Share this key with user/workshop/institution (owner only)
// PATCH  /api/apikeys/{id} - Update key name (owner only)
// DELETE /api/apikeys/{id}?cascade=true - Delete key + all shares (owner only)
// DELETE /api/apikeys/{id} - Unshare (delete single share)
var ApiKeysId = handler.NewEndpoint(
	"/api/apikeys/{id:uuid}",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		shareID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid share ID"}
		}

		switch request.R.Method {
		case http.MethodGet:
			share, linkedShares, err := db.GetApiKeyShareInfo(request.Ctx, request.User.ID, shareID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get share: " + err.Error()}
			}
			return map[string]any{
				"share":        share,
				"linkedShares": linkedShares,
			}, nil

		case http.MethodPost:
			var req ShareRequest
			if httpErr := request.BodyJSON(&req); httpErr != nil {
				return nil, httpErr
			}

			if req.UserID == nil && req.WorkshopID == nil && req.InstitutionID == nil {
				return nil, &obj.HTTPError{StatusCode: 400, Message: "At least one of userId, workshopId, or institutionId is required"}
			}

			newShareID, err := db.ShareApiKeyFromShare(request.Ctx, request.User.ID, shareID, req.UserID, req.WorkshopID, req.InstitutionID, req.AllowPublic)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to share: " + err.Error()}
			}

			return ShareResponse{ID: *newShareID}, nil

		case http.MethodPatch:
			var req UpdateApiKeyRequest
			if httpErr := request.BodyJSON(&req); httpErr != nil {
				return nil, httpErr
			}

			if req.Name != nil {
				if err := db.UpdateApiKeyNameFromShare(request.Ctx, request.User.ID, shareID, *req.Name); err != nil {
					return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update: " + err.Error()}
				}
			}

			return map[string]string{"status": "updated"}, nil

		case http.MethodDelete:
			cascade := request.R.URL.Query().Get("cascade") == "true"

			if cascade {
				if err := db.DeleteApiKeyFromShare(request.Ctx, request.User.ID, shareID); err != nil {
					return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to delete key: " + err.Error()}
				}
				return map[string]string{"status": "deleted"}, nil
			}

			if err := db.DeleteApiKeyShare(request.Ctx, request.User.ID, shareID); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to unshare: " + err.Error()}
			}
			return map[string]string{"status": "unshared"}, nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
