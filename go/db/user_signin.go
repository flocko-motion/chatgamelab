// package: db / database access and repository layer
// type:    data
// job:     the account behind a sign-in: created at an Auth0 subject's first sign-in, completed by registration, and a participant's standing
// limits:  no sessions or cookies (-> schemaf, api/auth); input rules live in api/auth
package db

import (
	"cgl/obj"
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

// CompleteRegistration gives an account its name, email, age group and
// language, and its first role. An account registers once: a second time is a
// conflict, as is a name another account has.
func CompleteRegistration(ctx context.Context, id uuid.UUID, name string, email *string, ageGroup string, language string) (*obj.User, error) {
	taken, err := IsNameTakenByOther(ctx, name, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, obj.ErrConflict("the name is taken")
	}
	if email != nil {
		taken, err := IsEmailTakenByOther(ctx, *email, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, obj.ErrConflict("the email belongs to another account")
		}
	}

	n, err := queries().CompleteRegistration(ctx, CompleteRegistrationParams{
		ID:       id,
		Name:     userName(name),
		Email:    stringPtrToNullString(email),
		AgeGroup: sql.NullString{String: ageGroup, Valid: true},
		Language: language,
	})
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		// A concurrent registration took the name or email first.
		return nil, obj.ErrConflict("the name or email is taken")
	}
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, obj.ErrConflict("the account is registered already")
	}

	assignInitialRole(ctx, id, email)
	return GetUserByID(ctx, id)
}

// ParticipantAccess reports whether an account may act: a *ParticipantAuthError
// for a participant account that lost its role or whose workshop is inactive,
// nil for every other account.
func ParticipantAccess(ctx context.Context, id uuid.UUID) error {
	status, err := queries().GetParticipantStatus(ctx, id)
	if err != nil {
		return err
	}
	switch {
	case !status.HasToken:
		return nil
	case !status.HasRole:
		return &ParticipantAuthError{Code: "invalid_token", Message: "no longer a workshop participant"}
	case !status.WorkshopActive:
		return &ParticipantAuthError{Code: "workshop_inactive", Message: "Workshop is inactive"}
	}
	return nil
}
