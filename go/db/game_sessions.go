// package: db / database access and repository layer
// type:    data
// job:     purges expired sessions for the janitor.
// limits:  v1's janitor only; it moves into the core with the system actor (#344).
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// purgeBatchSize caps how many sessions one DELETE statement touches, keeping
// lock duration and WAL volume bounded on installations with a large backlog.
const purgeBatchSize = 200

// PurgeExpiredSessions deletes every session idle since cutoff, with its
// events, and answers their ids.
func PurgeExpiredSessions(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	var purged []uuid.UUID
	for {
		ids, err := queries().PurgeExpiredGameSessions(ctx, PurgeExpiredGameSessionsParams{
			ModifiedAt: cutoff,
			Limit:      purgeBatchSize,
		})
		if err != nil {
			return purged, fmt.Errorf("failed to purge expired sessions: %w", err)
		}
		purged = append(purged, ids...)
		if len(ids) < purgeBatchSize {
			return purged, nil
		}
		if err := ctx.Err(); err != nil {
			return purged, err
		}
	}
}
