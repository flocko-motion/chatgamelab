// package: testutil / HTTP test client
// type:    logic
// job:     provides authenticated and public test clients wrapping the API for use in tests
// limits:  test-only HTTP helpers; no suite lifecycle management (-> testutil suite)
package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"cgl/config"
)

var (
	TestServerURL = "http://localhost:7102" // Default, will be overridden by suite
)

var (
	testServerInit sync.Once
)

// UserClient represents a test user with their own authentication context
type UserClient struct {
	Name  string
	ID    string
	Email string
	Token string
	t     *testing.T
}

// PublicClient represents an unauthenticated client
type PublicClient struct {
	t *testing.T
}

// initTestServer ensures the test server URL is configured
func initTestServer(t *testing.T) {
	testServerInit.Do(func() {
		if err := config.SetServerConfig(TestServerURL, ""); err != nil {
			t.Fatalf("failed to set test server URL: %v", err)
		}
	})
}

// Public returns a public (unauthenticated) client for testing
func Public(t *testing.T) *PublicClient {
	t.Helper()
	initTestServer(t)
	return &PublicClient{t: t}
}

// ErrorValidator is a function that validates an error
type ErrorValidator func(error) bool

// ErrorPrefix returns a validator that checks if error message starts with prefix
//
//deadcode:keep // companion to ErrorContains in the test-helper API, kept for tests that assert on error prefixes
func ErrorPrefix(prefix string) ErrorValidator {
	return func(err error) bool {
		if err == nil {
			return false
		}
		return strings.HasPrefix(err.Error(), prefix)
	}
}

// ErrorContains returns a validator that checks if error message contains substring
func ErrorContains(substring string) ErrorValidator {
	return func(err error) bool {
		if err == nil {
			return false
		}
		return strings.Contains(err.Error(), substring)
	}
}

// validateError checks if error exists and validates it with provided validators
// Returns true if validation passes, false otherwise
func validateError(t *testing.T, err error, context string, validators ...ErrorValidator) bool {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error but got none", context)
		return false
	}

	// If no validators provided, just accept any error
	if len(validators) == 0 {
		t.Logf("%s: got expected error: %v", context, err)
		return true
	}

	// Validate with all provided validators
	for _, validator := range validators {
		if !validator(err) {
			t.Fatalf("%s: error validation failed: %v", context, err)
			return false
		}
	}

	t.Logf("%s: got expected error: %v", context, err)
	return true
}

// --- UserClient request primitives ---

// makeRequest performs an HTTP request with the user's token in the Authorization header
// This bypasses the config storage system (which is only for CLI usage)
func (u *UserClient) makeRequest(method, endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/%s", serverURL, strings.TrimPrefix(endpoint, "/"))

	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set Authorization header directly without touching config storage
	if u.Token != "" {
		req.Header.Set("Authorization", "Bearer "+u.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Get performs an authenticated GET request
func (u *UserClient) Get(endpoint string, out interface{}) error {
	u.t.Helper()
	return u.makeRequest("GET", endpoint, nil, out)
}

// Post performs an authenticated POST request
func (u *UserClient) Post(endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()
	return u.makeRequest("POST", endpoint, payload, out)
}

// Patch performs an authenticated PATCH request
func (u *UserClient) Patch(endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()
	return u.makeRequest("PATCH", endpoint, payload, out)
}

// Put performs an authenticated PUT request
func (u *UserClient) Put(endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()
	return u.makeRequest("PUT", endpoint, payload, out)
}

// Delete performs an authenticated DELETE request
func (u *UserClient) Delete(endpoint string) error {
	u.t.Helper()
	return u.makeRequest("DELETE", endpoint, nil, nil)
}
