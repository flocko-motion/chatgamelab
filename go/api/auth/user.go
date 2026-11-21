// package: auth / who is calling
// type:    logic
// job:     the errors that refuse a request's caller; resolving the caller is the secure core's (core.Resolve)
// limits:  schemaf authenticates the request and owns sessions; permissions are the core's
package auth

import (
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
	// ErrWorkshopInactive: the participant's workshop is inactive.
	ErrWorkshopInactive = schemafapi.Error(schemafapi.ErrForbidden, "auth_workshop_inactive", "workshop is inactive")
	// ErrNoAccount: the subject names no usable account: unknown, deleted, or
	// a participant who left their workshop.
	ErrNoAccount = fmt.Errorf("no such account: %w", schemafapi.ErrForbidden)
)
