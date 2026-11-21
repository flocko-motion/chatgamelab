package core

import (
	"errors"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// A registration needs a name of at most 24 characters and an age group; a
// blank email is none, a malformed one is refused.
func TestRegistrationValidate(t *testing.T) {
	email := "ada@example.org"
	valid := []Registration{
		{Name: "Ada", AgeGroup: "u18"},
		{Name: "  Ada  ", AgeGroup: "u13", Email: &email},
		{Name: "abcdefghijklmnopqrstuvwx", AgeGroup: "u13p"}, // 24 runes
	}
	for _, r := range valid {
		if _, err := r.validate(); err != nil {
			t.Errorf("%+v: unexpected error %v", r, err)
		}
	}

	bad := "not an address"
	blank := "   "
	invalid := []Registration{
		{Name: "", AgeGroup: "u18"},
		{Name: "   ", AgeGroup: "u18"},
		{Name: "abcdefghijklmnopqrstuvwxy", AgeGroup: "u18"}, // 25 runes
		{Name: "Ada"},
		{Name: "Ada", AgeGroup: "u12"},
		{Name: "Ada", AgeGroup: "u18", Email: &bad},
	}
	for _, r := range invalid {
		if _, err := r.validate(); !errors.Is(err, schemafapi.ErrBadRequest) {
			t.Errorf("%+v: want a bad request, got %v", r, err)
		}
	}

	got, err := Registration{Name: " Ada ", AgeGroup: "u18", Email: &blank}.validate()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Ada" || got.Email != nil {
		t.Errorf("want trimmed name and no email, got %q %v", got.Name, got.Email)
	}
}
