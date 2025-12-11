package api

import (
	"cgl/api/endpoints"
	"cgl/api/handler"
	"cgl/db"
	"fmt"
	"log"
	"net/http"
)

func RunServer(port int, devMode bool) {
	endpoints.DevMode = devMode

	if devMode {
		log.Println("Development mode enabled")
	}

	db.Init()

	endpointList := []handler.Endpoint{
		endpoints.Game,
		endpoints.Games,
		endpoints.Image,
		endpoints.Report,
		endpoints.Session,
		endpoints.Status,
		endpoints.Upgrade,
		endpoints.User,
		endpoints.Users,
		endpoints.Version,
		endpoints.PublicGame,
		endpoints.PublicSession,
	}
	if devMode {
		endpointList = append(endpointList,
			endpoints.UserAdd,
			endpoints.UserJwt,
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

// NewRouter sets up our routes and returns a *http.ServeMux.
func NewRouter(endpoints []handler.Endpoint) *http.ServeMux {
	router := http.NewServeMux()

	for _, endpoint := range endpoints {
		if endpoint.Public {
			router.Handle(endpoint.Path, endpoint.Handler)
		} else {
			router.Handle(endpoint.Path, handler.EnsureValidToken()(
				endpoint.Handler,
			))
		}
	}
	return router
}
