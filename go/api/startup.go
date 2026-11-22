// package: api / background services
// type:    wiring
// job:     prepares the platform once schemaf has the database up: telemetry and dev logging; ends the engine's sessions and flushes telemetry at shutdown
// limits:  schemaf opens the database and serves HTTP; the data jobs (preseed, first admin, purge) are the core's services (core/system.go)
package api

import (
	"context"
	"time"

	"cgl/log"
	"cgl/mail"
	"cgl/telemetry"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// Startup prepares the platform and then waits for shutdown to end the
// engine's sessions and flush telemetry; a schemaf service.
func Startup(ctx context.Context) {
	telemetry.Init(Version)

	if schemafapi.DevMode() {
		log.SetDebug(true)
		log.Info("development mode enabled", "debug_logging", true)
		mail.KeepSent()
	}

	<-ctx.Done()
	// The server is stopping; the database stays open until the services
	// return, so every session writes its last checkpoint now.
	Engine.Close()
	telemetry.Flush(2 * time.Second)
}
