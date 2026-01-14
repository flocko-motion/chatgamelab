package httpx

import (
	"encoding/json"
	"net/http"

	"gopkg.in/yaml.v3"
)

// ErrorResponse is the standard error format for the API
type ErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
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
		Type:    "user_not_registered",
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

// WriteError writes a JSON error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{
		Type:    "error",
		Message: message,
	})
}

// ReadJSON decodes the request body as JSON into the given struct
func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
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
