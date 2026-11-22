// Package identity is who is asking: the caller of one request. Its only
// constructor demands the admission schemaf hands to the OnRequest hook, so a
// caller can only be built there, from the subject schemaf authenticated in
// this request; no code, the core included, can build one by hand.
package identity

import (
	"context"
	"database/sql"
	"errors"

	"cgl/api/auth"
	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// Caller is who is asking. Its fields are unexported: outside this package
// nobody can construct or alter one, and a user loaded by a query is never a
// Caller. Its facts are loaded once per request; it lives for that request
// and is never cached beyond it.
type Caller struct {
	account         *uuid.UUID // nil: a guest nobody signed in; owns nothing
	role            rules.Role
	organisation    *uuid.UUID
	currentWorkshop *uuid.UUID // a guest: their own, fixed; others: workshop mode
	registering     bool       // signed in, registration not yet complete
	suspension      Suspension // their organisation's, if it is suspended

	// the current workshop's settings
	workshopShowsPublic   bool
	workshopShowsOthers   bool
	workshopAllowsSharing bool
}

// guest is the caller nobody signed in: a guest without an account yet,
// holding only Everyone's
// grants.
var guest = &Caller{role: rules.Guest}

// Guest is the caller of a request nobody signed in to, and of a context
// nobody resolved: no rights by accident.
func Guest() *Caller { return guest }

// New is the caller of the request whose admission adm is. It is the only way
// to an account's caller, and works only inside the OnRequest hook: adm panics
// once the hook has returned. One query loads everything (V2ReadCaller).
//
// It refuses what is refused on every route: a subject that names no account,
// a deleted account, a guest who left their workshop or whose workshop
// is inactive. An account still registering is a caller; the bridge refuses
// it where v1 did.
func New(ctx context.Context, adm *schemafapi.Admission) (*Caller, error) {
	subject, ok := adm.Subject()
	if !ok {
		return guest, nil
	}
	id, err := uuid.Parse(subject)
	if err != nil {
		return nil, auth.ErrNoAccount
	}
	declaration := legal.CurrentDeclaration()
	row, err := store.Queries().V2ReadCaller(ctx, db.V2ReadCallerParams{
		ID:                 id,
		DeclarationVersion: declaration.Version,
		DeclarationHash:    declaration.Hash,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrNoAccount
	}
	if err != nil {
		return nil, err
	}

	guest := row.Role.String == dbGuest // a guest's account: always in its workshop
	switch {
	case row.Deleted:
		return nil, auth.ErrNoAccount
	case row.GuestAccount && !guest:
		return nil, auth.ErrNoAccount // left their workshop
	case guest && !row.GuestWorkshopActive:
		return nil, auth.ErrWorkshopInactive
	}

	c := &Caller{
		account:      &row.ID,
		registering:  row.Registering,
		suspension:   suspensionOf(row.SuspendedByAdmin, row.ConsentMissing),
		organisation: nullable(row.InstitutionID),

		workshopShowsPublic:   row.WorkshopShowsPublic,
		workshopShowsOthers:   row.WorkshopShowsOthers,
		workshopAllowsSharing: row.WorkshopAllowsSharing,
	}
	if guest {
		// a guest, always in their own workshop: no workshop mode to leave
		c.role = rules.Guest
		c.currentWorkshop = nullable(row.WorkshopID)
	} else {
		c.role = rules.Role(row.Role.String)
		c.currentWorkshop = nullable(row.ActiveWorkshopID)
	}
	return c, nil
}

// dbGuest is how the database stores the role of a guest in a workshop,
// bound to that workshop.
const dbGuest = "guest"

// Account is the caller's account, or nil for a guest nobody signed in.
func (c *Caller) Account() *uuid.UUID { return c.account }

// Role is the caller's role: Guest for a caller nobody signed in.
func (c *Caller) Role() rules.Role { return c.role }

// Organisation is the caller's organisation, if any.
func (c *Caller) Organisation() *uuid.UUID { return c.organisation }

// CurrentWorkshop is the workshop the caller is in, if any.
func (c *Caller) CurrentWorkshop() *uuid.UUID { return c.currentWorkshop }

// Contexts are the situations the caller is in, which decide which of their
// grants hold.
func (c *Caller) Contexts() []rules.Context {
	if c.suspension != NotSuspended {
		return []rules.Context{rules.Suspended}
	}
	if c.registering {
		return []rules.Context{rules.Always, rules.OutsideWorkshop, rules.Registering}
	}
	if c.currentWorkshop == nil {
		return []rules.Context{rules.Always, rules.OutsideWorkshop}
	}
	contexts := []rules.Context{rules.Always, rules.InWorkshop}
	if c.workshopShowsPublic {
		contexts = append(contexts, rules.PublicShown)
	}
	if c.workshopShowsOthers {
		contexts = append(contexts, rules.OthersShown)
	}
	if c.workshopAllowsSharing {
		contexts = append(contexts, rules.SharingAllowed)
	}
	return contexts
}

// Registering says the caller signed in but has not completed registration.
func (c *Caller) Registering() bool { return c.registering }

// Suspension is why the caller's organisation is suspended, if it is: as a
// head, staff or a guest in one of its workshops, the caller then holds only
// what Suspended grants.
func (c *Caller) Suspension() Suspension { return c.suspension }

// Suspension is why an organisation is suspended.
type Suspension string

const (
	NotSuspended       Suspension = ""
	SuspendedByAdmin   Suspension = "admin"   // an admin suspended it; only an admin lifts it
	SuspendedNoConsent Suspension = "consent" // its newest consent does not cover the current declaration
)

// suspensionOf is the suspension of both causes; an admin's goes first, as no
// consent lifts it.
func suspensionOf(byAdmin, consentMissing bool) Suspension {
	switch {
	case byAdmin:
		return SuspendedByAdmin
	case consentMissing:
		return SuspendedNoConsent
	}
	return NotSuspended
}

func nullable(id uuid.NullUUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	return &id.UUID
}
