// package: auth / who is calling
// type:    logic
// job:     the errors that refuse a sign-in or a request's caller; signing in and resolving the caller are the secure core's (core.OnLogin, core.Resolve)
// limits:  schemaf authenticates the request and owns sessions; permissions are the core's
package auth

import (
	"errors"
	"fmt"

	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

var (
	// ErrNoCaller: the request carries no signed-in caller. Endpoints that
	// resolve the user require auth, so schemaf answers 401 before this.
	ErrNoCaller = fmt.Errorf("not signed in: %w", schemafapi.ErrUnauthorized)
	// ErrRegistrationIncomplete: the account has no name yet. Every endpoint
	// refuses it, except completing registration, /api/user/me and logout.
	ErrRegistrationIncomplete = schemafapi.Error(schemafapi.ErrForbidden, obj.ErrCodeUserNotRegistered, "registration incomplete")
	// ErrOrganisationSuspended: the caller's organisation is suspended, by an
	// admin or for want of a consent to the declaration. Every endpoint
	// refuses its members, except /api/users/me, which reports it, logout,
	// and the consent its head may give.
	ErrOrganisationSuspended = schemafapi.Error(schemafapi.ErrForbidden, obj.ErrCodeOrganisationSuspended, "your organisation is suspended")
	// ErrWorkshopInactive: the participant's workshop is inactive.
	ErrWorkshopInactive = schemafapi.Error(schemafapi.ErrForbidden, "auth_workshop_inactive", "workshop is inactive")
	// ErrInvalidGuestToken: no guest signs in with this code.
	ErrInvalidGuestToken = errors.New("invalid or expired token")
	// ErrNoGuestToken: the request names no token.
	ErrNoGuestToken = fmt.Errorf("a token is required: %w", schemafapi.ErrBadRequest)
	// ErrNoAccount: the subject names no usable account: unknown, deleted, or
	// a participant who left their workshop.
	ErrNoAccount = fmt.Errorf("no such account: %w", schemafapi.ErrForbidden)
)
