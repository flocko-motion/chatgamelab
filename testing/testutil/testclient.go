package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/obj"
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

	if resp.StatusCode != http.StatusOK {
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

// Delete performs an authenticated DELETE request
func (u *UserClient) Delete(endpoint string) error {
	u.t.Helper()
	return u.makeRequest("DELETE", endpoint, nil, nil)
}

// GetInvitesIncoming returns the user's incoming invites (composable high-level API)
func (u *UserClient) GetInvitesIncoming() ([]obj.UserRoleInvite, error) {
	u.t.Helper()
	var invites []obj.UserRoleInvite
	err := u.Get("invites", &invites)
	return invites, err
}

// GetInvitesOutgoing returns all invites for a specific institution (composable high-level API)
func (u *UserClient) GetInvitesOutgoing(institutionID string) ([]obj.UserRoleInvite, error) {
	u.t.Helper()
	var invites []obj.UserRoleInvite
	err := u.Get("invites/institution/"+institutionID, &invites)
	return invites, err
}

// GetInvite returns a specific invite by ID (composable high-level API)
func (u *UserClient) GetInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var invite obj.UserRoleInvite
	err := u.Get("invites/"+inviteID, &invite)
	return invite, err
}

// GetInstitutions returns the user's institutions (composable high-level API)
func (u *UserClient) GetInstitutions() ([]obj.Institution, error) {
	u.t.Helper()
	var institutions []obj.Institution
	err := u.Get("institutions", &institutions)
	return institutions, err
}

// AcceptInvite accepts an invite by ID (composable high-level API)
func (u *UserClient) AcceptInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	if err := u.Post("invites/"+inviteID+"/accept", nil, nil); err != nil {
		return obj.UserRoleInvite{}, err
	}
	return u.GetInvite(inviteID)
}

// DeclineInvite declines an invite by ID (composable high-level API)
func (u *UserClient) DeclineInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	if err := u.Post("invites/"+inviteID+"/decline", nil, nil); err != nil {
		return obj.UserRoleInvite{}, err
	}
	return u.GetInvite(inviteID)
}

// RevokeInvite revokes an invite by ID (composable high-level API)
func (u *UserClient) RevokeInvite(inviteID string) error {
	u.t.Helper()
	return u.Delete("invites/" + inviteID)
}

// CreateInstitution creates an institution (composable high-level API)
func (u *UserClient) CreateInstitution(name string) (obj.Institution, error) {
	u.t.Helper()
	var result obj.Institution
	payload := routes.CreateInstitutionRequest{
		Name: name,
	}
	err := u.Post("institutions", payload, &result)
	return result, err
}

// InviteToInstitution creates an institution invite by user ID (composable high-level API)
func (u *UserClient) InviteToInstitution(institutionID, role, invitedUserID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var result obj.UserRoleInvite
	payload := routes.CreateInstitutionInviteRequest{
		InstitutionID: institutionID,
		Role:          role,
		InvitedUserID: &invitedUserID,
	}
	err := u.Post("invites/institution", payload, &result)
	return result, err
}

// InviteToInstitutionByEmail creates an institution invite by email (composable high-level API)
func (u *UserClient) InviteToInstitutionByEmail(institutionID, role, invitedEmail string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var result obj.UserRoleInvite
	payload := routes.CreateInstitutionInviteRequest{
		InstitutionID: institutionID,
		Role:          role,
		InvitedEmail:  &invitedEmail,
	}
	err := u.Post("invites/institution", payload, &result)
	return result, err
}

func (u *UserClient) GetRole() string {
	u.t.Helper()
	me, err := u.GetMe()
	if err != nil {
		u.t.Fatalf("User %q: failed to get me: %v", u.Name, err)
	}
	if me.Role == nil {
		u.t.Fatalf("User %q: no role", u.Name)
	}
	return string(me.Role.Role)
}

// GetMe returns the current user's profile (composable high-level API)
func (u *UserClient) GetMe() (obj.User, error) {
	u.t.Helper()
	var result obj.User
	err := u.Get("users/me", &result)
	return result, err
}

// GetInstitution returns a specific institution by ID (composable high-level API)
func (u *UserClient) GetInstitution(institutionID string) (obj.Institution, error) {
	u.t.Helper()
	var result obj.Institution
	err := u.Get("institutions/"+institutionID, &result)
	return result, err
}

// GetUsers returns all users (composable high-level API)
func (u *UserClient) GetUsers() ([]obj.User, error) {
	u.t.Helper()
	var result []obj.User
	err := u.Get("users", &result)
	return result, err
}

// RemoveMember removes a member from an institution (composable high-level API)
func (u *UserClient) RemoveMember(institutionID, userID string) error {
	u.t.Helper()
	return u.Delete("institutions/" + institutionID + "/members/" + userID)
}

// CreateWorkshop creates a new workshop (composable high-level API)
func (u *UserClient) CreateWorkshop(institutionID, name string) (obj.Workshop, error) {
	u.t.Helper()
	payload := map[string]interface{}{
		"institutionId": institutionID,
		"name":          name,
		"active":        true,
		"public":        false,
	}
	var result obj.Workshop
	err := u.Post("workshops", payload, &result)
	return result, err
}

// UpdateWorkshop updates a workshop (composable high-level API)
func (u *UserClient) UpdateWorkshop(workshopID string, updates map[string]interface{}) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Patch("workshops/"+workshopID, updates, &result)
	return result, err
}

// GetWorkshop retrieves a workshop by ID (composable high-level API)
func (u *UserClient) GetWorkshop(workshopID string) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Get("workshops/"+workshopID, &result)
	return result, err
}

// ListWorkshops lists workshops for an institution (composable high-level API)
func (u *UserClient) ListWorkshops(institutionID string) ([]obj.Workshop, error) {
	u.t.Helper()
	var result []obj.Workshop
	err := u.Get("workshops?institutionId="+institutionID, &result)
	return result, err
}

// GetParticipantToken retrieves the access token for a workshop participant (composable high-level API)
func (u *UserClient) GetParticipantToken(participantID string) (*string, error) {
	u.t.Helper()
	var result map[string]string
	err := u.Get("workshops/participants/"+participantID+"/token", &result)
	if err != nil {
		return nil, err
	}
	token := result["token"]
	return &token, nil
}

// CreateWorkshopInvite creates a workshop invite (composable high-level API)
func (u *UserClient) CreateWorkshopInvite(workshopID, role string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	payload := map[string]interface{}{
		"workshopId": workshopID,
		"role":       role,
	}
	var result obj.UserRoleInvite
	err := u.Post("invites/workshop", payload, &result)
	return result, err
}

// ReactivateInvite reactivates a revoked invite (composable high-level API)
func (u *UserClient) ReactivateInvite(inviteID string) error {
	u.t.Helper()
	return u.Post("invites/"+inviteID+"/reactivate", nil, nil)
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
