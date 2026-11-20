// package: api / background services
// type:    wiring
// job:     prepares the platform once schemaf has the database up: telemetry, dev logging and preseed, admin promotions; flushes telemetry at shutdown
// limits:  schemaf opens the database and serves HTTP; the preseed and promotions live in db
package api

import (
	"context"
	"time"

	"cgl/db"
	"cgl/log"
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
		db.Preseed(ctx)
	}
	db.PromoteAdminEmails(ctx)

	<-ctx.Done()
	// The server is stopping; the database stays open until the services
	// return, so every session writes its last checkpoint now.
	Engine.Close()
	telemetry.Flush(2 * time.Second)
}
