package testutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/obj"

	"github.com/google/uuid"
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

// UploadGame creates a new game and uploads YAML content from testdata/games (composable high-level API)
// Example: game := Must(alice.UploadGame("simple-quest"))
func (u *UserClient) UploadGame(yamlName string) (obj.Game, error) {
	u.t.Helper()

	// Read YAML file from testdata/games - try multiple paths
	relativePaths := []string{
		fmt.Sprintf("testdata/games/%s.yaml", yamlName),
		fmt.Sprintf("../testdata/games/%s.yaml", yamlName),
		fmt.Sprintf("testing/testdata/games/%s.yaml", yamlName),
	}

	var yamlContent []byte
	var err error
	var triedPaths []string

	for _, relPath := range relativePaths {
		absPath, _ := filepath.Abs(relPath)
		triedPaths = append(triedPaths, absPath)
		yamlContent, err = os.ReadFile(relPath)
		if err == nil {
			break
		}
	}

	if err != nil {
		return obj.Game{}, fmt.Errorf("failed to read game file %s.yaml, tried absolute paths: %v", yamlName, triedPaths)
	}

	// Make game name unique to avoid UNIQUE constraint conflicts across tests
	uniqueSuffix := uuid.New().String()[:8]
	uniqueName := fmt.Sprintf("Test Game %s", uniqueSuffix)

	// Create game with unique name
	var game obj.Game
	err = u.Post("games/new", routes.CreateGameRequest{
		Name: uniqueName,
	}, &game)
	if err != nil {
		return obj.Game{}, fmt.Errorf("failed to create game: %w", err)
	}

	// Replace the name in YAML content with the unique name (avoid UNIQUE constraint on upload)
	yamlStr := string(yamlContent)
	if idx := strings.Index(yamlStr, "name: "); idx != -1 {
		endIdx := strings.Index(yamlStr[idx:], "\n")
		if endIdx != -1 {
			yamlStr = yamlStr[:idx] + "name: " + uniqueName + yamlStr[idx+endIdx:]
		}
	}

	// Set user's token
	if err := client.SaveJwt(u.Token); err != nil {
		return obj.Game{}, fmt.Errorf("failed to set token for game upload: %w", err)
	}

	// Upload via PUT /games/{id}/yaml
	endpoint := fmt.Sprintf("games/%s/yaml", game.ID.String())
	if err := client.ApiPutRaw(endpoint, yamlStr); err != nil {
		return obj.Game{}, fmt.Errorf("failed to upload game YAML: %w", err)
	}

	// Fetch updated game to get YAML-populated fields
	var updatedGame obj.Game
	err = u.Get("games/"+game.ID.String(), &updatedGame)
	if err != nil {
		return obj.Game{}, fmt.Errorf("failed to fetch updated game: %w", err)
	}

	return updatedGame, nil
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

// UpdateUserName updates a user's name by ID (composable high-level API)
func (u *UserClient) UpdateUserName(userID string, name string) (obj.User, error) {
	u.t.Helper()
	var result obj.User
	payload := map[string]string{"name": name}
	err := u.Post("users/"+userID, payload, &result)
	return result, err
}

// SetUserLanguage sets the user's language preference (composable high-level API)
func (u *UserClient) SetUserLanguage(language string) error {
	u.t.Helper()
	return u.Patch("users/me/language", map[string]string{"language": language}, nil)
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

// AddApiKey reads an API key from a file and creates it (composable high-level API)
func (u *UserClient) AddApiKey(apiKey, name, platform string) (obj.ApiKeyShare, error) {
	u.t.Helper()

	var result obj.ApiKeyShare
	err := u.Post("apikeys/new", routes.CreateApiKeyRequest{
		Name:     name,
		Platform: platform,
		Key:      apiKey,
	}, &result)
	return result, err
}

// CreateGameSession creates a new game session (composable high-level API)
// Returns the session and the initial message.
// API key is resolved server-side (sponsor → workshop → user default).
func (u *UserClient) CreateGameSession(gameID string) (routes.SessionResponse, error) {
	u.t.Helper()

	var response routes.SessionResponse
	err := u.Post("games/"+gameID+"/sessions", nil, &response)
	return response, err
}

// GetGameSession loads a session with all messages (composable high-level API)
// Simulates a player returning to a session (e.g. browser reload).
func (u *UserClient) GetGameSession(sessionID string) (routes.SessionResponse, error) {
	u.t.Helper()
	var response routes.SessionResponse
	err := u.Get("sessions/"+sessionID+"?messages=all", &response)
	return response, err
}

// SendGameMessage sends a message to a game session and returns the AI response (composable high-level API)
// This returns the initial response with plot outline and status fields.
// Use SendGameMessageWithStream to also consume the full expanded story.
func (u *UserClient) SendGameMessage(sessionID string, message string) (obj.GameSessionMessage, error) {
	u.t.Helper()

	var response obj.GameSessionMessage
	err := u.Post("sessions/"+sessionID, routes.SessionActionRequest{
		Message: message,
	}, &response)
	return response, err
}

// SendGameMessageWithStream sends a message and consumes the SSE stream to get the full expanded story
func (u *UserClient) SendGameMessageWithStream(sessionID string, message string) (obj.GameSessionMessage, error) {
	u.t.Helper()

	// Get initial response with plot outline
	initialResponse, err := u.SendGameMessage(sessionID, message)
	if err != nil {
		return obj.GameSessionMessage{}, err
	}

	// If not streaming, return the initial response
	if !initialResponse.Stream {
		return initialResponse, nil
	}

	// Consume the SSE stream to get full expanded story
	fullStory, imageData, err := u.consumeMessageStream(initialResponse.ID.String())
	if err != nil {
		return obj.GameSessionMessage{}, fmt.Errorf("failed to consume stream: %w", err)
	}

	// Update response with full content
	initialResponse.Message = fullStory
	initialResponse.Image = imageData
	initialResponse.Stream = false

	return initialResponse, nil
}

// consumeMessageStream connects to SSE endpoint and consumes all chunks
func (u *UserClient) consumeMessageStream(messageID string) (string, []byte, error) {
	u.t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return "", nil, fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/messages/%s/stream", serverURL, messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Authorization", "Bearer "+u.Token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("stream request failed with status %d", resp.StatusCode)
	}

	var fullText strings.Builder
	var imageData []byte
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		var chunk obj.GameSessionMessageChunk
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			return "", nil, fmt.Errorf("failed to parse chunk: %w", err)
		}

		// Accumulate text
		if chunk.Text != "" {
			fullText.WriteString(chunk.Text)
		}

		// Accumulate image data
		if len(chunk.ImageData) > 0 {
			imageData = append(imageData, chunk.ImageData...)
		}

		// Check for completion or error
		if chunk.Error != "" {
			return "", nil, fmt.Errorf("stream error: %s", chunk.Error)
		}

		if chunk.TextDone && chunk.ImageDone {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("scanner error: %w", err)
	}

	return fullText.String(), imageData, nil
}

// RemoveUserRole removes a user's role (composable high-level API)
func (u *UserClient) RemoveUserRole(userID string) error {
	u.t.Helper()
	return u.Delete("users/" + userID + "/role")
}

// GetSystemSettings returns the global system settings (composable high-level API)
func (u *UserClient) GetSystemSettings() (obj.SystemSettings, error) {
	u.t.Helper()
	var result obj.SystemSettings
	err := u.Get("system/settings", &result)
	return result, err
}

// SetSystemFreeUseApiKey sets or clears the global free-use API key (composable high-level API)
// Pass nil to clear the free-use key.
func (u *UserClient) SetSystemFreeUseApiKey(apiKeyID *string) (obj.SystemSettings, error) {
	u.t.Helper()
	var payload interface{}
	if apiKeyID != nil {
		id, err := uuid.Parse(*apiKeyID)
		if err != nil {
			return obj.SystemSettings{}, fmt.Errorf("invalid apiKeyID: %w", err)
		}
		payload = routes.SetFreeUseApiKeyRequest{ApiKeyID: &id}
	} else {
		payload = routes.SetFreeUseApiKeyRequest{ApiKeyID: nil}
	}
	var result obj.SystemSettings
	err := u.Patch("system/settings/free-use-key", payload, &result)
	return result, err
}

// DeleteApiKey deletes an API key share, optionally cascading to delete the key and all shares (composable high-level API)
func (u *UserClient) DeleteApiKey(shareID string, cascade bool) error {
	u.t.Helper()
	endpoint := "apikeys/" + shareID
	if cascade {
		endpoint += "?cascade=true"
	}
	return u.Delete(endpoint)
}

// GetApiKeys returns the user's API keys and all their linked shares (composable high-level API)
func (u *UserClient) GetApiKeys() (routes.ApiKeysResponse, error) {
	u.t.Helper()
	var result routes.ApiKeysResponse
	err := u.Get("apikeys", &result)
	return result, err
}

// SetDefaultApiKey sets the given API key share as the user's default (composable high-level API)
func (u *UserClient) SetDefaultApiKey(shareID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	var result obj.ApiKeyShare
	err := u.Put("apikeys/"+shareID+"/default", nil, &result)
	return result, err
}

// SetInstitutionFreeUseApiKey sets or clears the free-use API key share for an institution (composable high-level API)
// Pass nil to clear.
func (u *UserClient) SetInstitutionFreeUseApiKey(institutionID string, shareID *string) (obj.Institution, error) {
	u.t.Helper()
	var sid *uuid.UUID
	if shareID != nil {
		parsed, err := uuid.Parse(*shareID)
		if err != nil {
			return obj.Institution{}, fmt.Errorf("invalid shareID: %w", err)
		}
		sid = &parsed
	}
	var result obj.Institution
	err := u.Patch("institutions/"+institutionID+"/free-use-key", map[string]interface{}{
		"shareId": sid,
	}, &result)
	return result, err
}

// ShareApiKeyWithInstitution shares an API key with an institution (composable high-level API)
func (u *UserClient) ShareApiKeyWithInstitution(shareID string, institutionID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	instID, err := uuid.Parse(institutionID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("invalid institutionID: %w", err)
	}
	var result obj.ApiKeyShare
	err = u.Post("apikeys/"+shareID+"/shares", routes.ShareRequest{
		InstitutionID: &instID,
	}, &result)
	return result, err
}

// SetWorkshopApiKey sets (or clears) the default API key for a workshop (composable high-level API)
func (u *UserClient) SetWorkshopApiKey(workshopID string, apiKeyShareID *string) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Put("workshops/"+workshopID+"/api-key", routes.SetWorkshopApiKeyRequest{
		ApiKeyShareID: apiKeyShareID,
	}, &result)
	return result, err
}

// SetActiveWorkshop sets the user's active workshop context (composable high-level API)
// Pass nil to leave workshop mode.
func (u *UserClient) SetActiveWorkshop(workshopID *string) (obj.User, error) {
	u.t.Helper()
	var wsID *uuid.UUID
	if workshopID != nil {
		parsed, err := uuid.Parse(*workshopID)
		if err != nil {
			return obj.User{}, fmt.Errorf("invalid workshopID: %w", err)
		}
		wsID = &parsed
	}
	var result obj.User
	err := u.Put("users/me/active-workshop", map[string]interface{}{
		"workshopId": wsID,
	}, &result)
	return result, err
}

// GetApiKeyStatus checks whether an API key can be resolved for a game (composable high-level API)
func (u *UserClient) GetApiKeyStatus(gameID string) (bool, error) {
	u.t.Helper()
	var result map[string]bool
	err := u.Get("games/"+gameID+"/api-key-status", &result)
	if err != nil {
		return false, err
	}
	return result["available"], nil
}

// ShareApiKeyWithWorkshop shares an API key with a workshop (composable high-level API)
func (u *UserClient) ShareApiKeyWithWorkshop(shareID string, workshopID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	wsID, err := uuid.Parse(workshopID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("invalid workshopID: %w", err)
	}
	var result obj.ApiKeyShare
	err = u.Post("apikeys/"+shareID+"/shares", routes.ShareRequest{
		WorkshopID: &wsID,
	}, &result)
	return result, err
}

// DeleteGame deletes a game by ID (composable high-level API)
func (u *UserClient) DeleteGame(gameID string) error {
	u.t.Helper()
	return u.Delete("games/" + gameID)
}

// UpdateGame updates a game's properties (composable high-level API)
func (u *UserClient) UpdateGame(gameID string, updates map[string]interface{}) (obj.Game, error) {
	u.t.Helper()
	var result obj.Game
	err := u.Post("games/"+gameID, updates, &result)
	return result, err
}

// GetGameByID returns a game by ID (composable high-level API)
func (u *UserClient) GetGameByID(gameID string) (obj.Game, error) {
	u.t.Helper()
	var result obj.Game
	err := u.Get("games/"+gameID, &result)
	return result, err
}

// ListGames returns all games visible to the user (composable high-level API)
func (u *UserClient) ListGames() ([]obj.Game, error) {
	u.t.Helper()
	var result []obj.Game
	err := u.Get("games", &result)
	return result, err
}

// ListInstitutions returns all institutions visible to the user (composable high-level API)
func (u *UserClient) ListInstitutions() ([]obj.Institution, error) {
	u.t.Helper()
	var result []obj.Institution
	err := u.Get("institutions", &result)
	return result, err
}

// DeleteInstitution deletes an institution by ID (composable high-level API)
func (u *UserClient) DeleteInstitution(institutionID string) error {
	u.t.Helper()
	return u.Delete("institutions/" + institutionID)
}

// DeleteWorkshop deletes a workshop by ID (composable high-level API)
func (u *UserClient) DeleteWorkshop(workshopID string) error {
	u.t.Helper()
	return u.Delete("workshops/" + workshopID)
}

// GetInstitutionApiKeys returns API keys shared with an institution (composable high-level API)
func (u *UserClient) GetInstitutionApiKeys(institutionID string) ([]obj.ApiKeyShare, error) {
	u.t.Helper()
	var result []obj.ApiKeyShare
	err := u.Get("institutions/"+institutionID+"/apikeys", &result)
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
