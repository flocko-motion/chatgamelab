// package: db / database access and repository layer
// type:    wiring
// job:     hand the hand-written functions schemaf's connection and the generated queries.
// limits:  does not define SQL queries (-> db/queries) or per-entity repository logic.
package db

import (
	"database/sql"
	"errors"

	"cgl/obj"

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
