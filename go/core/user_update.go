package core

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"cgl/constants"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/lang"
	"cgl/obj"

	"github.com/google/uuid"
)

// UserChanges is what the caller asks to change in a user. A nil field
// stays as it is; only what differs from the stored user counts as a
// change. The role is not here: it changes with the organisation.
type UserChanges struct {
	// card
	Name *string

	// account
	Language             *string
	AgeGroup             *string // "u13", "u13p" or "u18"
	AiQualityTier        *string // high, medium, low; "": the server's default
	DefaultApiKeyShareID *uuid.UUID

	// contact
	Email *string // "": none
}

// ageGroups are the age groups an account may have.
var ageGroups = []string{obj.AgeGroupU13, obj.AgeGroupU13p, obj.AgeGroupU18}

// UpdateUser changes a user as the caller asks and returns them as stored.
//
// A user the caller may not read does not exist for them (ErrNotFound).
// Each section the request changes must be one the caller may edit: {Edit,
// UserCard} for the name, {Edit, UserAccount} for the account, {Edit,
// UserContact} for the email. One refused change refuses the whole request;
// a request that changes nothing writes nothing.
func UpdateUser(ctx context.Context, id uuid.UUID, ch UserChanges) (*UserCard, error) {
	user, err := ReadUser(ctx, id)
	if err != nil {
		return nil, err
	}
	next := user.row
	cardChanged := ch.applyCard(&next)
	accountChanged := ch.applyAccount(&next)
	contactChanged := ch.applyContact(&next)

	switch {
	case !cardChanged && !accountChanged && !contactChanged:
		return user, nil
	case cardChanged && !user.MayEditCard():
		return nil, forbidden("you may not change this user's name")
	case accountChanged && !user.MayEditAccount():
		return nil, forbidden("you may not change this user's account")
	case contactChanged && !user.MayEditContact():
		return nil, forbidden("you may not change this user's email")
	}
	if err := validateUser(ctx, user.row, next); err != nil {
		return nil, err
	}

	err = store.Queries().V2UpdateUser(ctx, db.V2UpdateUserParams{
		ID:                   id,
		ModifiedBy:           account(ctx).UUID, // whoever may edit a user has an account
		Name:                 next.name,
		Language:             next.language,
		AgeGroup:             sqlString(next.ageGroup),
		AiQualityTier:        sqlString(next.aiQualityTier),
		DefaultApiKeyShareID: nullUUID(next.defaultKeyShare),
		Email:                sqlString(next.email),
	})
	if err != nil {
		return nil, err
	}
	return ReadUser(ctx, id)
}

// DeleteUser deletes a user, with their games, keys, shares and roles.
//
// A user the caller may not read does not exist for them (ErrNotFound).
// Deleting takes {Delete, UserCard}. An organisation's last head is never
// deleted: it would leave the organisation without one.
func DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := ReadUser(ctx, id)
	if err != nil {
		return err
	}
	if !user.MayDelete() {
		return forbidden("you may not delete this user")
	}
	if org := user.row.organisationID; user.row.role == "head" && org != nil {
		heads, err := store.Queries().CountHeadsByInstitution(ctx, uuid.NullUUID{UUID: *org, Valid: true})
		if err != nil {
			return err
		}
		if heads <= 1 {
			return conflict(obj.ErrCodeLastHead, "an organisation needs at least one head")
		}
	}
	// v1's cascade over the user's games, keys, shares and roles; it moves
	// into the core as those entities are ported.
	return db.DeleteUser(ctx, id)
}

// applyCard writes the card's changes into row and says whether any differs.
func (ch UserChanges) applyCard(row *userRow) bool {
	changed := false
	set(&row.name, ch.Name, &changed)
	return changed
}

// applyAccount writes the account's changes into row and says whether any
// differs.
func (ch UserChanges) applyAccount(row *userRow) bool {
	changed := false
	set(&row.language, ch.Language, &changed)
	setOptional(&row.ageGroup, ch.AgeGroup, &changed)
	setOptional(&row.aiQualityTier, ch.AiQualityTier, &changed)
	if ch.DefaultApiKeyShareID != nil &&
		(row.defaultKeyShare == nil || *row.defaultKeyShare != *ch.DefaultApiKeyShareID) {
		row.defaultKeyShare, changed = ch.DefaultApiKeyShareID, true
	}
	return changed
}

// applyContact writes the contact's changes into row and says whether any
// differs.
func (ch UserChanges) applyContact(row *userRow) bool {
	changed := false
	setOptional(&row.email, ch.Email, &changed)
	return changed
}

// validateUser refuses a user no one may store, whoever asks: what changed
// from was to next.
func validateUser(ctx context.Context, was, next userRow) error {
	q := store.Queries()
	if next.name != was.name {
		switch {
		case strings.TrimSpace(next.name) == "":
			return refuse("a user needs a name")
		case constants.IsProfane(next.name):
			return badRequest(obj.ErrCodeProfaneName, "this name is not allowed")
		}
		taken, err := q.IsNameTakenByOther(ctx, db.IsNameTakenByOtherParams{
			Name: sql.NullString{String: next.name, Valid: true}, ID: next.id,
		})
		if err != nil {
			return err
		}
		if taken {
			return conflict(obj.ErrCodeConflict, "this name is taken")
		}
	}
	if next.email != nil && (was.email == nil || *next.email != *was.email) {
		taken, err := q.IsEmailTakenByOther(ctx, db.IsEmailTakenByOtherParams{
			Email: sql.NullString{String: *next.email, Valid: true}, ID: next.id,
		})
		if err != nil {
			return err
		}
		if taken {
			return conflict(obj.ErrCodeConflict, "this email is taken")
		}
	}
	if next.language != was.language && !lang.IsValidLanguageCode(next.language) {
		return refuse("unsupported language %q", next.language)
	}
	if next.ageGroup != nil && !slices.Contains(ageGroups, *next.ageGroup) {
		return refuse("unknown age group %q; known: %s", *next.ageGroup, strings.Join(ageGroups, ", "))
	}
	return nil
}
