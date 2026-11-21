package core

import (
	"context"
	"testing"

	"cgl/core/internal/identity"
	"cgl/core/internal/rules"
)

// A context nobody resolved holds the anonymous caller, a guest: no rights
// by accident, only Everyone's and the Guest role's (which has none of its
// own), never SignedIn's.
func TestUnresolvedContextIsAnonymous(t *testing.T) {
	c := from(context.Background())
	if c != identity.Guest() {
		t.Fatalf("unresolved context has caller %+v, want anonymous", c)
	}
	got := roles(c)
	if len(got) != 2 || got[0] != rules.Everyone || got[1] != rules.Guest {
		t.Fatalf("anonymous roles = %v, want [everyone guest]", got)
	}
	if len(rules.Rules[rules.Guest].For(c.Contexts())) != 0 {
		t.Fatalf("the Guest role grants something of its own")
	}
}

// The caller put into a context is the one read out of it.
func TestCallerTravelsInContext(t *testing.T) {
	want := identity.Guest()
	got := from(withCaller(context.Background(), want))
	if got != want {
		t.Fatalf("from(withCaller(c)) = %p, want %p", got, want)
	}
}
