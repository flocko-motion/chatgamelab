// package: api / HTTP server entrypoint
// type:    wiring
// job:     runs the session retention purge once at startup and then on a ticker for the life of the server.
// limits:  holds no SQL (-> db/purge.go) and no HTTP surface (-> api/routes).
package api

import (
	"cgl/db"
	"cgl/functional"
	"cgl/log"
	"context"
	"strconv"
	"time"
)

const (
	// defaultSessionRetentionHours is how long a session survives its last activity.
	// Only the session of the game a player is currently playing is reachable in the UI,
	// so anything idle beyond this window is dead weight.
	defaultSessionRetentionHours = 48
	// sessionPurgeInterval is how often the purge runs after the startup pass.
	sessionPurgeInterval = time.Hour
)

// startSessionPurger launches the retention purge in the background.
// SESSION_RETENTION_HOURS overrides the window; 0 switches the purge off.
func startSessionPurger(ctx context.Context) {
	retention := sessionRetention()
	if retention <= 0 {
		log.Info("session purge disabled", "env", "SESSION_RETENTION_HOURS=0")
		return
	}

	log.Info("session purge enabled", "retention", retention, "interval", sessionPurgeInterval)

	go func() {
		ticker := time.NewTicker(sessionPurgeInterval)
		defer ticker.Stop()

		for {
			purgeExpiredSessions(ctx, retention)

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// purgeExpiredSessions runs one purge pass, logging the outcome.
func purgeExpiredSessions(ctx context.Context, retention time.Duration) {
	cutoff := time.Now().Add(-retention)

	purged, err := db.PurgeExpiredSessions(ctx, cutoff)
	if err != nil {
		log.Error("session purge failed", "error", err, "cutoff", cutoff, "purged", purged)
		return
	}
	if purged == 0 {
		log.Debug("session purge found nothing to delete", "cutoff", cutoff)
		return
	}

	log.Info("session purge completed", "purged", purged, "cutoff", cutoff)
}

// sessionRetention reads the retention window from the environment.
// An unparsable value falls back to the default rather than leaving sessions to pile up.
func sessionRetention() time.Duration {
	raw := functional.EnvOrDefault("SESSION_RETENTION_HOURS", strconv.Itoa(defaultSessionRetentionHours))

	hours, err := strconv.Atoi(raw)
	if err != nil {
		log.Warn("invalid SESSION_RETENTION_HOURS, using default", "value", raw, "default_hours", defaultSessionRetentionHours)
		return defaultSessionRetentionHours * time.Hour
	}
	if hours < 0 {
		log.Warn("negative SESSION_RETENTION_HOURS, using default", "value", raw, "default_hours", defaultSessionRetentionHours)
		return defaultSessionRetentionHours * time.Hour
	}

	return time.Duration(hours) * time.Hour
}

// defaultApplicationRetentionDays is how long the personal data of a never-confirmed
// organisation application is kept (ORG_APPLICATION_RETENTION_DAYS).
const defaultApplicationRetentionDays = 90

// applicationCleanupInterval is how often expired applications are processed.
const applicationCleanupInterval = 24 * time.Hour

// startApplicationCleanup expires overdue organisation applications and anonymises old
// unconfirmed ones, once at startup and then daily.
func startApplicationCleanup(ctx context.Context) {
	days := defaultApplicationRetentionDays
	if raw := functional.EnvOrDefault("ORG_APPLICATION_RETENTION_DAYS", ""); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			days = v
		} else {
			log.Warn("invalid ORG_APPLICATION_RETENTION_DAYS, using default", "value", raw, "default_days", defaultApplicationRetentionDays)
		}
	}
	retention := time.Duration(days) * 24 * time.Hour
	log.Info("organisation application cleanup enabled", "retention_days", days)

	go func() {
		ticker := time.NewTicker(applicationCleanupInterval)
		defer ticker.Stop()
		for {
			expired, anonymized, err := db.ExpireAndAnonymizeInstitutionApplications(ctx, retention)
			if err != nil {
				log.Error("organisation application cleanup failed", "error", err)
			} else if expired > 0 || anonymized > 0 {
				log.Info("organisation application cleanup", "expired", expired, "anonymized", anonymized)
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}
