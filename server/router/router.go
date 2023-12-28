package router

import (
	"net/http"

	"github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"

	"webapp-server/middleware"
)

// New sets up our routes and returns a *http.ServeMux.
func New(endpoints []Endpoint) *http.ServeMux {
	router := http.NewServeMux()

	// This route is always accessible.
	router.Handle("/api/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Hello from a public endpoint! You don't need to be authenticated to see this."}`))
	}))

	for _, endpoint := range endpoints {
		if endpoint.Public {
			router.Handle(endpoint.Path, endpoint.Handler)
		} else {
			router.Handle(endpoint.Path, middleware.EnsureValidToken()(
				endpoint.Handler,
			))
		}
	}
	// This route is only accessible if the user has a valid access_token.
	//router.Handle("/api/external", middleware.EnsureValidToken()(
	//	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		// CORS Headers.
	//		//w.Header().Set("Access-Control-Allow-Credentials", "true")
	//		//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	//		//w.Header().Set("Access-Control-Allow-Headers", "Authorization")
	//		SetCorsHeaders(w)
	//
	//		w.Header().Set("Content-Type", "application/json")
	//		w.WriteHeader(http.StatusOK)
	//		w.Write([]byte(`{"message":"Hello from a private endpoint! You need to be authenticated to see this."}`))
	//	}),
	//))

	// This route is only accessible if the user has a
	// valid access_token with the read:messages scope.
	router.Handle("/api/private-scoped", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS Headers.
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")

			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasScope("read:messages") {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"message":"Insufficient scope."}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"Hello from a private endpoint! You need to be authenticated to see this."}`))
		}),
	))

	return router
}
