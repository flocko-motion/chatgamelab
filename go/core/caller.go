// Package core is the secure core: the one place that decides what a caller
// may see and change. Its exported types are all the rest of the code gets;
// how it decides lives in core/internal. DRAFT: a sketch to settle the shape.
//
// Every function of core works for the caller in its ctx, without exception:
// the caller is resolved once per request by a hook and travels in the ctx,
// so no code hands a caller around, and none can slip in another. Names follow
// the actions of the grant table: ListGameCards is {List, GameCard},
// ReadGameCard is {Read, GameCard}.
package core

import (
	"context"

	"cgl/core/internal/access"
	"cgl/core/internal/identity"
	"cgl/core/internal/rules"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// Resolve is the OnRequest hook: it resolves the request's caller once and
// puts it into the context every handler gets. It refuses what is refused on
// every route (see identity.New); everything else the core decides later.
//
//	app.OnRequest(core.Resolve)
func Resolve(ctx context.Context, adm *schemafapi.Admission) (context.Context, error) {
	c, err := identity.New(ctx, adm)
	if err != nil {
		return nil, err
	}
	return withCaller(ctx, c), nil
}

// ctxKeyCaller is the context key for the caller; unexported, so only the core
// can put a caller into a context or read one out.
type ctxKeyCaller struct{}

// withCaller is ctx carrying c.
func withCaller(ctx context.Context, c *identity.Caller) context.Context {
	return context.WithValue(ctx, ctxKeyCaller{}, c)
}

// from is the caller in ctx, put there by Resolve. Without one, the caller is
// a guest nobody signed in: a context nobody resolved holds no rights.
func from(ctx context.Context) *identity.Caller {
	if c, ok := ctx.Value(ctxKeyCaller{}).(*identity.Caller); ok && c != nil {
		return c
	}
	return identity.Guest()
}

// roles are the grant lists that apply to c: Everyone, their role (a caller
// nobody signed in is a guest), and SignedIn for an account that completed
// registration.
func roles(c *identity.Caller) []rules.Role {
	rs := []rules.Role{rules.Everyone, c.Role()}
	if c.Account() != nil && !c.Registering() {
		rs = append(rs, rules.SignedIn)
	}
	return rs
}

// granted says whether the caller in ctx holds the grant {action, entity,
// relation}: a line of the grant table under one of the caller's roles, in a
// context they are in.
func granted(ctx context.Context, action rules.Action, entity rules.Entity, relation rules.Relation) bool {
	c := from(ctx)
	return access.Relations(roles(c), c.Contexts(), action, entity)[relation]
}

// account is the caller's account as a query parameter: NULL for a guest nobody signed in,
// so "own" matches nothing.
func account(ctx context.Context) uuid.NullUUID {
	return nullUUID(from(ctx).Account())
}

// currentWorkshop is the workshop the caller is in, as a query parameter: NULL
// outside any workshop, so "of current workshop" matches nothing.
func currentWorkshop(ctx context.Context) uuid.NullUUID {
	return nullUUID(from(ctx).CurrentWorkshop())
}

// ownOrganisation is the caller's organisation as a query parameter: NULL
// without one, so "of own organisation" matches nothing.
func ownOrganisation(ctx context.Context) uuid.NullUUID {
	return nullUUID(from(ctx).Organisation())
}
