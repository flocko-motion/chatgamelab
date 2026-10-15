// package: routes / server status HTTP handler
// type:    logic
// job:     handles the public health/status endpoint reporting server state, uptime and backup health
// limits:  no route registration (-> router.go); build info (-> version.go); backups are run by docker/db/backup.sh, never here
package routes

import (
	"net/http"
	"strconv"
	"time"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/functional"
	"cgl/obj"
)

var serverStartTime = time.Now()

// Backup health values reported in StatusResponse.Backup.
const (
	backupDisabled = "disabled" // BACKUP_ENABLED is not "true"
	backupOk       = "ok"       // newest run succeeded and is recent enough
	backupFailed   = "failed"   // newest run failed
	backupStale    = "stale"    // newest run succeeded but is older than the allowed age
	backupUnknown  = "unknown"  // enabled, but no run was ever recorded (e.g. the host cron entry is missing)
)

// defaultBackupMaxAgeHours tolerates a daily backup plus a missed slot before reporting stale.
const defaultBackupMaxAgeHours = 48

// StatusResponse reports the server's health status, uptime and backup health.
type StatusResponse struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Backup    string `json:"backup"`              // disabled | ok | failed | stale | unknown
	BackupAge string `json:"backupAge,omitempty"` // time since the newest recorded run
}

// GetStatus godoc
//
//	@Summary		Get server status
//	@Description	Returns the current server status, uptime and database backup health
//	@Tags			status
//	@Produce		json
//	@Success		200	{object}	StatusResponse
//	@Router			/status [get]
func GetStatus(w http.ResponseWriter, r *http.Request) {
	backup, age := backupHealth(r)

	resp := StatusResponse{
		Status: "running",
		Uptime: functional.HumanizeDuration(time.Since(serverStartTime)),
		Backup: backup,
	}
	if age != nil {
		resp.BackupAge = functional.HumanizeDuration(*age)
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

// backupHealth reads the newest backup logbook entry and grades it. It never reports
// an error: this endpoint is the container liveness probe, so an unreadable logbook
// degrades to "unknown" rather than failing the whole health check.
func backupHealth(r *http.Request) (string, *time.Duration) {
	if functional.EnvOrDefault("BACKUP_ENABLED", "false") != "true" {
		return backupDisabled, nil
	}

	latest, err := db.GetLatestBackup(r.Context())
	if err != nil {
		return backupUnknown, nil
	}

	return gradeBackup(latest, backupMaxAge())
}

// gradeBackup turns the newest logbook entry into a health value. A nil entry means
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

func backupMaxAge() time.Duration {
	hours, err := strconv.Atoi(functional.EnvOrDefault("BACKUP_MAX_AGE_HOURS", ""))
	if err != nil || hours <= 0 {
		hours = defaultBackupMaxAgeHours
	}
	return time.Duration(hours) * time.Hour
}
