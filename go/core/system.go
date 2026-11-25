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
	// sessionPurgeInterval is how often the purge runs after the first pass.
	sessionPurgeInterval = time.Hour
	// purgeBatchSize is how many sessions one purge statement deletes.
	purgeBatchSize = 500
	// defaultSignUpRetentionHours is how long an account may stay in sign-up
	// (spec, Registration): an unfinished sign-up is removed after a day.
	defaultSignUpRetentionHours = 24
)

// Janitor is the service that cleans up at start and then hourly: it purges
// sessions idle beyond the site's retention window (forget drops a purged
// session the engine still holds) and removes accounts left in sign-up
// beyond theirs. The admin sets the session window in the site's settings,
// read on every pass, so a change applies within the hour;
// SIGNUP_RETENTION_HOURS sets the sign-up window. 0 switches that purge off.
func Janitor(forget func(sessionID string)) func(context.Context) {
	return func(ctx context.Context) {
		signUps := retentionHours("SIGNUP_RETENTION_HOURS", defaultSignUpRetentionHours)
		log.Info("janitor enabled", "signup_retention", signUps, "interval", sessionPurgeInterval)
		ticker := time.NewTicker(sessionPurgeInterval)
		defer ticker.Stop()
		for {
			if sessions := sessionRetention(ctx); sessions > 0 {
				purgeExpiredSessions(ctx, sessions, forget)
			}
			if signUps > 0 {
				purgeStaleSignUps(ctx, signUps)
			}
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

// sessionRetention is the site's session window as the admin set it; 0 when
// it cannot be read, so no session goes on a failed read.
func sessionRetention(ctx context.Context) time.Duration {
	hours, err := store.Queries().V2SessionRetentionHours(ctx)
	if err != nil {
		log.Error("reading the session retention failed, no sessions purged", "error", err)
		return 0
	}
	return time.Duration(hours) * time.Hour
}

// retentionHours reads a retention window from the environment; an
// unreadable value falls back to the default rather than letting rows pile
// up.
func retentionHours(env string, defaultHours int) time.Duration {
	raw := functional.EnvOrDefault(env, strconv.Itoa(defaultHours))
	hours, err := strconv.Atoi(raw)
	if err != nil || hours < 0 {
		log.Warn("invalid retention, using default", "env", env, "value", raw, "default_hours", defaultHours)
		return time.Duration(defaultHours) * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

// purgeStaleSignUps removes the accounts left in sign-up beyond the
// retention window. An account in sign-up with a name, a role or a session
// is no unfinished sign-up: it stays, with a warning.
func purgeStaleSignUps(ctx context.Context, retention time.Duration) {
	cutoff := time.Now().Add(-retention)
	stale, err := store.Queries().V2StaleSignUps(ctx, db.V2StaleSignUpsParams{Cutoff: cutoff, MaxRows: purgeBatchSize})
	if err != nil {
		log.Error("sign-up purge failed", "error", err, "cutoff", cutoff)
		return
	}
	removed := 0
	for _, a := range stale {
		if a.HasName || a.HasRole || a.HasSession {
			log.Warn("sign-up purge keeps an account in sign-up that holds data",
				"user_id", a.ID, "name", a.HasName, "role", a.HasRole, "session", a.HasSession)
			continue
		}
		n, err := store.Queries().V2DeleteSignUp(ctx, a.ID)
		if err != nil {
			// another table still refers to the account: it holds data
			log.Warn("sign-up purge keeps an account another record refers to", "user_id", a.ID, "error", err)
			continue
		}
		removed += int(n)
	}
	if removed == 0 {
		log.Debug("sign-up purge found nothing to remove", "cutoff", cutoff)
		return
	}
	log.Info("sign-up purge completed", "removed", removed, "cutoff", cutoff)
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
