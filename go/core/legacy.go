package core

// The legacy bridge: v1's view of the caller, for handlers and db functions not
// yet ported. Each call marks a place still to port; once the last is gone,
// this file goes too. New code never calls it.

import (
	"context"

	"cgl/api/auth"
	"cgl/db"
	"cgl/obj"
)

// LegacyCaller is the registered account calling an AuthRequired endpoint, as
// v1's caller(ctx) returned it. It refuses anonymous (ErrNoCaller) and an
// account still registering (ErrRegistrationIncomplete).
func LegacyCaller(ctx context.Context) (*obj.User, error) {
	user, err := LegacyRegistering(ctx)
	if err != nil {
		return nil, err
	}
	if from(ctx).Registering() {
		return nil, auth.ErrRegistrationIncomplete
	}
	return user, nil
}

// LegacyMaybeCaller is LegacyCaller for AuthOptional endpoints: nil and no
// error for anonymous.
func LegacyMaybeCaller(ctx context.Context) (*obj.User, error) {
	if from(ctx).Account() == nil {
		return nil, nil
	}
	return LegacyCaller(ctx)
}

// LegacyRegistering is the account calling, registered or still registering:
// for completing registration and reporting who is calling. It refuses
// anonymous (ErrNoCaller).
func LegacyRegistering(ctx context.Context) (*obj.User, error) {
	user, err := LegacyMaybeRegistering(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, auth.ErrNoCaller
	}
	return user, nil
}

// LegacyMaybeRegistering is LegacyRegistering with nil and no error for
// anonymous.
func LegacyMaybeRegistering(ctx context.Context) (*obj.User, error) {
	user, err := from(ctx).Legacy(ctx)
	if err != nil || user == nil {
		return nil, err
	}
	// v1 promotes admin emails on every request; until #452 moves it to sign-in,
	// the bridge keeps doing so. The core's caller of this request still holds
	// the role it was resolved with; the promotion counts from the next request.
	return db.CheckAndPromoteAdmin(ctx, user), nil
}
