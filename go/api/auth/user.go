// package: auth / who is calling
// type:    logic
// job:     resolves schemaf's subject to the account behind a request, and holds the registration lock
// limits:  schemaf authenticates the request and owns sessions; permissions stay in the handlers and db
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
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

// User is the registered account calling the request. It refuses an account
// whose registration is incomplete (ErrRegistrationIncomplete).
func User(ctx context.Context) (*obj.User, error) {
	user, err := Registering(ctx)
	if err != nil {
		return nil, err
	}
	if user.Name == "" {
		return nil, ErrRegistrationIncomplete
	}
	return user, nil
}

// MaybeUser is User for endpoints that also serve anonymous callers: nil and
// no error without a signed-in caller.
func MaybeUser(ctx context.Context) (*obj.User, error) {
	if _, ok := schemafapi.Subject(ctx); !ok {
		return nil, nil
	}
	return User(ctx)
}

// Registering is the account calling the request, registered or still
// registering. Endpoints use User; completing registration uses Registering.
func Registering(ctx context.Context) (*obj.User, error) {
	sub, ok := schemafapi.Subject(ctx)
	if !ok {
		return nil, ErrNoCaller
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return nil, ErrNoAccount
	}
	user, err := db.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoAccount
	}
	if err != nil {
		return nil, err
	}
	if user.DeletedAt != nil && !user.DeletedAt.IsZero() {
		return nil, ErrNoAccount
	}
	if err := participantAccess(ctx, id); err != nil {
		return nil, err
	}
	return db.CheckAndPromoteAdmin(ctx, user), nil
}

// participantAccess maps db.ParticipantAccess to the resolver's errors.
func participantAccess(ctx context.Context, id uuid.UUID) error {
	err := db.ParticipantAccess(ctx, id)
	var perr *db.ParticipantAuthError
	if errors.As(err, &perr) {
		if perr.Code == "workshop_inactive" {
			return ErrWorkshopInactive
		}
		return ErrNoAccount
	}
	return err
}
