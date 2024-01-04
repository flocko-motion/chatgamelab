package router

import (
	"encoding/json"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"net/http"
	"webapp-server/db"
	"webapp-server/obj"
)

type Endpoint struct {
	// The path of the endpoint.
	Path           string
	Public         bool
	RequiredScopes []string
	ContentType    string
	Handler        http.HandlerFunc
}

type Request struct {
	R    *http.Request
	User *db.User
}

type HandlerJson func(request Request) (interface{}, *obj.HTTPError)

func NewEndpointJson(path string, public bool, handler HandlerJson) Endpoint {
	endpoint := Endpoint{
		Path:           path,
		Public:         public,
		RequiredScopes: []string{},
		ContentType:    "application/json",
	}

	endpoint.Handler = func(w http.ResponseWriter, r *http.Request) {
		SetCorsHeaders(w)
		w.Header().Set("Content-Type", "application/json")

		token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

		claims := token.CustomClaims.(*CustomClaims)
		for _, requiredScope := range endpoint.RequiredScopes {
			if !claims.HasScope(requiredScope) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"Insufficient scope."}`))
				return
			}
		}

		var httpError *obj.HTTPError
		var err error

		request := Request{
			R: r,
		}

		if userId := token.RegisteredClaims.Subject; userId != "" {
			request.User, err = db.GetUserByAuth0ID(userId)

			// unknown user
			if err != nil {
				newUser := &db.User{
					Auth0ID: userId,
				}
				if err = db.CreateUser(newUser); err != nil {
					httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to create user"}
				} else {
					request.User = newUser
				}
			}
		}

		var res interface{}
		if httpError == nil {
			res, httpError = handler(request)
		}

		var resBytes []byte
		if httpError == nil {
			if resBytes, err = json.Marshal(res); err != nil {
				httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to marshal json"}
			}
		}

		if httpError != nil {
			w.WriteHeader(httpError.StatusCode)
			_, _ = w.Write(httpError.Json())
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resBytes)
	}

	return endpoint
}

// NewRouter sets up our routes and returns a *http.ServeMux.
func NewRouter(endpoints []Endpoint) *http.ServeMux {
	router := http.NewServeMux()

	//// This route is always accessible.
	//router.Handle("/api/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	w.Header().Set("Content-Type", "application/json")
	//	w.WriteHeader(http.StatusOK)
	//	_, _ = w.Write([]byte(`{"message":"Hello from a public endpoint! You don't need to be authenticated to see this."}`))
	//}))

	for _, endpoint := range endpoints {
		if endpoint.Public {
			router.Handle(endpoint.Path, endpoint.Handler)
		} else {
			router.Handle(endpoint.Path, EnsureValidToken()(
				endpoint.Handler,
			))
		}
	}
	return router
}
