// package: api / schemaf endpoints
// type:    logic
// job:     parses the ids endpoints take
// limits:  the caller is the secure core's (core.Resolve)
package api

import (
	"cgl/obj"

	"github.com/google/uuid"
)

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
