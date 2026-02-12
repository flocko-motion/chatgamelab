package httpx

import (
	"bytes"
	"cgl/obj"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// getCookiePath returns the path for cookies based on API_BASE_URL environment variable.
// Extracts the path component from API_BASE_URL (e.g., "/api" from "https://example.com/api").
// Defaults to "/api" if not set or no path component.
func getCookiePath() string {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		return "/api"
	}

	// Try to parse as URL to extract path
	if parsed, err := url.Parse(apiBaseURL); err == nil && parsed.Path != "" && parsed.Path != "/" {
		return strings.TrimSuffix(parsed.Path, "/")
	}

	// If it's just a path (starts with /), use it directly
	if strings.HasPrefix(apiBaseURL, "/") {
		return strings.TrimSuffix(apiBaseURL, "/")
	}

	return "/api"
}

// ErrorResponse is the standard error format for the API
type ErrorResponse struct {
	Code    string `json:"code"`    // Machine-readable error code
	Message string `json:"message"` // Human-readable error message
	Type    string `json:"type"`    // Deprecated: use Code instead
}

// UserNotRegisteredResponse is returned when an Auth0 user is not yet registered in the system
type UserNotRegisteredResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Auth0ID string `json:"auth0Id"`
}

// WriteUserNotRegistered writes a response indicating the user needs to complete registration
func WriteUserNotRegistered(w http.ResponseWriter, auth0ID string) {
	WriteJSON(w, http.StatusForbidden, UserNotRegisteredResponse{
		Type:    obj.ErrCodeUserNotRegistered,
		Message: "User is not registered. Please complete registration.",
		Auth0ID: auth0ID,
	})
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteYAML writes a YAML response with the given status code
func WriteYAML(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(status)
	_ = yaml.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response with a generic error code
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteErrorWithCode(w, status, obj.ErrCodeGeneric, message)
}

// WriteErrorWithCode writes a JSON error response with a specific error code
func WriteErrorWithCode(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
		Type:    code, // For backwards compatibility
	})
}

// WriteHTTPError writes a JSON error response from an HTTPError, using its Code if set
func WriteHTTPError(w http.ResponseWriter, err *obj.HTTPError) {
	code := err.Code
	if code == "" {
		code = obj.ErrCodeGeneric
	}
	WriteErrorWithCode(w, err.StatusCode, code, err.Message)
}

// ErrorCodeToStatus maps error codes to HTTP status codes
func ErrorCodeToStatus(code string) int {
	switch code {
	case obj.ErrCodeValidation, obj.ErrCodeInvalidInput, obj.ErrCodeInvalidPlatform, obj.ErrCodeNameTooLong, obj.ErrCodeProfaneName:
		return http.StatusBadRequest
	case obj.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case obj.ErrCodeForbidden, obj.ErrCodeUserNotRegistered:
		return http.StatusForbidden
	case obj.ErrCodeNotFound:
		return http.StatusNotFound
	case obj.ErrCodeConflict, obj.ErrCodeDuplicateName, obj.ErrCodeLastHead:
		return http.StatusConflict
	case obj.ErrCodeServerError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WriteAppError writes an AppError as a JSON response with the appropriate HTTP status code
func WriteAppError(w http.ResponseWriter, err *obj.AppError) {
	status := ErrorCodeToStatus(err.Code)
	WriteJSON(w, status, ErrorResponse{
		Code:    err.Code,
		Message: err.Message,
		Type:    err.Code, // For backwards compatibility
	})
}

// ReadJSON decodes the request body as JSON into the given struct
func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ReadYAML decodes the request body as YAML into the given struct
func ReadYAML(r *http.Request, v any) error {
	return yaml.NewDecoder(r.Body).Decode(v)
}

// ReadJSONOrYAML auto-detects content type and decodes accordingly
// Checks Content-Type header first, then tries to detect from content
func ReadJSONOrYAML(r *http.Request, v any) error {
	contentType := r.Header.Get("Content-Type")

	// If explicitly YAML, use YAML decoder
	if contentType == "application/x-yaml" || contentType == "text/yaml" {
		return ReadYAML(r, v)
	}

	// If explicitly JSON, use JSON decoder
	if strings.Contains(contentType, "application/json") {
		return ReadJSON(r, v)
	}

	// No explicit content type or unknown - read body and auto-detect
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	// Try JSON first (starts with { or [)
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return json.Unmarshal(body, v)
	}

	// Otherwise assume YAML
	return yaml.Unmarshal(body, v)
}

// SetCORSHeaders sets CORS headers for cross-origin requests
// Deprecated: Use SetCORSHeadersWithOrigin for credential support
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// SetCORSHeadersWithOrigin sets CORS headers with credential support
// When credentials are used, Access-Control-Allow-Origin cannot be "*"
func SetCORSHeadersWithOrigin(w http.ResponseWriter, origin string) {
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

// SetNoCacheHeaders sets headers to prevent caching
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// SetSessionCookie sets an HttpOnly cookie for participant sessions
// The cookie is used as a fallback authentication mechanism for workshop participants
func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	// Determine if we're in a secure context (HTTPS)
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     "cgl_session",
		Value:    token,
		Path:     getCookiePath(),
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie (for logout)
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "cgl_session",
		Value:    "",
		Path:     getCookiePath(),
		MaxAge:   -1,
		HttpOnly: true,
	})
}
