// package: db / database access and repository layer
// type:    data
// job:     deletes game sessions and their messages once they exceed the retention window.
// limits:  no scheduling (-> api/janitor.go); no permission checks, since retention applies to every owner alike.
package db

import (
	db "cgl/db/sqlc"
	"cgl/log"
	"context"
	"fmt"
	"time"
)

// purgeBatchSize caps how many sessions one DELETE statement touches, keeping lock
// duration and WAL volume bounded on installations with a large backlog.
const purgeBatchSize = 200

// PurgeExpiredSessions deletes every session whose last activity predates cutoff,
// together with its messages, and returns the number of sessions deleted.
func PurgeExpiredSessions(ctx context.Context, cutoff time.Time) (int64, error) {
	var total int64
	for {
		purged, err := queries().PurgeExpiredGameSessions(ctx, db.PurgeExpiredGameSessionsParams{
			ModifiedAt: cutoff,
			Limit:      purgeBatchSize,
		})
		if err != nil {
			return total, fmt.Errorf("failed to purge expired sessions: %w", err)
		}

		total += purged
		if purged < purgeBatchSize {
			return total, nil
		}

		log.Debug("purged a batch of expired sessions", "batch", purged, "total", total)

		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
	}
}

// CountExpiredSessions reports how many sessions predate cutoff without deleting them.
func CountExpiredSessions(ctx context.Context, cutoff time.Time) (int64, error) {
	count, err := queries().CountExpiredGameSessions(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to count expired sessions: %w", err)
	}
	return count, nil
}
