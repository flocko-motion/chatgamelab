package core

import (
	"context"
	"net/mail"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// An organisation consents to the declaration when it registers. Without a
// consent to the current version (it registered before the declaration
// existed or before its current version, or a head withdrew the consent) it
// is suspended, and one of its heads consents to lift that: one head's
// consent covers the organisation. With it, the head confirms or changes the
// organisation's operator, address and youth-protection restriction.
// Consenting and withdrawing are a head's edits of their organisation's
// details: they take {Edit, OrganisationDetails, OfOwnOrganisation}, which a
// head holds while the organisation is suspended too.

// OrganisationConsent is an organisation as its heads consent for it: its
// consent status, and the details and restriction a consent confirms.
type OrganisationConsent struct {
	ID               uuid.UUID
	Name             string
	Operator         *string
	Street           *string
	PostalCode       *string
	City             *string
	PromptConstraint *string
	Status           legal.ConsentStatus
}

// ReadOrganisationConsent is organisation id as its heads consent for it. It
// takes {Edit, OrganisationDetails}: a head of that organisation. One the
// caller may not read does not exist for them (ErrNotFound).
func ReadOrganisationConsent(ctx context.Context, id uuid.UUID) (OrganisationConsent, error) {
	org, err := ReadOrganisation(ctx, id)
	if err != nil {
		return OrganisationConsent{}, err
	}
	if !org.MayEditDetails() {
		return OrganisationConsent{}, forbidden("only a head of this organisation consents in its name")
	}
	details, err := org.AsOrganisationDetails()
	if err != nil {
		return OrganisationConsent{}, err
	}
	settings, err := org.AsOrganisationSettings()
	if err != nil {
		return OrganisationConsent{}, err
	}
	record, err := org.AsOrganisationRecord()
	if err != nil {
		return OrganisationConsent{}, err
	}
	return OrganisationConsent{
		ID:               org.ID(),
		Name:             org.Name(),
		Operator:         details.Operator(),
		Street:           details.Street(),
		PostalCode:       details.PostalCode(),
		City:             details.City(),
		PromptConstraint: settings.PromptConstraint(),
		Status:           record.Consent(),
	}, nil
}

// DeclarationConsent is a head's consent to the current declaration, with
// the organisation's details they confirm with it.
type DeclarationConsent struct {
	PersonName     string // who consents, in full
	PersonFunction string // their function in the organisation
	Email          string // where the confirmation goes

	Operator         string
	Street           string
	PostalCode       string
	City             string
	PromptConstraint string // the organisation's youth-protection restriction

	AcceptDeclaration  bool
	DeclarationVersion string // the version shown to the head: the current one
}

// ConsentToDeclaration records the caller's consent to the current
// declaration for organisation id, stores its details, confirms the consent
// by mail and returns the organisation as its heads consent for it. It
// takes {Edit, OrganisationDetails}: a head of that organisation. It lifts a suspension for want of a consent, never an
// admin's.
func ConsentToDeclaration(ctx context.Context, id uuid.UUID, d DeclarationConsent) (OrganisationConsent, error) {
	org, err := ReadOrganisationConsent(ctx, id)
	if err != nil {
		return OrganisationConsent{}, err
	}
	d, err = d.validate()
	if err != nil {
		return OrganisationConsent{}, err
	}

	account := *from(ctx).Account()
	consent := legal.Consent{
		PersonName:     d.PersonName,
		PersonFunction: d.PersonFunction,
		Email:          d.Email,
		Organisation:   org.Name,
		Operator:       d.Operator,
		Street:         d.Street,
		PostalCode:     d.PostalCode,
		City:           d.City,
		Version:        legal.CurrentDeclaration().Version,
	}
	err = store.InTx(ctx, func(q *db.Queries) error {
		err := q.V2UpdateOrganisationDetails(ctx, db.V2UpdateOrganisationDetailsParams{
			ID:                id,
			Operator:          d.Operator,
			Street:            d.Street,
			PostalCode:        d.PostalCode,
			City:              d.City,
			PromptConstraints: d.PromptConstraint,
			ModifiedBy:        account,
		})
		if err != nil {
			return err
		}
		consent.At, err = recordConsent(ctx, q, id, account, consent)
		return err
	})
	if err != nil {
		return OrganisationConsent{}, err
	}
	language, err := myLanguage(ctx)
	if err != nil {
		return OrganisationConsent{}, err
	}
	confirmConsent(ctx, consent, language)
	return ReadOrganisationConsent(ctx, id)
}

// WithdrawConsent withdraws organisation id's standing consent in the
// caller's name, which suspends the organisation until a head consents
// again, and returns the organisation as its heads consent for it. It takes
// {Edit, OrganisationDetails}; an organisation without a
// standing consent has none to withdraw (409). The log keeps both entries.
func WithdrawConsent(ctx context.Context, id uuid.UUID) (OrganisationConsent, error) {
	org, err := ReadOrganisationConsent(ctx, id)
	if err != nil {
		return OrganisationConsent{}, err
	}
	latest, err := store.Queries().V2LatestConsent(ctx, id)
	if err != nil {
		return OrganisationConsent{}, err
	}
	if len(latest) == 0 || latest[0].Kind != consentKind {
		return OrganisationConsent{}, conflict(obj.ErrCodeConflict, "the organisation has no consent to withdraw")
	}
	account := *from(ctx).Account()
	_, err = store.Queries().V2RecordWithdrawal(ctx, db.V2RecordWithdrawalParams{
		InstitutionID:      id,
		UserID:             account,
		InstitutionName:    org.Name,
		DeclarationVersion: latest[0].DeclarationVersion,
		DeclarationHash:    latest[0].DeclarationHash,
	})
	if err != nil {
		return OrganisationConsent{}, err
	}
	log.Info("organisation's consent withdrawn", "institution_id", id, "user_id", account)
	return ReadOrganisationConsent(ctx, id)
}

// The kinds of entry in the consent log.
const (
	consentKind    = "consent"
	withdrawalKind = "withdrawal"
)

// consentStatus is whether organisation id's newest entry in the consent log
// covers the current declaration.
func consentStatus(ctx context.Context, id uuid.UUID) (legal.ConsentStatus, error) {
	latest, err := store.Queries().V2LatestConsent(ctx, id)
	if err != nil {
		return legal.ConsentStatus{}, err
	}
	if len(latest) == 0 {
		return legal.ConsentStatusOf(nil), nil
	}
	return legal.ConsentStatusOf(&legal.LogEntry{
		Withdrawal: latest[0].Kind == withdrawalKind,
		Version:    latest[0].DeclarationVersion,
		Hash:       latest[0].DeclarationHash,
		At:         latest[0].CreatedAt,
	}), nil
}

// validate refuses a consent no one may give, and returns it trimmed.
func (d DeclarationConsent) validate() (DeclarationConsent, error) {
	fields := []struct {
		name  string
		value *string
		max   int
	}{
		{"personName", &d.PersonName, maxPersonField},
		{"personFunction", &d.PersonFunction, maxPersonField},
		{"email", &d.Email, 2 * maxPersonField},
		{"operator", &d.Operator, maxOrganisationField},
		{"street", &d.Street, maxOrganisationField},
		{"postalCode", &d.PostalCode, maxPostalCode},
		{"city", &d.City, maxOrganisationField},
		{"promptConstraint", &d.PromptConstraint, legal.MaxPromptConstraintLength},
	}
	for _, f := range fields {
		if err := requireField(f.name, f.value, f.max); err != nil {
			return d, err
		}
	}
	if addr, err := mail.ParseAddress(d.Email); err != nil || addr.Address != d.Email {
		return d, refuse("invalid email address")
	}
	return d, checkDeclaration(d.AcceptDeclaration, d.DeclarationVersion)
}

// myLanguage is the language of the caller's account.
func myLanguage(ctx context.Context) (string, error) {
	me, err := ReadUser(ctx, *from(ctx).Account())
	if err != nil {
		return "", err
	}
	account, err := me.AsUserAccount()
	if err != nil {
		return "", err
	}
	return account.Language(), nil
}
