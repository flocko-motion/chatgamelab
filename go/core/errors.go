package core

import (
	"database/sql"
	"fmt"

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
