package endpoints

import (
	"net/http"

	"cgl/api/auth"
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

type UserJwtResponse struct {
	UserID   string `json:"user_id"`
	Auth0ID  string `json:"auth0_id"`
	Token    string `json:"token"`
	ExpireAt int64  `json:"expire_at"`
}

var UserJwt = handler.NewEndpoint(
	"/api/user/jwt",
	true, // public for dev use
	"application/json",
	func(request handler.Request) (interface{}, *obj.HTTPError) {
		if !DevMode {
			return nil, &obj.HTTPError{StatusCode: http.StatusForbidden, Message: "JWT generation only available in dev mode"}
		}

		userIdentifier := request.R.URL.Query().Get("id")
		if userIdentifier == "" {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: "id query parameter is required"}
		}

		var userID uuid.UUID
		var auth0ID string

		// Try parsing as UUID first
		if id, err := uuid.Parse(userIdentifier); err == nil {
			user, err := db.GetUserByID(request.Ctx, id)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "User not found by ID"}
			}
			userID = user.ID
			if user.Auth0Id != nil {
				auth0ID = *user.Auth0Id
			}
		} else {
			// Try as Auth0 ID
			user, err := db.GetUserByAuth0ID(request.Ctx, userIdentifier)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "User not found by Auth0 ID"}
			}
			userID = user.ID
			if user.Auth0Id != nil {
				auth0ID = *user.Auth0Id
			}
		}

		tokenString, expireAt, err := auth.GenerateToken(userID.String())
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to sign token"}
		}

		return UserJwtResponse{
			UserID:   userID.String(),
			Auth0ID:  auth0ID,
			Token:    tokenString,
			ExpireAt: expireAt,
		}, nil
	},
)
