package router

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"webapp-server/db"
	"webapp-server/obj"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
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
	User *obj.User
	Ctx  context.Context
}

func (r *Request) GetParam(key string) string {
	return r.R.URL.Query().Get(key)
}

type Handler func(request Request) (interface{}, *obj.HTTPError)

func NewEndpoint(path string, public bool, contentType string, handler Handler) Endpoint {
	endpoint := Endpoint{
		Path:           path,
		Public:         public,
		RequiredScopes: []string{},
		ContentType:    contentType,
	}

	endpoint.Handler = func(w http.ResponseWriter, r *http.Request) {
		var httpError *obj.HTTPError
		var err error

		request := Request{
			R:   r,
			Ctx: context.Background(),
		}

		SetCorsHeaders(w)
		SetNoCacheHeaders(w)
		w.Header().Set("Content-Type", endpoint.ContentType)

		log.Printf("Handling request for %s", r.URL.Path)
		tokenObj := r.Context().Value(jwtmiddleware.ContextKey{})
		if tokenObj != nil {
			token := tokenObj.(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*CustomClaims)
			for _, requiredScope := range endpoint.RequiredScopes {
				if !claims.HasScope(requiredScope) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"message":"Insufficient scope."}`))
					return
				}
			}

			if auth0ID := token.RegisteredClaims.Subject; auth0ID != "" {
				request.User, err = db.GetUserByAuth0ID(request.Ctx, auth0ID)

				// unknown user - create them
				if err != nil {
					request.User, err = db.CreateUser(request.Ctx, "Unnamed Auth0 User", nil, auth0ID)
					if err != nil {
						httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to create user"}
					}
				}
			}
		}

		var res interface{}
		if httpError == nil {
			log.Printf("Passing over to handler")
			res, httpError = handler(request)
		}

		var resBytes []byte
		if httpError == nil {
			switch endpoint.ContentType {
			case "application/json":
				if resBytes, err = json.Marshal(res); err != nil {
					httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to marshal json"}
				}
			case "image/png":
				resBytes = res.([]byte)
			case "text/csv":
				resBytes = []byte(res.(string))
			default:
				httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Handler has unknown content type"}
			}
		}

		if httpError != nil {
			w.Header().Set("Content-Type", "application/json")
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
