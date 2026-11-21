// Package store is the core's low-level database access: the queries schemaf
// generates into go/db, and the hand-written access sqlc cannot express
// (transactions across queries). It decides nothing: every decision on who may
// see or change what is the core's.
package store

import (
	"context"

	"cgl/db"

	schemafdb "github.com/flocko-motion/schemaf/db"
)

// Queries are the generated queries on schemaf's connection.
func Queries() *db.Queries {
	return db.New(schemafdb.DB())
}

// InTx runs fn on the generated queries in one transaction: every write of
// fn is stored, or, when fn fails, none.
func InTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := schemafdb.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}
