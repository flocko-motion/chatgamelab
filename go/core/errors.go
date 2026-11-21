package core

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"github.com/google/uuid"
)

// ErrNotFound: the caller may not see it, so for them it does not exist.
var ErrNotFound = schemafapi.Error(schemafapi.ErrNotFound, "not_found", "not found")

// ErrLocked: the caller holds the entity, so its existence is no secret; a
// section they may not see is refused, not hidden.
var ErrLocked = schemafapi.Error(schemafapi.ErrForbidden, "forbidden", "section is locked")

// refuse is a request the core cannot serve as asked: 400, validation_error.
func refuse(format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeValidation, fmt.Sprintf(format, args...))
}

// forbidden: the caller may see it, but not do this to it: 403.
func forbidden(format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrForbidden, obj.ErrCodeForbidden, fmt.Sprintf(format, args...))
}

// forbiddenWith: 403 with a code of its own, e.g. institution_age_not_allowed.
func forbiddenWith(code, format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrForbidden, code, fmt.Sprintf(format, args...))
}

// badRequest: 400 with a code of its own, e.g. profane_name.
func badRequest(code, format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrBadRequest, code, fmt.Sprintf(format, args...))
}

// conflict: 409, the request clashes with what is stored (a taken name,
// the last head).
func conflict(code, format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrConflict, code, fmt.Sprintf(format, args...))
}

// fromV1 reads an error of a v1 validator the core still calls as the core
// answers it: a taken public page address 409, any other refusal 400 with
// its code. Any other error passes.
func fromV1(err error) error {
	var appErr *obj.AppError
	if !errors.As(err, &appErr) {
		return err
	}
	if appErr.Code == obj.ErrCodePublicSlugTaken {
		return conflict(appErr.Code, "%s", appErr.Message)
	}
	return badRequest(appErr.Code, "%s", appErr.Message)
}

// nameTooLong: 400, name_too_long.
func nameTooLong(format string, args ...any) error {
	return schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeNameTooLong, fmt.Sprintf(format, args...))
}

func ptrUUID(id uuid.NullUUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	return &id.UUID
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func nullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func sqlString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
