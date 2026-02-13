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
	ErrCodeAiError                 = "ai_error"
	ErrCodeInvalidJsonSchema       = "invalid_json_schema"
	ErrCodeInvalidApiKey           = "invalid_api_key"
	ErrCodeBillingNotActive        = "billing_not_active"
	ErrCodeOrgVerificationRequired = "organization_verification_required"
	ErrCodeRateLimitExceeded       = "rate_limit_exceeded"
	ErrCodeInsufficientQuota       = "insufficient_quota"
	ErrCodeContentFiltered         = "content_filtered"
)

// Error type constructors

func ErrValidation(message string) *AppError {
	return NewAppError(ErrCodeValidation, message)
}

func ErrValidationf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeValidation, fmt.Sprintf(format, args...))
}

func ErrUnauthorized(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message)
}

func ErrUnauthorizedf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeUnauthorized, fmt.Sprintf(format, args...))
}

func ErrForbidden(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message)
}

func ErrForbiddenf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeForbidden, fmt.Sprintf(format, args...))
}

func ErrNotFound(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message)
}

func ErrNotFoundf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf(format, args...))
}

func ErrConflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, message)
}

func ErrConflictf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeConflict, fmt.Sprintf(format, args...))
}

func ErrInvalidPlatform(message string) *AppError {
	return NewAppError(ErrCodeInvalidPlatform, message)
}

func ErrInvalidPlatformf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeInvalidPlatform, fmt.Sprintf(format, args...))
}

func ErrInvalidInput(message string) *AppError {
	return NewAppError(ErrCodeInvalidInput, message)
}

func ErrInvalidInputf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeInvalidInput, fmt.Sprintf(format, args...))
}

func ErrServerError(message string) *AppError {
	return NewAppError(ErrCodeServerError, message)
}

func ErrServerErrorf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeServerError, fmt.Sprintf(format, args...))
}

func ErrUserNotRegistered(message string) *AppError {
	return NewAppError(ErrCodeUserNotRegistered, message)
}

func ErrUserNotRegisteredf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeUserNotRegistered, fmt.Sprintf(format, args...))
}

func ErrDuplicateName(message string) *AppError {
	return NewAppError(ErrCodeDuplicateName, message)
}

func ErrDuplicateNamef(format string, args ...any) *AppError {
	return NewAppError(ErrCodeDuplicateName, fmt.Sprintf(format, args...))
}

func ErrNameTooLong(message string) *AppError {
	return NewAppError(ErrCodeNameTooLong, message)
}

func ErrNameTooLongf(format string, args ...any) *AppError {
	return NewAppError(ErrCodeNameTooLong, fmt.Sprintf(format, args...))
}

func ErrProfaneName(message string) *AppError {
	return NewAppError(ErrCodeProfaneName, message)
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
