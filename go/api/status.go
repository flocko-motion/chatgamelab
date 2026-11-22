// package: api / backup health in /api/status
// type:    logic
// job:     grades the newest backup logbook entry and reports it as "backupHealth" in schemaf's /api/status
// limits:  schemaf serves /api/status with status, uptime and its own backup; the logbook is written by docker/db/backup.sh
package api

import (
	"context"
	"strconv"
	"time"

	"cgl/core"
	"cgl/functional"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

func init() {
	schemafapi.RegisterStatus("backupHealth", backupHealthStatus)
}

// The grades backupHealth reports.
const (
	backupDisabled = "disabled" // BACKUP_ENABLED is not "true"
	backupOk       = "ok"       // newest run succeeded and is recent enough
	backupFailed   = "failed"   // newest run failed
	backupStale    = "stale"    // newest run succeeded but is older than the allowed age
	backupUnknown  = "unknown"  // enabled, but no run was ever recorded (e.g. the host cron entry is missing)
)

// defaultBackupMaxAgeHours tolerates a daily backup plus a missed slot before reporting stale.
const defaultBackupMaxAgeHours = 48

// backupHealthLookup bounds the logbook read, so /api/status stays a fast probe.
const backupHealthLookup = 2 * time.Second

// BackupHealth is the grade of the newest backup run and its age.
type BackupHealth struct {
	Status string `json:"status"`        // disabled | ok | failed | stale | unknown
	Age    string `json:"age,omitempty"` // time since the newest recorded run
}

// backupHealthStatus reads the newest logbook entry and grades it. An
// unreadable logbook degrades to "unknown": /api/status is the liveness probe.
func backupHealthStatus() any {
	if functional.EnvOrDefault("BACKUP_ENABLED", "false") != "true" {
		return BackupHealth{Status: backupDisabled}
	}
	ctx, cancel := context.WithTimeout(context.Background(), backupHealthLookup)
	defer cancel()
	latest, err := core.LatestBackup(ctx)
	if err != nil {
		return BackupHealth{Status: backupUnknown}
	}
	grade, age := gradeBackup(latest, backupMaxAge())
	health := BackupHealth{Status: grade}
	if age != nil {
		health.Age = functional.HumanizeDuration(*age)
	}
	return health
}

// gradeBackup turns the newest logbook entry into a grade. A nil entry means
// no run was ever recorded, which is the shape a missing host cron entry takes.
func gradeBackup(latest *obj.BackupLog, maxAge time.Duration) (string, *time.Duration) {
	if latest == nil {
		return backupUnknown, nil
	}
	age := time.Since(latest.FinishedAt)
	switch {
	case !latest.Success:
		return backupFailed, &age
	case age > maxAge:
		return backupStale, &age
	default:
		return backupOk, &age
	}
}

// backupMaxAge is BACKUP_MAX_AGE_HOURS, or the default for an unset or invalid value.
func backupMaxAge() time.Duration {
	hours, err := strconv.Atoi(functional.EnvOrDefault("BACKUP_MAX_AGE_HOURS", ""))
	if err != nil || hours <= 0 {
		hours = defaultBackupMaxAgeHours
	}
	return time.Duration(hours) * time.Hour
}
