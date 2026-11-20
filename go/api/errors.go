// package: api / schemaf endpoints
// type:    logic
// job:     turns the platform's errors into schemaf's error responses, keeping the codes the frontend branches on
// limits:  schemaf writes the response (status, code, message); the codes themselves live in obj
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"cgl/obj"
	"cgl/tokenlock"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"gopkg.in/yaml.v3"
)

// appError gives an AppError or HTTPError from the db and game layers its
// status and keeps its code; any other error passes through and answers 500.
func appError(err error) error {
	var appErr *obj.AppError
	if errors.As(err, &appErr) {
		status, ok := codeStatus[appErr.Code]
		if !ok {
			status = http.StatusInternalServerError
		}
		return coded(status, appErr.Code, appErr.Message)
	}
	var httpErr *obj.HTTPError
	if errors.As(err, &httpErr) {
		return httpError(httpErr)
	}
	return err
}

// httpError answers with an HTTPError's status, code and message.
func httpError(e *obj.HTTPError) error {
	code := e.Code
	if code == "" {
		code = defaultCode(e.StatusCode)
	}
	return coded(e.StatusCode, code, e.Message)
}

// codeStatus is the status an AppError's code answers with; a code missing
// here answers 500.
var codeStatus = map[string]int{
	obj.ErrCodeValidation:               http.StatusBadRequest,
	obj.ErrCodeInvalidInput:             http.StatusBadRequest,
	obj.ErrCodeInvalidPlatform:          http.StatusBadRequest,
	obj.ErrCodeNameTooLong:              http.StatusBadRequest,
	obj.ErrCodeProfaneName:              http.StatusBadRequest,
	obj.ErrCodePublicSlugInvalid:        http.StatusBadRequest,
	obj.ErrCodePublicDescriptionTooLong: http.StatusBadRequest,
	obj.ErrCodePublicLinksInvalid:       http.StatusBadRequest,
	obj.ErrCodeUnauthorized:             http.StatusUnauthorized,
	obj.ErrCodeForbidden:                http.StatusForbidden,
	obj.ErrCodeUserNotRegistered:        http.StatusForbidden,
	obj.ErrCodeNotFound:                 http.StatusNotFound,
	obj.ErrCodeConflict:                 http.StatusConflict,
	obj.ErrCodeDuplicateName:            http.StatusConflict,
	obj.ErrCodeLastHead:                 http.StatusConflict,
	obj.ErrCodePublicSlugTaken:          http.StatusConflict,
}

// sentinels maps the statuses the platform answers with to schemaf's sentinels
// and their default codes.
var sentinels = map[int]struct {
	err  error
	code string
}{
	http.StatusBadRequest:         {schemafapi.ErrBadRequest, "bad_request"},
	http.StatusUnauthorized:       {schemafapi.ErrUnauthorized, "unauthorized"},
	http.StatusForbidden:          {schemafapi.ErrForbidden, "forbidden"},
	http.StatusNotFound:           {schemafapi.ErrNotFound, "not_found"},
	http.StatusConflict:           {schemafapi.ErrConflict, "conflict"},
	http.StatusTooManyRequests:    {schemafapi.ErrTooManyRequests, "too_many_requests"},
	http.StatusServiceUnavailable: {schemafapi.ErrUnavailable, "unavailable"},
}

// defaultCode is schemaf's code for a status, "internal" for any other.
func defaultCode(status int) string {
	if s, ok := sentinels[status]; ok {
		return s.code
	}
	return "internal"
}

// coded is an error answering with status, code and message. A status
// without a sentinel answers 500, keeping the code; a code schemaf reserves
// for another status gives way to the status's own.
func coded(status int, code, msg string) error {
	s, ok := sentinels[status]
	if !ok {
		return codedInternal{code: code, msg: msg}
	}
	if isDefaultCode(code) && code != s.code {
		code = s.code
	}
	return schemafapi.Error(s.err, code, msg)
}

func isDefaultCode(code string) bool {
	if code == "internal" {
		return true
	}
	for _, s := range sentinels {
		if s.code == code {
			return true
		}
	}
	return false
}

// codedInternal is a 500 that keeps its code; schemaf withholds its message.
type codedInternal struct{ code, msg string }

func (e codedInternal) Error() string { return e.msg }
func (e codedInternal) Code() string  { return e.code }

// Refusals with schemaf's default code and a message for the user.
func errBadRequest(msg string) error   { return coded(http.StatusBadRequest, "bad_request", msg) }
func errUnauthorized(msg string) error { return coded(http.StatusUnauthorized, "unauthorized", msg) }
func errForbidden(msg string) error    { return coded(http.StatusForbidden, "forbidden", msg) }
func errNotFound(msg string) error     { return coded(http.StatusNotFound, "not_found", msg) }
func errConflict(msg string) error     { return coded(http.StatusConflict, "conflict", msg) }

// codeTokenLocked tells the frontend to show "try again later".
const codeTokenLocked = "token_locked"

// tokenLocked refuses a route that looks up a guessable token while the
// site-wide token lock holds (-> tokenlock); nil otherwise.
func tokenLocked() error {
	retry := tokenlock.RetryAfter()
	if retry <= 0 {
		return nil
	}
	return schemafapi.RetryAfter(
		schemafapi.Error(schemafapi.ErrTooManyRequests, codeTokenLocked, "Too many failed attempts, try again later"),
		max(retry, time.Second))
}

// writeJSON answers a raw endpoint with v as JSON and status 200.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeYAML answers a raw endpoint with v as YAML and status 200.
func writeYAML(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/x-yaml")
	_ = yaml.NewEncoder(w).Encode(v)
}
