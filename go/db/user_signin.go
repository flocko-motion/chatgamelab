// package: db / database access and repository layer
// type:    data
// job:     the account behind a sign-in: created at an Auth0 subject's first sign-in, and a guest's standing
// limits:  no sessions or cookies (-> schemaf, api/auth); registering is the core's (core.Register)
package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// AccountForAuth0 returns the account of an Auth0 subject, creating it on the
// subject's first sign-in. A new account has no name: its registration is
// incomplete.
func AccountForAuth0(ctx context.Context, auth0ID string) (uuid.UUID, error) {
	sub := sql.NullString{String: auth0ID, Valid: true}
	id, err := queries().GetUserIDByAuth0ID(ctx, sub)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	id, err = queries().CreateAccountForAuth0(ctx, sub)
	if errors.Is(err, sql.ErrNoRows) {
		// A concurrent first sign-in created it.
		return queries().GetUserIDByAuth0ID(ctx, sub)
	}
	return id, err
}
