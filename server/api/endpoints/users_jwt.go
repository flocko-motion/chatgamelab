package endpoints

import (
	"net/http"

	"cgl/api/auth"
	"cgl/api/handler"
	"cgl/db"
	"cgl/obj"
)

type UsersJwtResponse struct {
	UserID   string `json:"user_id"`
	Auth0ID  string `json:"auth0_id"`
	Token    string `json:"token"`
	ExpireAt int64  `json:"expire_at"`
}

var UsersJwt = handler.NewEndpoint(
	"/api/users/{id:uuid}/jwt",
	true, // public for dev use
	"application/json",
	func(request handler.Request) (res any, httpErr *obj.HTTPError) {
		if !DevMode {
			return nil, &obj.HTTPError{StatusCode: http.StatusForbidden, Message: "JWT generation only available in dev mode"}
		}

		userID, err := request.GetPathParamUUID("id")
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: "Invalid user ID"}
		}

		user, err := db.GetUserByID(request.Ctx, userID)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusNotFound, Message: "User not found"}
		}

		var auth0ID string
		if user.Auth0Id != nil {
			auth0ID = *user.Auth0Id
		}

		tokenString, expireAt, err := auth.GenerateToken(userID.String())
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to sign token"}
		}

		return UsersJwtResponse{
			UserID:   userID.String(),
			Auth0ID:  auth0ID,
			Token:    tokenString,
			ExpireAt: expireAt,
		}, nil
	},
)
