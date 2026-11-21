package core

import (
	"context"
	"testing"

	"cgl/core/internal/identity"
	"cgl/core/internal/rules"
)

// A context nobody resolved holds the anonymous caller: no rights by accident.
func TestUnresolvedContextIsAnonymous(t *testing.T) {
	c := from(context.Background())
	if c != identity.Anonymous() {
		t.Fatalf("unresolved context has caller %+v, want anonymous", c)
	}
	if got := roles(c); len(got) != 1 || got[0] != rules.Everyone {
		t.Fatalf("anonymous roles = %v, want [everyone]", got)
	}
}

// The caller put into a context is the one read out of it.
func TestCallerTravelsInContext(t *testing.T) {
	want := identity.Anonymous()
	got := from(withCaller(context.Background(), want))
	if got != want {
		t.Fatalf("from(withCaller(c)) = %p, want %p", got, want)
	}
}
