// package: api / startup and shutdown
// type:    wiring
// job:     prepares the platform before schemaf serves (telemetry, dev logging); ends the engine's sessions and flushes telemetry at shutdown
// limits:  schemaf opens the database and serves HTTP; the data jobs (preseed, first admin, purge) are the core's (core/system.go)
package api

import (
	"context"
	"time"

	"cgl/log"
	"cgl/mail"
	"cgl/telemetry"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// Prepare sets up telemetry and, in development mode, debug logging and the
// kept mail; a schemaf before-serve hook, so it is done before the first
// request.
func Prepare(ctx context.Context) error {
	telemetry.Init(RunningVersion())

	if schemafapi.DevMode() {
		log.SetDebug(true)
		log.Info("development mode enabled", "debug_logging", true)
		mail.KeepSent()
	}
	return nil
}

// Shutdown waits for the server to stop, then ends the engine's sessions and
// flushes telemetry; a schemaf service.
func Shutdown(ctx context.Context) {
	<-ctx.Done()
	// The server is stopping; the database stays open until the services
	// return, so every session writes its last checkpoint now.
	Engine.Close()
	telemetry.Flush(2 * time.Second)
}
