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
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// Caller is who is asking. Its fields are unexported: outside this package
// nobody can construct or alter one, and a user loaded by a query is never a
// Caller. Its facts are loaded once per request; it lives for that request
// and is never cached beyond it.
type Caller struct {
	account         *uuid.UUID // nil: anonymous; owns nothing
	role            rules.Role
	organisation    *uuid.UUID
	currentWorkshop *uuid.UUID // participant: fixed; others: workshop mode
	registering     bool       // signed in, registration not yet complete

	// the current workshop's settings
	workshopShowsPublic       bool
	workshopShowsParticipants bool
	workshopAllowsSharing     bool

	legacy *obj.User // v1's view of the caller, loaded by Legacy on first use
}

// anonymous is the caller without an account: it holds only Everyone's grants.
var anonymous = &Caller{}

// Anonymous is the caller of a request nobody signed in to, and of a context
// nobody resolved: no rights by accident.
func Anonymous() *Caller { return anonymous }

// New is the caller of the request whose admission adm is. It is the only way
// to an account's caller, and works only inside the OnRequest hook: adm panics
// once the hook has returned. One query loads everything (V2ReadCaller).
//
// It refuses what is refused on every route: a subject that names no account,
// a deleted account, a participant who left their workshop or whose workshop
// is inactive. An account still registering is a caller; the bridge refuses
// it where v1 did.
func New(ctx context.Context, adm *schemafapi.Admission) (*Caller, error) {
	subject, ok := adm.Subject()
	if !ok {
		return anonymous, nil
	}
	id, err := uuid.Parse(subject)
	if err != nil {
		return nil, auth.ErrNoAccount
	}
	row, err := store.Queries().V2ReadCaller(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrNoAccount
	}
	if err != nil {
		return nil, err
	}

	role := rules.Role(row.Role.String)
	switch {
	case row.Deleted:
		return nil, auth.ErrNoAccount
	case row.ParticipantAccount && role != rules.Participant:
		return nil, auth.ErrNoAccount // left their workshop
	case role == rules.Participant && !row.ParticipantWorkshopActive:
		return nil, auth.ErrWorkshopInactive
	}

	c := &Caller{
		account:      &row.ID,
		role:         role,
		registering:  row.Registering,
		organisation: nullable(row.InstitutionID),

		workshopShowsPublic:       row.WorkshopShowsPublic,
		workshopShowsParticipants: row.WorkshopShowsParticipants,
		workshopAllowsSharing:     row.WorkshopAllowsSharing,
	}
	if role == rules.Participant {
		c.currentWorkshop = nullable(row.WorkshopID)
	} else {
		c.currentWorkshop = nullable(row.ActiveWorkshopID)
	}
	return c, nil
}

// Account is the caller's account, or nil for anonymous.
func (c *Caller) Account() *uuid.UUID { return c.account }

// Role is the caller's role; empty for anonymous.
func (c *Caller) Role() rules.Role { return c.role }

// Organisation is the caller's organisation, if any.
func (c *Caller) Organisation() *uuid.UUID { return c.organisation }

// CurrentWorkshop is the workshop the caller is in, if any.
func (c *Caller) CurrentWorkshop() *uuid.UUID { return c.currentWorkshop }

// Contexts are the situations the caller is in, which decide which of their
// grants hold.
func (c *Caller) Contexts() []rules.Context {
	if c.currentWorkshop == nil {
		return []rules.Context{rules.Always, rules.OutsideWorkshop}
	}
	contexts := []rules.Context{rules.Always, rules.InWorkshop}
	if c.workshopShowsPublic {
		contexts = append(contexts, rules.PublicShown)
	}
	if c.workshopShowsParticipants {
		contexts = append(contexts, rules.ParticipantsShown)
	}
	if c.workshopAllowsSharing {
		contexts = append(contexts, rules.SharingAllowed)
	}
	return contexts
}

// Registering says the caller signed in but has not completed registration.
func (c *Caller) Registering() bool { return c.registering }

// Legacy is v1's view of the caller, for the legacy bridge only; nil for
// anonymous. It loads on first use, so only code not yet ported pays for it.
func (c *Caller) Legacy(ctx context.Context) (*obj.User, error) {
	if c.account == nil {
		return nil, nil
	}
	if c.legacy == nil {
		user, err := db.GetUserByID(ctx, *c.account)
		if err != nil {
			return nil, err
		}
		c.legacy = user
	}
	return c.legacy, nil
}

func nullable(id uuid.NullUUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	return &id.UUID
}
