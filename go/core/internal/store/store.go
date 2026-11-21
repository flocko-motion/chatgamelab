// Package store is the core's low-level database access: the queries schemaf
// generates into go/db, and the hand-written access sqlc cannot express
// (transactions across queries). It decides nothing: every decision on who may
// see or change what is the core's.
package store

import (
	"cgl/db"

	schemafdb "github.com/flocko-motion/schemaf/db"
)

// Queries are the generated queries on schemaf's connection.
func Queries() *db.Queries {
	return db.New(schemafdb.DB())
}
