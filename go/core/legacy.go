package core

// The legacy bridge: v1's view of the caller, for handlers and db functions not
// yet ported. Each call marks a place still to port; once the last is gone,
// this file goes too. New code never calls it.

import (
	"context"

	"cgl/api/auth"
	"cgl/core/internal/identity"
	"cgl/db"
	"cgl/obj"
)

// LegacyCaller is the registered account calling an AuthRequired endpoint, as
// v1's caller(ctx) returned it. It refuses a guest nobody signed in (ErrNoCaller), an
// account still registering (ErrRegistrationIncomplete) and a member of a
// suspended organisation (ErrOrganisationSuspended).
func LegacyCaller(ctx context.Context) (*obj.User, error) {
	user, err := from(ctx).Legacy(ctx)
	switch {
	case err != nil:
		return nil, err
	case user == nil:
		return nil, auth.ErrNoCaller
	case from(ctx).Registering():
		return nil, auth.ErrRegistrationIncomplete
	case from(ctx).Suspension() != identity.NotSuspended:
		return nil, auth.ErrOrganisationSuspended
	}
	// v1 promotes admin emails on every request; until #452 moves it to
	// sign-in, the bridge keeps doing so. The core's caller of this request
	// still holds the role it was resolved with; the promotion counts from the
	// next request.
	return db.CheckAndPromoteAdmin(ctx, user), nil
}

// LegacyMaybeCaller is LegacyCaller for AuthOptional endpoints: nil and no
// error for a guest nobody signed in.
func LegacyMaybeCaller(ctx context.Context) (*obj.User, error) {
	if from(ctx).Account() == nil {
		return nil, nil
	}
	return LegacyCaller(ctx)
}
