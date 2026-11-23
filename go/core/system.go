package core

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"strings"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// The core's own jobs, without a user: the hooks and services app.Wire registers.
// They act on the store directly, inside the core. No caller exists for
// them, and nothing exported hands one out: the system's rights never leave
// the core.

// Bootstrap prepares the data once the database is up: in dev mode the dev
// accounts, games and site rules, then the first admin from ADMIN_EMAILS. A
// schemaf before-serve hook: it is done before the first request.
func Bootstrap(ctx context.Context) {
	if schemafapi.DevMode() {
		preseed(ctx)
	}
	promoteAdminEmails(ctx)
}

const (
	// defaultSessionRetentionHours is how long a session survives its last
	// activity: only the session of the game a player is playing is
	// reachable, so anything idle beyond it is dead weight.
	defaultSessionRetentionHours = 48
	// sessionPurgeInterval is how often the purge runs after the first pass.
	sessionPurgeInterval = time.Hour
	// purgeBatchSize is how many sessions one purge statement deletes.
	purgeBatchSize = 500
)

// Janitor is the service that purges sessions idle beyond the retention
// window, at start and then hourly; forget drops a purged session the
// engine still holds. SESSION_RETENTION_HOURS sets the window; 0 switches
// the purge off.
func Janitor(forget func(sessionID string)) func(context.Context) {
	return func(ctx context.Context) {
		retention := sessionRetention()
		if retention <= 0 {
			log.Info("session purge disabled", "env", "SESSION_RETENTION_HOURS=0")
			return
		}
		log.Info("session purge enabled", "retention", retention, "interval", sessionPurgeInterval)
		ticker := time.NewTicker(sessionPurgeInterval)
		defer ticker.Stop()
		for {
			purgeExpiredSessions(ctx, retention, forget)
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}
}

// purgeExpiredSessions runs one purge pass, in batches, and logs it.
func purgeExpiredSessions(ctx context.Context, retention time.Duration, forget func(string)) {
	cutoff := time.Now().Add(-retention)
	total := 0
	for {
		ids, err := store.Queries().V2PurgeExpiredSessions(ctx, db.V2PurgeExpiredSessionsParams{ModifiedAt: cutoff, Limit: purgeBatchSize})
		for _, id := range ids {
			forget(id.String())
		}
		total += len(ids)
		if err != nil {
			log.Error("session purge failed", "error", err, "cutoff", cutoff, "purged", total)
			return
		}
		if len(ids) < purgeBatchSize || ctx.Err() != nil {
			break
		}
	}
	if total == 0 {
		log.Debug("session purge found nothing to delete", "cutoff", cutoff)
		return
	}
	log.Info("session purge completed", "purged", total, "cutoff", cutoff)
}

// sessionRetention reads the retention window from the environment; an
// unreadable value falls back to the default rather than letting sessions
// pile up.
func sessionRetention() time.Duration {
	raw := functional.EnvOrDefault("SESSION_RETENTION_HOURS", strconv.Itoa(defaultSessionRetentionHours))
	hours, err := strconv.Atoi(raw)
	if err != nil || hours < 0 {
		log.Warn("invalid SESSION_RETENTION_HOURS, using default", "value", raw, "default_hours", defaultSessionRetentionHours)
		return defaultSessionRetentionHours * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

// The first admin of an empty deployment comes from ADMIN_EMAILS: an
// account with a listed email becomes admin, at start, at registration or
// at sign-in, while no admin exists yet. Once one does, the list elevates
// nobody: admins make admins.

// isAdminEmail says whether email is in ADMIN_EMAILS.
func isAdminEmail(email string) bool {
	for _, listed := range strings.Split(os.Getenv("ADMIN_EMAILS"), ",") {
		if listed = strings.TrimSpace(listed); listed != "" && listed == strings.TrimSpace(email) {
			return true
		}
	}
	return false
}

// adminBootstrapOpen says whether ADMIN_EMAILS may still make an admin: while
// none exists. A failed count closes it.
func adminBootstrapOpen(ctx context.Context) bool {
	count, err := store.Queries().V2CountAdmins(ctx)
	if err != nil {
		log.Warn("failed to count admins, refusing admin bootstrap", "error", err)
		return false
	}
	return count == 0
}

// makeFirstAdmin makes an individual with an admin email the admin, while
// the bootstrap is open; anyone else keeps their role.
func makeFirstAdmin(ctx context.Context, user uuid.UUID, email *string) bool {
	if email == nil || !isAdminEmail(*email) || !adminBootstrapOpen(ctx) {
		return false
	}
	roles, err := store.Queries().V2AccountRoles(ctx, user)
	if err != nil {
		log.Warn("admin bootstrap: failed to read roles", "user_id", user, "error", err)
		return false
	}
	for _, r := range roles {
		if r != string(obj.RoleIndividual) {
			log.Warn("admin bootstrap: only an individual becomes admin", "user_id", user, "role", r)
			return false
		}
	}
	err = store.InTx(ctx, func(q *db.Queries) error {
		if err := q.V2DeleteUserRoles(ctx, user); err != nil {
			return err
		}
		_, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{UserID: user, Role: string(obj.RoleAdmin)})
		return err
	})
	if err != nil {
		log.Warn("admin bootstrap: failed to make the admin", "user_id", user, "error", err)
		return false
	}
	log.Info("admin bootstrap: first admin made", "user_id", user)
	return true
}

// promoteAdminEmails makes the first admin from ADMIN_EMAILS at start.
func promoteAdminEmails(ctx context.Context) {
	if os.Getenv("ADMIN_EMAILS") == "" || !adminBootstrapOpen(ctx) {
		return
	}
	for _, email := range strings.Split(os.Getenv("ADMIN_EMAILS"), ",") {
		if email = strings.TrimSpace(email); email == "" {
			continue
		}
		u, err := store.Queries().V2GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
		if err != nil {
			continue
		}
		if makeFirstAdmin(ctx, u.ID, &email) {
			return
		}
	}
}

// assignInitialRole gives a new account its first role: admin for an admin
// email while no admin exists, otherwise individual.
func assignInitialRole(ctx context.Context, user uuid.UUID, email *string) error {
	if _, err := store.Queries().V2CreateUserRole(ctx, db.V2CreateUserRoleParams{UserID: user, Role: string(obj.RoleIndividual)}); err != nil {
		return err
	}
	makeFirstAdmin(ctx, user, email)
	return nil
}
