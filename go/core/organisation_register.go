package core

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"
	"cgl/log"
	"cgl/mail"
	"cgl/obj"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// An organisation registers itself while the account of the person who signs
// for it registers (Register with an organisation): the person consents to
// the declaration in its name and becomes its head. The organisation exists
// at once; its first consent, given by its creator, records how it
// registered, and is confirmed by mail. The person is an adult, reached at
// their account's email.

// The longest the fields of an organisation's registration may be, in
// characters.
const (
	maxPersonField       = 100
	maxOrganisationField = 150
	maxPostalCode        = 20
)

// OrganisationSignUp is what registering an organisation takes beyond an
// account: who signs for it, the organisation, its youth-protection
// restriction, and the consent to the declaration.
type OrganisationSignUp struct {
	PersonName     string // who signs, in full
	PersonFunction string // their function in the organisation

	Name       string
	Operator   string // the organisation's Träger
	Street     string
	PostalCode string
	City       string

	// PromptConstraint is the organisation's youth-protection restriction
	// (Jugendschutz-Prompt-Einschränkung), which the declaration obliges it
	// to set.
	PromptConstraint string

	AcceptDeclaration  bool
	DeclarationVersion string // the version shown to the person: the current one
}

// registerOrganisation stores organisation s with account as its head, signed
// for by them at email, records their consent and returns it. It decides
// nothing: its caller has checked that the account may, and runs it in its
// transaction.
func registerOrganisation(ctx context.Context, q *db.Queries, account uuid.UUID, email string, s OrganisationSignUp) (legal.Consent, error) {
	taken, err := q.V2OrganisationNameTaken(ctx, s.Name)
	if err != nil {
		return legal.Consent{}, err
	}
	if taken {
		return legal.Consent{}, errOrganisationNameTaken
	}

	id := uuid.New()
	err = q.V2CreateOrganisation(ctx, db.V2CreateOrganisationParams{
		ID:                id,
		Creator:           account,
		Name:              s.Name,
		Operator:          sqlString(&s.Operator),
		Street:            sqlString(&s.Street),
		PostalCode:        sqlString(&s.PostalCode),
		City:              sqlString(&s.City),
		PromptConstraints: sqlString(&s.PromptConstraint),
	})
	if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return legal.Consent{}, errOrganisationNameTaken // a concurrent registration was first
	}
	if err != nil {
		return legal.Consent{}, err
	}
	if err := q.V2MakeHead(ctx, db.V2MakeHeadParams{UserID: account, InstitutionID: uuid.NullUUID{UUID: id, Valid: true}}); err != nil {
		return legal.Consent{}, err
	}

	consent := legal.Consent{
		PersonName:      s.PersonName,
		PersonFunction:  s.PersonFunction,
		Email:           email,
		Organisation:    s.Name,
		Operator:        s.Operator,
		Street:          s.Street,
		PostalCode:      s.PostalCode,
		City:            s.City,
		Version:         legal.CurrentDeclaration().Version,
		NewOrganisation: true,
	}
	consent.At, err = recordConsent(ctx, q, id, account, consent)
	if err != nil {
		return legal.Consent{}, err
	}
	log.Info("organisation registered", "institution_id", id, "name", s.Name, "user_id", account)
	return consent, nil
}

// mayRegisterAsHead says whether the caller may register an organisation
// with themselves as its head.
func mayRegisterAsHead(ctx context.Context) bool {
	asHead, _ := mayCreateOrganisation(ctx)
	return asHead
}

var errOrganisationNameTaken = conflict(obj.ErrCodeInstitutionNameTaken, "an organisation with this name exists already")

// checkAdult refuses an age group other than u18: only an adult signs for an
// organisation.
func checkAdult(ageGroup *string) error {
	if ageGroup == nil || *ageGroup != obj.AgeGroupU18 {
		return forbiddenWith(obj.ErrCodeInstitutionAgeNotAllowed, "only an adult (18+) registers an organisation")
	}
	return nil
}

// validate refuses a sign-up no one may complete, and returns it trimmed.
func (s OrganisationSignUp) validate() (OrganisationSignUp, error) {
	fields := []struct {
		name  string
		value *string
		max   int
	}{
		{"personName", &s.PersonName, maxPersonField},
		{"personFunction", &s.PersonFunction, maxPersonField},
		{"institutionName", &s.Name, maxOrganisationField},
		{"operator", &s.Operator, maxOrganisationField},
		{"street", &s.Street, maxOrganisationField},
		{"postalCode", &s.PostalCode, maxPostalCode},
		{"city", &s.City, maxOrganisationField},
		{"promptConstraint", &s.PromptConstraint, legal.MaxPromptConstraintLength},
	}
	for _, f := range fields {
		if err := requireField(f.name, f.value, f.max); err != nil {
			return s, err
		}
	}
	return s, checkDeclaration(s.AcceptDeclaration, s.DeclarationVersion)
}

// requireField trims *value and refuses it empty or longer than max
// characters.
func requireField(name string, value *string, max int) error {
	*value = strings.TrimSpace(*value)
	switch {
	case *value == "":
		return refuse("%s is required", name)
	case utf8.RuneCountInString(*value) > max:
		return refuse("%s has at most %d characters", name, max)
	}
	return nil
}

// checkDeclaration refuses a consent not given, or given to a version other
// than the current one.
func checkDeclaration(accepted bool, version string) error {
	switch {
	case !accepted:
		return badRequest(obj.ErrCodeInstitutionConsentsRequired, "the declaration must be accepted")
	case version != legal.CurrentDeclaration().Version:
		return badRequest(obj.ErrCodeDeclarationVersionOutdated, "the declaration was updated; read the current version")
	}
	return nil
}

// recordConsent records consent to the current declaration for organisation
// id, given by account, and returns when it was given.
func recordConsent(ctx context.Context, q *db.Queries, id, account uuid.UUID, c legal.Consent) (time.Time, error) {
	declaration := legal.CurrentDeclaration()
	return q.V2RecordConsent(ctx, db.V2RecordConsentParams{
		InstitutionID:       id,
		UserID:              account,
		Email:               c.Email,
		PersonName:          c.PersonName,
		PersonFunction:      c.PersonFunction,
		InstitutionName:     c.Organisation,
		InstitutionOperator: c.Operator,
		Street:              c.Street,
		PostalCode:          c.PostalCode,
		City:                c.City,
		DeclarationVersion:  declaration.Version,
		DeclarationHash:     declaration.Hash,
	})
}

// confirmConsent mails the confirmation of c: to the person, in language, and
// a copy to the site's operator. A mail that fails is logged; the consent
// stands.
func confirmConsent(ctx context.Context, c legal.Consent, language string) {
	b := DefaultBranding()
	if stored, err := storedBranding(ctx, store.Queries()); err == nil {
		b = applyBranding(stored)
	} else {
		log.Warn("site branding unreadable, mailing with its defaults", "error", err)
	}
	server, copyTo, err := mailServer(ctx)
	if err != nil {
		log.Warn("mail settings unreadable, logging the confirmation instead", "error", err)
	}
	if copyTo == "" {
		copyTo = b.SupportEmail
	}
	site := legal.Site{Name: b.SiteName, Operator: b.OperatorName, OperatorLegal: b.LegalOperator()}
	for _, msg := range legal.Confirmations(c, site, language, copyTo) {
		_ = mail.Send(server, msg) // Send logs its failures
	}
}
