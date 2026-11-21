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
	"cgl/obj"

	"github.com/lib/pq"
)

// An account exists from its first sign-in through Auth0, with no name and no
// role: it is registering. Registering completes it: a name and an age group,
// optionally an email. Until then it holds only what Everyone may in the
// Registering context, none of SignedIn's grants.

// maxUserName is the longest a user's name may be, in characters.
const maxUserName = 24

// Registration is what completes an account: a name, the age group from the
// age check, an optional email and the language the person registered in.
type Registration struct {
	Name     string
	Email    *string // nil or blank: none
	AgeGroup string  // "u13", "u13p" or "u18"
	Language string  // ISO 639-1; empty: the account keeps its language
}

// Register completes the caller's own account and returns their user. It
// takes {Create, UserCard, Own}, which only a registering caller holds; an
// account registers once. The rights the answer carries are the registering
// caller's: from the next request on, the caller is the registered account.
func Register(ctx context.Context, r Registration) (*UserCard, error) {
	c := from(ctx)
	switch {
	case c.Account() != nil && !c.Registering():
		return nil, conflict(obj.ErrCodeConflict, "the account is registered already")
	case !granted(ctx, rules.Create, rules.UserCard, rules.Own):
		return nil, forbidden("only an account still registering registers")
	}
	r, err := r.validate()
	if err != nil {
		return nil, err
	}
	id := *c.Account()

	q := store.Queries()
	taken, err := q.IsNameTakenByOther(ctx, db.IsNameTakenByOtherParams{
		Name: sql.NullString{String: r.Name, Valid: true}, ID: id,
	})
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, conflict(obj.ErrCodeConflict, "this name is taken")
	}
	if r.Email != nil {
		taken, err := q.IsEmailTakenByOther(ctx, db.IsEmailTakenByOtherParams{
			Email: sql.NullString{String: *r.Email, Valid: true}, ID: id,
		})
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, conflict(obj.ErrCodeConflict, "this email belongs to another account")
		}
	}

	n, err := q.CompleteRegistration(ctx, db.CompleteRegistrationParams{
		ID:       id,
		Name:     sql.NullString{String: r.Name, Valid: true},
		Email:    sqlString(r.Email),
		AgeGroup: sql.NullString{String: r.AgeGroup, Valid: true},
		Language: r.Language,
	})
	if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return nil, conflict(obj.ErrCodeConflict, "this name or email is taken") // a concurrent registration was first
	}
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, conflict(obj.ErrCodeConflict, "the account is registered already")
	}

	// v1's first role: individual, or admin for an admin email while no admin
	// exists yet (moving to sign-in with #452).
	db.AssignInitialRole(ctx, id, r.Email)
	return ReadUser(ctx, id)
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
	return r, nil
}
