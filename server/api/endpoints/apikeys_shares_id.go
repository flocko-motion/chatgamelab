package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"net/http"
)

var ApiKeysSharesId = handler.NewEndpoint(
	"/api/apikeys/shares/{id:uuid}",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		shareID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid share ID"}
		}

		switch request.R.Method {
		case http.MethodDelete:
			if err := db.DeleteApiKeyShare(request.Ctx, request.User.ID, shareID); err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to delete share: " + err.Error()}
			}
			return map[string]string{"status": "deleted"}, nil

		default:
			return nil, &obj.HTTPError{StatusCode: http.StatusMethodNotAllowed, Message: "Method not allowed"}
		}
	},
)
