package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"cgl/api/client"
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

// --- UserClient API methods ---

// UploadGame uploads a game YAML file from testdata/games to an existing game
// Example: alice.UploadGame(gameID, "simple-quest")
func (u *UserClient) UploadGame(gameID, name string) {
	u.t.Helper()

	// Read YAML file from testdata/games
	yamlPath := fmt.Sprintf("../testdata/games/%s.yaml", name)
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		u.t.Fatalf("User %q: failed to read game file %s: %v", u.Name, yamlPath, err)
	}

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		u.t.Fatalf("User %q: failed to set token for game upload: %v", u.Name, err)
	}

	// Upload via PUT /games/{id}/yaml
	endpoint := fmt.Sprintf("games/%s/yaml", gameID)
	if err := client.ApiPutRaw(endpoint, string(yamlContent)); err != nil {
		u.t.Fatalf("User %q: failed to upload game %s: %v", u.Name, name, err)
	}

	u.t.Logf("User %q uploaded game %s to %s", u.Name, name, gameID)
}

// Get performs an authenticated GET request
func (u *UserClient) Get(endpoint string, out interface{}) error {
	u.t.Helper()

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		return fmt.Errorf("failed to set token: %w", err)
	}

	return client.ApiGet(endpoint, out)
}

// Post performs an authenticated POST request
func (u *UserClient) Post(endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		return fmt.Errorf("failed to set token: %w", err)
	}

	return client.ApiPost(endpoint, payload, out)
}

// Patch performs an authenticated PATCH request
func (u *UserClient) Patch(endpoint string, payload interface{}, out interface{}) error {
	u.t.Helper()

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		return fmt.Errorf("failed to set token: %w", err)
	}

	return client.ApiPatch(endpoint, payload, out)
}

// Delete performs an authenticated DELETE request
func (u *UserClient) Delete(endpoint string) error {
	u.t.Helper()

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		return fmt.Errorf("failed to set token: %w", err)
	}

	return client.ApiDelete(endpoint)
}

// MustGet performs GET and fails test on error
func (u *UserClient) MustGet(endpoint string, out interface{}) {
	u.t.Helper()
	if err := u.Get(endpoint, out); err != nil {
		u.t.Fatalf("User %q GET %s failed: %v", u.Name, endpoint, err)
	}
}

// MustPost performs POST and fails test on error
func (u *UserClient) MustPost(endpoint string, payload interface{}, out interface{}) {
	u.t.Helper()
	if err := u.Post(endpoint, payload, out); err != nil {
		u.t.Fatalf("User %q POST %s failed: %v", u.Name, endpoint, err)
	}
}

// MustPatch performs PATCH and fails test on error
func (u *UserClient) MustPatch(endpoint string, payload interface{}, out interface{}) {
	u.t.Helper()
	if err := u.Patch(endpoint, payload, out); err != nil {
		u.t.Fatalf("User %q PATCH %s failed: %v", u.Name, endpoint, err)
	}
}

// MustDelete performs DELETE and fails test on error
func (u *UserClient) MustDelete(endpoint string) {
	u.t.Helper()
	if err := u.Delete(endpoint); err != nil {
		u.t.Fatalf("User %q DELETE %s failed: %v", u.Name, endpoint, err)
	}
}

// FailGet expects GET to fail and validates the error
func (u *UserClient) FailGet(endpoint string, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Get(endpoint, nil)
	if err == nil {
		u.t.Fatalf("User %q GET %s: expected error but got none", u.Name, endpoint)
	}

	if len(validators) == 0 {
		u.t.Logf("User %q GET %s: got expected error: %v", u.Name, endpoint, err)
		return
	}

	for _, validator := range validators {
		if !validator(err) {
			u.t.Fatalf("User %q GET %s: error validation failed: %v", u.Name, endpoint, err)
		}
	}
	u.t.Logf("User %q GET %s: got expected error: %v", u.Name, endpoint, err)
}

// FailPost expects POST to fail and validates the error
func (u *UserClient) FailPost(endpoint string, payload interface{}, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Post(endpoint, payload, nil)
	validateError(u.t, err, fmt.Sprintf("User %q POST %s", u.Name, endpoint), validators...)
}

// FailPatch expects PATCH to fail and validates the error
func (u *UserClient) FailPatch(endpoint string, payload interface{}, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Patch(endpoint, payload, nil)
	validateError(u.t, err, fmt.Sprintf("User %q PATCH %s", u.Name, endpoint), validators...)
}

// FailDelete expects DELETE to fail and validates the error
func (u *UserClient) FailDelete(endpoint string, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Delete(endpoint)
	validateError(u.t, err, fmt.Sprintf("User %q DELETE %s", u.Name, endpoint), validators...)
}

// --- PublicClient API methods ---

// Get performs an unauthenticated GET request
func (p *PublicClient) Get(endpoint string, out interface{}) error {
	p.t.Helper()

	// Clear auth
	if err := config.SetServerConfig(TestServerURL, ""); err != nil {
		return fmt.Errorf("failed to clear auth: %w", err)
	}

	return client.ApiGet(endpoint, out)
}

// Post performs an unauthenticated POST request
func (p *PublicClient) Post(endpoint string, payload interface{}, out interface{}) error {
	p.t.Helper()

	// Clear auth
	if err := config.SetServerConfig(TestServerURL, ""); err != nil {
		return fmt.Errorf("failed to clear auth: %w", err)
	}

	return client.ApiPost(endpoint, payload, out)
}

// MustGet performs GET and fails test on error
func (p *PublicClient) MustGet(endpoint string, out interface{}) {
	p.t.Helper()
	if err := p.Get(endpoint, out); err != nil {
		p.t.Fatalf("Public GET %s failed: %v", endpoint, err)
	}
}

// MustPost performs POST and fails test on error
func (p *PublicClient) MustPost(endpoint string, payload interface{}, out interface{}) {
	p.t.Helper()
	if err := p.Post(endpoint, payload, out); err != nil {
		p.t.Fatalf("Public POST %s failed: %v", endpoint, err)
	}
}

// FailGet expects GET to fail and validates the error
func (p *PublicClient) FailGet(endpoint string, validators ...ErrorValidator) {
	p.t.Helper()
	err := p.Get(endpoint, nil)
	validateError(p.t, err, fmt.Sprintf("Public GET %s", endpoint), validators...)
}

// FailPost expects POST to fail and validates the error
func (p *PublicClient) FailPost(endpoint string, payload interface{}, validators ...ErrorValidator) {
	p.t.Helper()
	err := p.Post(endpoint, payload, nil)
	validateError(p.t, err, fmt.Sprintf("Public POST %s", endpoint), validators...)
}

// --- Helper utilities ---

// PrintJSON prints a value as formatted JSON for debugging
func PrintJSON(t *testing.T, label string, v interface{}) {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Logf("%s: (marshal error: %v)", label, err)
		return
	}
	t.Logf("%s: %s", label, string(data))
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()

	if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEmpty checks if a value is not empty
func AssertNotEmpty(t *testing.T, value interface{}, msg string) {
	t.Helper()

	if fmt.Sprintf("%v", value) == "" || fmt.Sprintf("%v", value) == "<nil>" {
		t.Errorf("%s: value is empty", msg)
	}
}
