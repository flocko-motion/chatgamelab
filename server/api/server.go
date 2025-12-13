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
	"os"
	"os/signal"
	"syscall"
	"time"
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
		endpoints.ApiKeys,
		endpoints.ApiKeysNew,
		endpoints.ApiKeysId,
		endpoints.GamesList,
		endpoints.GamesNew,
		endpoints.GamesId,
		endpoints.GamesIdSessions,
		endpoints.MessageStream,
		endpoints.Session,
		endpoints.Status,
		endpoints.Restart,
		endpoints.UsersList,
		endpoints.UsersMe,
		endpoints.UsersId,
		endpoints.Version,
	}
	if devMode {
		endpointList = append(endpointList,
			endpoints.UsersNew,
			endpoints.UsersJwt,
		)
	}
	mux := NewRouter(endpointList)

	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{
		Addr:    bindAddr,
		Handler: handler.CorsMiddleware(mux),
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s\n", bindAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("There was an error with the http server: %v", err)
		}
	}()

	// Wait for interrupt signal or context cancellation
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Received shutdown signal")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
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
		if endpoint.Auth == handler.AuthNone {
			continue
		}
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
