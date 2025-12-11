package api

import (
	"cgl/api/auth"
	"cgl/api/endpoints"
	"cgl/api/handler"
	"cgl/db"
	"context"
	"fmt"
	"log"
	"net/http"
)

func RunServer(ctx context.Context, port int, devMode bool) {

	auth.InitJwtGeneration()

	endpoints.DevMode = devMode

	if devMode {
		log.Println("Development mode enabled")
	}

	db.Init()
	db.Preseed(ctx)

	endpointList := []handler.Endpoint{
		endpoints.GamesList,
		endpoints.GamesNew,
		endpoints.GamesId,
		endpoints.GamesIdYaml,
		endpoints.Image,
		endpoints.Report,
		endpoints.Session,
		endpoints.Status,
		endpoints.Upgrade,
		endpoints.UsersList,
		endpoints.UsersMe,
		endpoints.UsersId,
		endpoints.Version,
		endpoints.PublicGame,
		endpoints.PublicSession,
	}
	if devMode {
		endpointList = append(endpointList,
			endpoints.UsersNew,
			endpoints.UsersJwt,
		)
	}
	mux := NewRouter(endpointList)

	http.Handle("/", handler.CorsMiddleware(mux))

	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Server listening on %s\n", bindAddr)
	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}

// Router handles routing with path parameter support
type Router struct {
	endpoints []handler.Endpoint
}

// NewRouter sets up our routes and returns a Router.
func NewRouter(endpoints []handler.Endpoint) *Router {
	// Wrap all endpoints with auth middleware (extracts user if token present)
	// For public endpoints, user is optional; for private, it's required
	for i, endpoint := range endpoints {
		endpoints[i].Handler = handler.EnsureValidToken()(endpoint.Handler).ServeHTTP
	}
	return &Router{endpoints: endpoints}
}

// ServeHTTP implements http.Handler
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, endpoint := range router.endpoints {
		if endpoint.MatchesPath(r.URL.Path) {
			endpoint.Handler(w, r)
			return
		}
	}
	// No match found
	http.NotFound(w, r)
}
