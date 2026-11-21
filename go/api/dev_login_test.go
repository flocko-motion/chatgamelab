package api

import (
	"context"
	"testing"

	"cgl/api/apitest"
)

func TestDevLoginIsNotFoundOutsideDevMode(t *testing.T) {
	t.Setenv("SCHEMAF_DEV", "")

	_, err := DevLoginEndpoint{}.Handle(context.Background(), DevLoginReq{Role: "admin-1"})

	rec, body := apitest.ServeError(t, func() error { return err })
	if rec.Code != 404 || body["code"] != "not_found" {
		t.Errorf("got %d %v, want 404 not_found", rec.Code, body)
	}
}

func TestDevLoginRefusesAnUnknownRole(t *testing.T) {
	t.Setenv("SCHEMAF_DEV", "1")

	_, err := DevLoginEndpoint{}.Handle(context.Background(), DevLoginReq{Role: "superuser"})

	rec, body := apitest.ServeError(t, func() error { return err })
	if rec.Code != 404 || body["code"] != "not_found" {
		t.Errorf("got %d %v, want 404 not_found", rec.Code, body)
	}
}

func TestDevLoginKnowsEverySeededAccount(t *testing.T) {
	for _, role := range []string{"admin-1", "admin-2", "head-1", "head-2", "staff-1", "staff-2", "individual-1", "individual-2", "guest"} {
		if _, ok := devLoginAccounts[role]; !ok {
			t.Errorf("no account for role %q", role)
		}
	}
}
