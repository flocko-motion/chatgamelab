// package: testutil / the words for refusals
// type:    logic
// job:     asserts that the API refused a call, and why, by the code in its error body
// limits:  checks the code of an answer, never its wording; "it failed" alone is no assertion
package testutil

import (
	"encoding/json"
	"errors"
)

// The codes the API refuses with ({"error": …, "code": <code>}).
const (
	// schemaf's codes for a status
	CodeBadRequest      = "bad_request"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeTooManyRequests = "too_many_requests"

	// the platform's codes
	CodeValidation               = "validation_error"
	CodeInvalidInput             = "invalid_input"
	CodeInvalidPlatform          = "invalid_platform"
	CodeDuplicateName            = "duplicate_name"
	CodeNameTooLong              = "name_too_long"
	CodeProfaneName              = "profane_name"
	CodeLastHead                 = "last_head"
	CodeUserNotRegistered        = "user_not_registered"
	CodePublicSlugTaken          = "public_slug_taken"
	CodePublicSlugInvalid        = "public_slug_invalid"
	CodePublicDescriptionTooLong = "public_description_too_long"
	CodePublicLinksInvalid       = "public_links_invalid"
	CodeTokenLocked              = "token_locked"
	CodeWorkshopInactive         = "auth_workshop_inactive"
	CodeNoApiKey                 = "no_api_key"

	// organisations registering themselves
	CodeInstitutionAgeNotAllowed    = "institution_age_not_allowed"
	CodeInstitutionNameTaken        = "institution_name_taken"
	CodeEmailBelongsToOtherAccount  = "email_belongs_to_other_account"
	CodeDeclarationVersionOutdated  = "declaration_version_outdated"
	CodeInstitutionConsentsRequired = "institution_consents_required"
	CodeOrganisationSuspended       = "organisation_suspended"
	CodeGuestStays                  = "guest_stays"

	// the engine's problems, refusing a session
	CodeDenied      = "denied"
	CodePaused      = "paused"
	CodeUnknown     = "unknown"
	CodeNoKey       = "no_key"
	CodeNoPlays     = "no_plays"
	CodeStale       = "stale"
	CodeKeyRejected = "key_rejected"
)

// Refused asserts that the API refused the call with the code.
func (s *BaseSuite) Refused(err error, code string, msgAndArgs ...any) bool {
	s.T().Helper()
	var ref *Refusal
	if !errors.As(err, &ref) {
		if err == nil {
			return s.Fail("expected a refusal with code "+code+", but the call succeeded", msgAndArgs...)
		}
		return s.Fail("expected a refusal with code "+code+", got another error: "+err.Error(), msgAndArgs...)
	}
	return s.Equal(code, ref.Code, append([]any{"refused with %d %q: %s", ref.Status, ref.Code, ref.Message}, msgAndArgs...)...)
}

// BadRequest asserts the call was refused as a bad request.
func (s *BaseSuite) BadRequest(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeBadRequest, msgAndArgs...)
}

// Unauthorized asserts the call was refused for lacking a sign-in.
func (s *BaseSuite) Unauthorized(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeUnauthorized, msgAndArgs...)
}

// Forbidden asserts the call was refused to this caller.
func (s *BaseSuite) Forbidden(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeForbidden, msgAndArgs...)
}

// NotFound asserts the call was refused because what it names doesn't exist
// (for this caller).
func (s *BaseSuite) NotFound(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeNotFound, msgAndArgs...)
}

// Conflict asserts the call was refused as clashing with what exists.
func (s *BaseSuite) Conflict(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeConflict, msgAndArgs...)
}

// Invalid asserts the call was refused for invalid input.
func (s *BaseSuite) Invalid(err error, msgAndArgs ...any) bool {
	s.T().Helper()
	return s.Refused(err, CodeValidation, msgAndArgs...)
}

// RefusedWithErrorAndCode asserts the refusal's body is exactly
// {"error": <message>, "code": <code>}: no other fields, a message present.
func (s *BaseSuite) RefusedWithErrorAndCode(err error, code string) bool {
	s.T().Helper()
	var ref *Refusal
	if !errors.As(err, &ref) {
		return s.Fail("expected a refusal", "got %v", err)
	}
	var body map[string]any
	if !s.NoError(json.Unmarshal(ref.Body, &body), "the body is JSON: %s", ref.Body) {
		return false
	}
	ok := s.Len(body, 2, "the body has exactly error and code: %s", ref.Body)
	ok = s.Equal(code, body["code"], "code: %s", ref.Body) && ok
	msg, isString := body["error"].(string)
	return s.True(isString && msg != "", "error is a message: %s", ref.Body) && ok
}

// ErrNothingListed is a list that showed the caller nothing: how a list
// refuses (an outsider gets an empty list, not 403).
var ErrNothingListed = errors.New("nothing listed")

// NothingListed is ErrNothingListed for an empty list, else err.
func NothingListed[T any](items []T, err error) error {
	if err == nil && len(items) == 0 {
		return ErrNothingListed
	}
	return err
}

// DeniedAccess says whether err is the API refusing access: no sign-in
// (unauthorized), not for this caller (forbidden), not there for them
// (not_found; the engine's denied and unknown for a session), or a list
// that showed nothing. A server error or a refused
// input is no denial.
func DeniedAccess(err error) bool {
	if errors.Is(err, ErrNothingListed) {
		return true
	}
	var ref *Refusal
	if !errors.As(err, &ref) {
		return false
	}
	switch ref.Code {
	case CodeUnauthorized, CodeForbidden, CodeNotFound, CodeDenied, CodeUnknown:
		return true
	}
	return false
}
