// package: obj / application error types
// type:    data
// job:     defines AppError, frontend-facing error codes, and typed error constructors
// limits:  error values only; HTTP mapping lives elsewhere (-> obj http_error, api/httpx)
package obj

import "fmt"

// Common error codes that frontend can handle
const (
	ErrCodeGeneric                   = "error"
	ErrCodeValidation                = "validation_error"
	ErrCodeUnauthorized              = "unauthorized"
	ErrCodeForbidden                 = "forbidden"
	ErrCodeNotFound                  = "not_found"
	ErrCodeConflict                  = "conflict"
	ErrCodeInvalidPlatform           = "invalid_platform"
	ErrCodeInvalidInput              = "invalid_input"
	ErrCodeServerError               = "server_error"
	ErrCodeUserNotRegistered         = "user_not_registered"
	ErrCodeDuplicateName             = "duplicate_name"
	ErrCodeNameTooLong               = "name_too_long"
	ErrCodeProfaneName               = "profane_name"
	ErrCodeNoApiKey                  = "no_api_key"
	ErrCodeSponsoredApiKeyNotWorking = "sponsored_api_key_not_working"
	ErrCodeLastHead                  = "last_head"

	// AI-specific error codes
	ErrCodeAiError                  = "ai_error"
	ErrCodeInvalidJsonSchema        = "invalid_json_schema"
	ErrCodeInvalidApiKey            = "invalid_api_key"
	ErrCodeBillingNotActive         = "billing_not_active"
	ErrCodeOrgVerificationRequired  = "organization_verification_required"
	ErrCodeRateLimitExceeded        = "rate_limit_exceeded"
	ErrCodeInsufficientQuota        = "insufficient_quota"
	ErrCodeContentFiltered          = "content_filtered"
	ErrCodePreviousResponseNotFound = "previous_response_not_found"
)

// Error type constructors

// ErrValidation returns a validation AppError with the given message.
func ErrValidation(message string) *AppError {
	return NewAppError(ErrCodeValidation, message)
}

// ErrValidationf returns a validation AppError with a formatted message.
func ErrValidationf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeValidation, fmt.Sprintf(format, args...))
}

// ErrUnauthorized returns an unauthorized AppError with the given message.
func ErrUnauthorized(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message)
}

// ErrForbidden returns a forbidden AppError with the given message.
func ErrForbidden(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message)
}

// ErrNotFound returns a not-found AppError with the given message.
func ErrNotFound(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message)
}

// ErrConflict returns a conflict AppError with the given message.
func ErrConflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, message)
}

// ErrInvalidPlatformf returns an invalid-platform AppError with a formatted message.
func ErrInvalidPlatformf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeInvalidPlatform, fmt.Sprintf(format, args...))
}

// ErrInvalidInput returns an invalid-input AppError with the given message.
func ErrInvalidInput(message string) *AppError {
	return NewAppError(ErrCodeInvalidInput, message)
}

// ErrServerError returns a server-error AppError with the given message.
func ErrServerError(message string) *AppError {
	return NewAppError(ErrCodeServerError, message)
}

// ErrServerErrorf returns a server-error AppError with a formatted message.
func ErrServerErrorf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeServerError, fmt.Sprintf(format, args...))
}

// ErrDuplicateNamef returns a duplicate-name AppError with a formatted message.
func ErrDuplicateNamef(format string, args ...any) *AppError {
	return NewAppError(ErrCodeDuplicateName, fmt.Sprintf(format, args...))
}

// ErrNameTooLong returns a name-too-long AppError with the given message.
func ErrNameTooLong(message string) *AppError {
	return NewAppError(ErrCodeNameTooLong, message)
}

// ErrProfaneName returns a profane-name AppError with the given message.
func ErrProfaneName(message string) *AppError {
	return NewAppError(ErrCodeProfaneName, message)
}

// ErrAiError returns an AI-error AppError with the given message.
func ErrAiError(message string) *AppError {
	return NewAppError(ErrCodeAiError, message)
}

// ErrAiErrorf returns an AI-error AppError with a formatted message.
func ErrAiErrorf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeAiError, fmt.Sprintf(format, args...))
}

// ErrInvalidApiKey returns an invalid-API-key AppError with the given message.
func ErrInvalidApiKey(message string) *AppError {
	return NewAppError(ErrCodeInvalidApiKey, message)
}

// AppError is a custom error type that carries an HTTP error code.
// It implements the standard error interface while providing additional context.
type AppError struct {
	Code    string // Machine-readable error code (e.g., "not_found", "unauthorized")
	Message string // Human-readable error message
	Err     error  // Optional underlying error for wrapping
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap implements the errors.Unwrap interface for error wrapping
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError with the given code and message
func NewAppError(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with an AppError
func WrapError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
