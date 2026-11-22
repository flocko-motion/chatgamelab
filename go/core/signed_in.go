package core

import (
	"context"

	"cgl/api/auth"
	"cgl/core/internal/identity"
)

// RequireSignedIn refuses a caller who may use nothing but the public site:
// a guest nobody signed in (ErrNoCaller), an account still registering
// (ErrRegistrationIncomplete), and a member of a suspended organisation
// (ErrOrganisationSuspended). For endpoints whose answer is the same for
// every signed-in caller, such as the engine's platforms.
func RequireSignedIn(ctx context.Context) error {
	c := from(ctx)
	switch {
	case c.Account() == nil:
		return auth.ErrNoCaller
	case c.Registering():
		return auth.ErrRegistrationIncomplete
	case c.Suspension() != identity.NotSuspended:
		return auth.ErrOrganisationSuspended
	}
	return nil
}
