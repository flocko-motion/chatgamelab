package api

import (
	"cgl/api/auth"
	"cgl/api/routes"
	"cgl/db"
	"cgl/log"
	"cgl/telemetry"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunServer(ctx context.Context, port int, devMode bool, readyChan chan struct{}) {

	telemetry.Init(routes.Version)

	auth.InitJwtGeneration()

	routes.DevMode = devMode

	if devMode {
		log.SetDebug(true)
		log.Info("development mode enabled", "debug_logging", true)
	}

	log.Debug("initializing database")
	db.Init()
	log.Debug("running database preseed")
	db.Preseed(ctx)

	log.Debug("checking admin email promotions")
	db.PromoteAdminEmails(ctx)

	log.Debug("setting up HTTP router")
	// Use new stdlib-based router
	h := routes.Handler()

	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{
		Addr:    bindAddr,
		Handler: h,
	}

	log.Info("server starting", "address", bindAddr)

	// Signal that server is ready (DB initialized, router set up)
	if readyChan != nil {
		close(readyChan)
	}

	// Start server in goroutine
	go func() {
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

	telemetry.Flush(2 * time.Second)
	log.Info("server stopped")
}
