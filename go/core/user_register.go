package core

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"slices"
	"strings"
	"unicode/utf8"

	"cgl/constants"
	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"
	"cgl/obj"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// An account exists from its first sign-in through Auth0, with no name and no
// role: it is registering. Registering completes it: a name and an age group,
// optionally an email. Until then it holds only what Everyone may in the
// Registering context, none of SignedIn's grants.
//
// An account registers as an individual, or as the head of an organisation it
// registers with it (organisation_register.go).

// maxUserName is the longest a user's name may be, in characters.
const maxUserName = 24

// Registration is what completes an account: a name, the age group from the
// age check, an optional email and the language the person registered in;
// with an organisation, the organisation it registers too.
type Registration struct {
	Name     string
	Email    *string // nil or blank: none
	AgeGroup string  // "u13", "u13p" or "u18"
	Language string  // ISO 639-1; empty: the account keeps its language

	// Organisation, when set, is registered with the account, which becomes
	// its head instead of an individual. It takes an adult (u18), an email
	// and AcceptTerms.
	Organisation *OrganisationSignUp
	// AcceptTerms is the consent to the terms of use and the privacy policy,
	// which an organisation's form asks for; an individual gives it before
	// signing up.
	AcceptTerms bool
}

// Register completes the caller's own account and returns their user. It
// takes {Create, UserCard, Own}, which only a registering caller holds, and
// with an organisation {Create, Organisation, Own}; an account registers
// once. The rights the answer carries are the registering caller's: from the
// next request on, the caller is the registered account.
func Register(ctx context.Context, r Registration) (*UserCard, error) {
	c := from(ctx)
	switch {
	case c.Account() != nil && !c.Registering():
		return nil, conflict(obj.ErrCodeConflict, "the account is registered already")
	case !granted(ctx, rules.Create, rules.UserCard, rules.Own):
		return nil, forbidden("only an account still registering registers")
	case r.Organisation != nil && !mayRegisterAsHead(ctx):
		return nil, forbidden("you may not register an organisation")
	}
	r, err := r.validate()
	if err != nil {
		return nil, err
	}
	id := *c.Account()
	if err := r.checkFree(ctx, id); err != nil {
		return nil, err
	}

	if r.Organisation == nil {
		if err := completeRegistration(ctx, store.Queries(), id, r); err != nil {
			return nil, err
		}
		// the first role: individual, or admin for an admin email while no
		// admin exists yet
		if err := assignInitialRole(ctx, id, r.Email); err != nil {
			return nil, err
		}
		return ReadUser(ctx, id)
	}
	return registerWithOrganisation(ctx, id, r)
}

// registerWithOrganisation completes account id and registers its
// organisation, all or nothing, then confirms the consent by mail.
func registerWithOrganisation(ctx context.Context, id uuid.UUID, r Registration) (*UserCard, error) {
	var consent legal.Consent
	err := store.InTx(ctx, func(q *db.Queries) error {
		if err := completeRegistration(ctx, q, id, r); err != nil {
			return err
		}
		var err error
		consent, err = registerOrganisation(ctx, q, id, *r.Email, *r.Organisation)
		return err
	})
	if err != nil {
		return nil, err
	}
	language, err := myLanguage(ctx)
	if err != nil {
		return nil, err
	}
	confirmConsent(consent, language)
	return ReadUser(ctx, id)
}

// completeRegistration stores r on account id, refusing a name or email a
// concurrent registration took first, and an account registered already.
func completeRegistration(ctx context.Context, q *db.Queries, id uuid.UUID, r Registration) error {
	n, err := q.V2CompleteRegistration(ctx, db.V2CompleteRegistrationParams{
		ID:       id,
		Name:     sql.NullString{String: r.Name, Valid: true},
		Email:    sqlString(r.Email),
		AgeGroup: sql.NullString{String: r.AgeGroup, Valid: true},
		Language: r.Language,
	})
	if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return conflict(obj.ErrCodeConflict, "this name or email is taken") // a concurrent registration was first
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return conflict(obj.ErrCodeConflict, "the account is registered already")
	}
	return nil
}

// checkFree refuses a name or email another account has.
func (r Registration) checkFree(ctx context.Context, id uuid.UUID) error {
	q := store.Queries()
	taken, err := q.V2IsNameTakenByOther(ctx, db.V2IsNameTakenByOtherParams{
		Name: sql.NullString{String: r.Name, Valid: true}, ID: id,
	})
	if err != nil {
		return err
	}
	if taken {
		return conflict(obj.ErrCodeConflict, "this name is taken")
	}
	if r.Email == nil {
		return nil
	}
	taken, err = q.V2EmailOfOtherAccount(ctx, db.V2EmailOfOtherAccountParams{Email: *r.Email, ID: id})
	if err != nil {
		return err
	}
	if taken {
		return conflict(obj.ErrCodeEmailBelongsToOtherAccount, "this email belongs to another account")
	}
	return nil
}

// validate refuses a registration no one may complete, and returns it
// trimmed, with a blank email dropped.
func (r Registration) validate() (Registration, error) {
	r.Name = strings.TrimSpace(r.Name)
	switch {
	case r.Name == "":
		return r, refuse("a name is required")
	case utf8.RuneCountInString(r.Name) > maxUserName:
		return r, nameTooLong("a name has at most %d characters", maxUserName)
	case constants.IsProfane(r.Name):
		return r, badRequest(obj.ErrCodeProfaneName, "this name is not allowed")
	}

	if r.Email != nil {
		email := strings.TrimSpace(*r.Email)
		if email == "" {
			r.Email = nil
		} else if _, err := mail.ParseAddress(email); err != nil {
			return r, refuse("invalid email address")
		} else {
			r.Email = &email
		}
	}

	switch {
	case r.AgeGroup == "":
		return r, refuse("an age group is required")
	case !slices.Contains(ageGroups, r.AgeGroup):
		return r, refuse("unknown age group %q; known: %s", r.AgeGroup, strings.Join(ageGroups, ", "))
	}

	r.Language = strings.TrimSpace(r.Language)
	if r.Organisation != nil {
		return r, r.validateOrganisation()
	}
	return r, nil
}

// validateOrganisation refuses an organisation's registration no one may
// complete, trimming its sign-up.
func (r *Registration) validateOrganisation() error {
	switch {
	case !r.AcceptTerms:
		return badRequest(obj.ErrCodeInstitutionConsentsRequired, "the terms of use and the privacy policy must be accepted")
	case r.Email == nil:
		return refuse("an organisation's head needs an email")
	}
	if err := checkAdult(&r.AgeGroup); err != nil {
		return err
	}
	s, err := r.Organisation.validate()
	r.Organisation = &s
	return err
}
