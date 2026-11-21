// package: api / schemaf endpoints
// type:    logic
// job:     gives endpoints the account calling them, and parses the ids they take
// limits:  the caller is resolved by the secure core (core.Resolve); these are its legacy bridge, gone once every endpoint is ported
package api

import (
	"context"

	"cgl/core"
	"cgl/obj"

	"github.com/google/uuid"
)

// caller is the registered account calling an AuthRequired endpoint.
func caller(ctx context.Context) (*obj.User, error) {
	return core.LegacyCaller(ctx)
}

// maybeCaller is the registered account calling an AuthOptional endpoint, or
// nil for a guest nobody signed in.
func maybeCaller(ctx context.Context) (*obj.User, error) {
	return core.LegacyMaybeCaller(ctx)
}

// idOf is the account's id, or nil without an account.
func idOf(user *obj.User) *uuid.UUID {
	if user == nil {
		return nil
	}
	return &user.ID
}

// parseID parses an id from the path, refusing it with msg (e.g. "Invalid game ID").
func parseID(raw, msg string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errBadRequest(msg)
	}
	return id, nil
}

// parseIDValidation is parseID for routes that refuse a bad id as a
// validation error, with its code.
func parseIDValidation(raw, msg string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, appError(obj.ErrValidation(msg))
	}
	return id, nil
}

// opt is an optional query parameter's value, empty when it was not sent.
func opt(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// MessageResp is a confirmation for the user.
type MessageResp struct {
	Message string `json:"message"`
}
