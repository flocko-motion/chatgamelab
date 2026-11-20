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
)

func init() {
	db.DevMode = devMode
}

// Startup prepares the platform and then waits for shutdown to flush
// telemetry; a schemaf service.
func Startup(ctx context.Context) {
	telemetry.Init(Version)

	if devMode {
		log.SetDebug(true)
		log.Info("development mode enabled", "debug_logging", true)
		db.Preseed(ctx)
	}
	db.PromoteAdminEmails(ctx)

	<-ctx.Done()
	telemetry.Flush(2 * time.Second)
}
