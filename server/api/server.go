package api

import (
	"cgl/api/auth"
	"cgl/api/routes"
	"cgl/db"
	"cgl/log"
	"context"
	"fmt"
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
		log.Info("development mode enabled")
	}

	log.Debug("initializing database")
	db.Init()

	log.Debug("initializing system settings")
	if err := db.InitSystemSettings(ctx); err != nil {
		log.Fatal("failed to initialize system settings", "error", err)
	}

	log.Debug("running database preseed")
	db.Preseed(ctx)

	log.Debug("setting up HTTP router")
	// Use new stdlib-based router
	h := routes.Handler()

	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{
		Addr:    bindAddr,
		Handler: h,
	}

	// Start server in goroutine
	go func() {
		log.Info("server listening", "address", bindAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal or context cancellation
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("received shutdown signal")
	case <-ctx.Done():
		log.Info("context cancelled")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
