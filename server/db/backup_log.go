// package: db / database access and repository layer
// type:    data
// job:     read the newest entry of the database backup logbook written by docker/db/backup.sh.
// limits:  read-only; rows are written by the backup script inside the db container, never by the server.
package db

import (
	"context"
	"database/sql"
	"errors"

	"cgl/obj"
)

// GetLatestBackup returns the most recent backup logbook entry, or nil if no
// backup has ever been recorded.
func GetLatestBackup(ctx context.Context) (*obj.BackupLog, error) {
	row, err := queries().GetLatestBackup(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &obj.BackupLog{
		ID:         row.ID,
		StartedAt:  row.StartedAt,
		FinishedAt: row.FinishedAt,
		Success:    row.Success,
		Filename:   sqlNullStringToMaybeString(row.Filename),
		SizeBytes:  sqlNullInt64ToMaybeInt64(row.SizeBytes),
		Error:      sqlNullStringToMaybeString(row.Error),
	}, nil
}
