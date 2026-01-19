package httpx

import (
	"bytes"
	"cgl/obj"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// Common error codes that frontend can handle
const (
	ErrCodeGeneric           = "error"
	ErrCodeValidation        = "validation_error"
	ErrCodeUnauthorized      = "unauthorized"
	ErrCodeForbidden         = "forbidden"
	ErrCodeNotFound          = "not_found"
	ErrCodeConflict          = "conflict"
	ErrCodeInvalidPlatform   = "invalid_platform"
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeServerError       = "server_error"
	ErrCodeUserNotRegistered = "user_not_registered"

	// AI-related error codes
	ErrCodeInvalidApiKey           = "invalid_api_key"
	ErrCodeOrgVerificationRequired = "org_verification_required"
	ErrCodeBillingNotActive        = "billing_not_active"
	ErrCodeAiError                 = "ai_error"
)

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
		Type:    ErrCodeUserNotRegistered,
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
	WriteErrorWithCode(w, status, ErrCodeGeneric, message)
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
		code = ErrCodeGeneric
	}
	WriteErrorWithCode(w, err.StatusCode, code, err.Message)
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
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// SetNoCacheHeaders sets headers to prevent caching
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
