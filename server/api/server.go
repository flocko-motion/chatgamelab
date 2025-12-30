package api

import (
	"cgl/api/auth"
	"cgl/api/routes"
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

	routes.DevMode = devMode

	if devMode {
		log.Println("Development mode enabled")
	}

	db.Init()
	db.Preseed(ctx)

	// Use new stdlib-based router
	h := routes.Handler()

	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{
		Addr:    bindAddr,
		Handler: h,
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
