package core

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"
	"cgl/log"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OrganisationDraft is a new organisation as an admin creates it: its name,
// and optionally its operator (Träger) and address.
type OrganisationDraft struct {
	Name       string
	Operator   string
	Street     string
	PostalCode string
	City       string
}

// CreateOrganisation creates an organisation for others and returns it. It
// takes {Create, OrganisationCard, Everywhere}: the admin's. It has no head
// until someone is invited as one; an organisation registers itself through
// Register.
func CreateOrganisation(ctx context.Context, d OrganisationDraft) (*OrganisationCard, error) {
	if _, forOthers := mayCreateOrganisation(ctx); !forOthers {
		return nil, forbidden("only an admin creates an organisation")
	}
	if err := requireField("name", &d.Name, maxOrganisationField); err != nil {
		return nil, err
	}
	optional := func(name string, value *string, max int) (*string, error) {
		if *value = strings.TrimSpace(*value); *value == "" {
			return nil, nil
		}
		return value, requireField(name, value, max)
	}
	operator, err := optional("operator", &d.Operator, maxOrganisationField)
	if err != nil {
		return nil, err
	}
	street, err := optional("street", &d.Street, maxOrganisationField)
	if err != nil {
		return nil, err
	}
	postalCode, err := optional("postalCode", &d.PostalCode, maxPostalCode)
	if err != nil {
		return nil, err
	}
	city, err := optional("city", &d.City, maxOrganisationField)
	if err != nil {
		return nil, err
	}
	if err := checkNameFree(ctx, d.Name); err != nil {
		return nil, err
	}

	id := uuid.New()
	err = store.Queries().V2CreateOrganisation(ctx, db.V2CreateOrganisationParams{
		ID:         id,
		Creator:    account(ctx).UUID,
		Name:       d.Name,
		Operator:   sqlString(operator),
		Street:     sqlString(street),
		PostalCode: sqlString(postalCode),
		City:       sqlString(city),
	})
	if err := nameTakenOr(err); err != nil {
		return nil, err
	}
	log.Info("organisation created", "institution_id", id, "name", d.Name, "admin_id", account(ctx).UUID)
	return ReadOrganisation(ctx, id)
}

// OrganisationChanges are the changes to an organisation the caller asks
// for; nil leaves a field as it is.
type OrganisationChanges struct {
	// card
	Name *string

	// settings
	PromptConstraint *string // its youth-protection restriction; "": none, the site's rule applies
}

// UpdateOrganisation changes an organisation as the caller asks and returns
// it as stored.
//
// An organisation the caller may not read does not exist for them
// (ErrNotFound). Each section the request changes must be one the caller may
// edit: {Edit, OrganisationCard}, {Edit, OrganisationSettings}; otherwise
// the whole request is refused and nothing is written. A request that
// changes nothing writes nothing.
func UpdateOrganisation(ctx context.Context, id uuid.UUID, ch OrganisationChanges) (*OrganisationCard, error) {
	org, err := ReadOrganisation(ctx, id)
	if err != nil {
		return nil, err
	}
	ch, err = ch.validate()
	if err != nil {
		return nil, err
	}
	row := org.row
	cardChanged := ch.Name != nil && *ch.Name != row.name
	settingsChanged := false
	setOptional(&row.promptConstraint, ch.PromptConstraint, &settingsChanged)
	switch {
	case cardChanged && !org.MayEditCard():
		return nil, forbidden("you may not rename this organisation")
	case settingsChanged && !org.MayEditSettings():
		return nil, forbidden("you may not change this organisation's settings")
	case !cardChanged && !settingsChanged:
		return org, nil
	}
	if cardChanged {
		if !strings.EqualFold(*ch.Name, org.row.name) {
			if err := checkNameFree(ctx, *ch.Name); err != nil {
				return nil, err
			}
		}
		row.name = *ch.Name
	}
	err = store.Queries().V2UpdateOrganisation(ctx, db.V2UpdateOrganisationParams{
		ID:                id,
		Name:              row.name,
		PromptConstraints: sqlString(row.promptConstraint),
		ModifiedBy:        account(ctx).UUID,
	})
	if err := nameTakenOr(err); err != nil {
		return nil, err
	}
	return ReadOrganisation(ctx, id)
}

// validate refuses changes no one may make, and returns them trimmed.
func (ch OrganisationChanges) validate() (OrganisationChanges, error) {
	if ch.Name != nil {
		name := *ch.Name
		if err := requireField("name", &name, maxOrganisationField); err != nil {
			return ch, err
		}
		ch.Name = &name
	}
	if ch.PromptConstraint != nil {
		constraint := strings.TrimSpace(*ch.PromptConstraint)
		if utf8.RuneCountInString(constraint) > legal.MaxPromptConstraintLength {
			return ch, refuse("promptConstraint has at most %d characters", legal.MaxPromptConstraintLength)
		}
		ch.PromptConstraint = &constraint
	}
	return ch, nil
}

// DeleteOrganisation deletes an organisation for good, with its workshops
// and the guests in them; its members stay, as individuals. An organisation
// the caller may not read does not exist for them (ErrNotFound). It takes
// {Delete, OrganisationCard}; suspending keeps the data instead.
func DeleteOrganisation(ctx context.Context, id uuid.UUID) error {
	org, err := ReadOrganisation(ctx, id)
	if err != nil {
		return err
	}
	if !org.MayDelete() {
		return forbidden("you may not delete this organisation")
	}
	// v1's cascade over workshops, guests, invites, key shares and roles; it
	// moves into the core as those entities are ported.
	if err := db.DeleteInstitutionCascade(ctx, id, account(ctx).UUID); err != nil {
		return err
	}
	log.Info("organisation deleted", "institution_id", id, "admin_id", account(ctx).UUID)
	return nil
}

// checkNameFree refuses a name another organisation has, ignoring case.
func checkNameFree(ctx context.Context, name string) error {
	taken, err := store.Queries().V2OrganisationNameTaken(ctx, name)
	if err != nil {
		return err
	}
	if taken {
		return errOrganisationNameTaken
	}
	return nil
}

// nameTakenOr is err, read as errOrganisationNameTaken where the database
// refused a name another organisation took meanwhile.
func nameTakenOr(err error) error {
	if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return errOrganisationNameTaken
	}
	return err
}
