package endpoints

import (
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
	"log"

	"github.com/google/uuid"
)

type UserUpdateRequest struct {
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	DefaultApiKeyShareID *uuid.UUID `json:"defaultApiKeyShareId,omitempty"`
}

var UsersId = handler.NewEndpoint(
	"/api/users/{id:uuid}",
	false,
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		userID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Invalid user ID"}
		}
		log.Printf("UsersId: %s %s", request.R.Method, userID)

		switch request.R.Method {
		case "GET":
			user, err := db.GetUserByID(request.Ctx, userID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 404, Message: "User not found"}
			}
			return user, nil

		case "POST":
			// only admins may access other users
			if userID != request.User.ID {
				if httpErr := request.RequireAdmin(); httpErr != nil {
					return nil, httpErr
				}
			}

			var req UserUpdateRequest
			if httpErr := request.BodyJSON(&req); httpErr != nil {
				return nil, httpErr
			}

			user, err := db.GetUserByID(request.Ctx, userID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 404, Message: "User not found"}
			}

			// Check if name or email changed
			emailChanged := (user.Email == nil && req.Email != "") ||
				(user.Email != nil && req.Email != *user.Email)
			nameChanged := req.Name != user.Name

			if nameChanged || emailChanged {
				var email *string
				if req.Email != "" {
					email = &req.Email
				}
				if err := db.UpdateUserDetails(request.Ctx, userID, req.Name, email); err != nil {
					return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to update user"}
				}
			}

			// Handle default API key share update
			if req.DefaultApiKeyShareID != nil {
				if err := db.SetUserDefaultApiKeyShare(request.Ctx, userID, req.DefaultApiKeyShareID); err != nil {
					return nil, &obj.HTTPError{StatusCode: 400, Message: "Failed to set default API key: " + err.Error()}
				}
			}

			// Refresh user data
			user, err = db.GetUserByID(request.Ctx, userID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Failed to get updated user"}
			}

			return user, nil

		default:
			return nil, &obj.HTTPError{StatusCode: 405, Message: "Method Not Allowed"}
		}
	},
)
