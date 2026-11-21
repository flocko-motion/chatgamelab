// package: db / database access and repository layer
// type:    wiring
// job:     hand the hand-written functions schemaf's connection and the generated queries.
// limits:  does not define SQL queries (-> db/queries) or per-entity repository logic.
package db

import (
	"context"
	"database/sql"
	"errors"

	"cgl/obj"

	"github.com/flocko-motion/schemaf/constants"
	schemafdb "github.com/flocko-motion/schemaf/db"
)

// sqlDb is schemaf's connection pool.
func sqlDb() *sql.DB {
	return schemafdb.DB()
}

// queries returns the generated queries on schemaf's connection.
func queries() *Queries {
	return New(sqlDb())
}

// Open connects schemaf's pool to dsn and applies the project's migrations.
// It is for the v1 server, which runs outside schemaf's app.
func Open(ctx context.Context, dsn string) error {
	if err := schemafdb.Init(dsn); err != nil {
		return err
	}
	return schemafdb.RunSet(ctx, sqlDb(), schemafdb.MigrationSet{Prefix: constants.ProjectName(), Files: Provider()})
}

// SchemaVersion is the newest of the project's migrations applied to the database.
func SchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := sqlDb().QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schemaf_migrations WHERE prefix = $1`,
		constants.ProjectName()).Scan(&version)
	return version, err
}

func sqlNullStringToMaybeString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func sqlNullBoolToMaybeBool(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

func stringToRole(s string) (obj.Role, error) {
	switch s {
	case string(obj.RoleAdmin):
		return obj.RoleAdmin, nil
	case string(obj.RoleHead):
		return obj.RoleHead, nil
	case string(obj.RoleStaff):
		return obj.RoleStaff, nil
	case string(obj.RoleGuest):
		return obj.RoleGuest, nil
	case string(obj.RoleIndividual):
		return obj.RoleIndividual, nil
	default:
		return obj.Role("invalid:" + s), errors.New("invalid role")
	}
}
