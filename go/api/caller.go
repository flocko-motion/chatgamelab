// package: api / schemaf endpoints
// type:    logic
// job:     gives endpoints the account calling them, and parses the ids they take
// limits:  resolution and the registration lock live in api/auth; permissions stay in the handlers and db
package api

import (
	"context"

	"cgl/api/auth"
	"cgl/obj"

	"github.com/google/uuid"
)

// caller is the registered account calling an AuthRequired endpoint.
func caller(ctx context.Context) (*obj.User, error) {
	return auth.User(ctx)
}

// maybeCaller is the registered account calling an AuthOptional endpoint, or
// nil for an anonymous caller.
func maybeCaller(ctx context.Context) (*obj.User, error) {
	return auth.MaybeUser(ctx)
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

// MessageResp is a confirmation for the user.
type MessageResp struct {
	Message string `json:"message"`
}
