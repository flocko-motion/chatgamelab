package core

import (
	"errors"
	"strings"
	"testing"

	"cgl/legal"
	"cgl/obj"

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

// validSignUp is a complete organisation sign-up.
func validSignUp() *OrganisationSignUp {
	return &OrganisationSignUp{
		PersonName: "Erika Muster", PersonFunction: "Leitung", Name: " JZ Nord ",
		Operator: "Stadt", Street: "Weg 1", PostalCode: "80331", City: "München",
		PromptConstraint: "Erlaube leichte Spannung.", AcceptDeclaration: true,
		DeclarationVersion: legal.CurrentDeclaration().Version,
	}
}

// Registering with an organisation takes an adult, an email, the terms and a
// complete sign-up consenting to the current declaration.
func TestRegistrationWithOrganisationValidate(t *testing.T) {
	email := "erika@example.org"
	valid := Registration{Name: "Erika", AgeGroup: "u18", Email: &email, AcceptTerms: true, Organisation: validSignUp()}
	got, err := valid.validate()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got.Organisation.Name != "JZ Nord" {
		t.Errorf("sign-up not trimmed: %q", got.Organisation.Name)
	}

	refused := map[string]struct {
		mutate func(r *Registration)
		code   string
	}{
		"a minor":       {func(r *Registration) { r.AgeGroup = "u13p" }, obj.ErrCodeInstitutionAgeNotAllowed},
		"no terms":      {func(r *Registration) { r.AcceptTerms = false }, obj.ErrCodeInstitutionConsentsRequired},
		"no email":      {func(r *Registration) { r.Email = nil }, obj.ErrCodeValidation},
		"no operator":   {func(r *Registration) { r.Organisation.Operator = " " }, obj.ErrCodeValidation},
		"no consent":    {func(r *Registration) { r.Organisation.AcceptDeclaration = false }, obj.ErrCodeInstitutionConsentsRequired},
		"old version":   {func(r *Registration) { r.Organisation.DeclarationVersion = "0.9" }, obj.ErrCodeDeclarationVersionOutdated},
		"long restrict": {func(r *Registration) { r.Organisation.PromptConstraint = strings.Repeat("a", 201) }, obj.ErrCodeValidation},
	}
	for name, c := range refused {
		r := valid
		r.Organisation = validSignUp()
		c.mutate(&r)
		_, err := r.validate()
		var coded interface{ Code() string }
		if !errors.As(err, &coded) || coded.Code() != c.code {
			t.Errorf("%s: want %s, got %v", name, c.code, err)
		}
	}
}
