// package: testutil / game API helpers
// type:    logic
// job:     high-level UserClient helpers for games, sessions, sponsors and shares
// limits:  game/session HTTP helpers only; no invites/orgs (-> testclient_orgs) or streaming (-> testclient_streaming)
package testutil

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/obj"

	"github.com/google/uuid"
)

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

// CreateGameSession creates a new game session (composable high-level API)
// Returns the session and the initial message.
// API key is resolved server-side (sponsor → workshop → user default).
func (u *UserClient) CreateGameSession(gameID string) (routes.SessionResponse, error) {
	u.t.Helper()

	var response routes.SessionResponse
	err := u.Post("games/"+gameID+"/sessions", nil, &response)
	return response, err
}

// CreateGameSessionWithStream creates a session and consumes the SSE stream for the initial message.
// TWO-PHASE INITIALIZATION: Creates session (phase 1), then sends empty action to trigger opening scene (phase 2).
// Returns the session response with the initial message fully populated (text, image, audio).
func (u *UserClient) CreateGameSessionWithStream(gameID string) (routes.SessionResponse, *StreamResult, error) {
	u.t.Helper()

	// PHASE 1: Create session (returns empty messages array)
	resp, err := u.CreateGameSession(gameID)
	if err != nil {
		return routes.SessionResponse{}, nil, err
	}

	if resp.GameSession == nil {
		return routes.SessionResponse{}, nil, fmt.Errorf("no session returned")
	}

	// PHASE 2: Send "init" system action to trigger opening scene generation
	openingMsg, streamResult, err := u.SendSystemMessage(resp.GameSession.ID.String(), "init")
	if err != nil {
		return routes.SessionResponse{}, nil, fmt.Errorf("failed to trigger opening scene: %w", err)
	}

	// Add the opening message to the response
	resp.Messages = []obj.GameSessionMessage{openingMsg}

	return resp, streamResult, nil
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

// DeleteGame deletes a game by ID (composable high-level API)
func (u *UserClient) DeleteGame(gameID string) error {
	u.t.Helper()
	return u.Delete("games/" + gameID)
}

// UpdateGame updates a game's properties (composable high-level API)
// Accepts either a map[string]interface{} for partial updates or a full obj.Game struct.
func (u *UserClient) UpdateGame(gameID string, updates interface{}) (obj.Game, error) {
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

// SetGameSponsor sets a public sponsorship on a game using an API key share (composable high-level API)
// Returns the updated game with the sponsor share ID set.
func (u *UserClient) SetGameSponsor(gameID string, shareID string) (obj.Game, error) {
	u.t.Helper()
	shareUUID, err := uuid.Parse(shareID)
	if err != nil {
		return obj.Game{}, fmt.Errorf("invalid shareID: %w", err)
	}
	var result obj.Game
	err = u.Put("games/"+gameID+"/sponsor", routes.SponsorGameRequest{
		ShareID: shareUUID,
	}, &result)
	return result, err
}

// RemoveGameSponsor removes the public sponsorship from a game (composable high-level API)
func (u *UserClient) RemoveGameSponsor(gameID string) (obj.Game, error) {
	u.t.Helper()
	var result obj.Game
	err := u.makeRequest("DELETE", "games/"+gameID+"/sponsor", nil, &result)
	return result, err
}

// CloneGame clones a game by ID (composable high-level API)
func (u *UserClient) CloneGame(gameID string) (obj.Game, error) {
	u.t.Helper()
	var result obj.Game
	err := u.Post("games/"+gameID+"/clone", nil, &result)
	return result, err
}

// GetGameYAML exports a game as YAML (composable high-level API)
func (u *UserClient) GetGameYAML(gameID string) (string, error) {
	u.t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return "", fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/games/%s/yaml", serverURL, gameID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+u.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// ListGamesWithFilter lists games with a filter parameter (composable high-level API)
func (u *UserClient) ListGamesWithFilter(filter string) ([]obj.Game, error) {
	u.t.Helper()
	var result []obj.Game
	err := u.Get("games?filter="+filter, &result)
	return result, err
}

// ListGamesWithSearch lists games with a search query (composable high-level API)
func (u *UserClient) ListGamesWithSearch(search string) ([]obj.Game, error) {
	u.t.Helper()
	var result []obj.Game
	err := u.Get("games?search="+search, &result)
	return result, err
}

// CreateGameShare creates a personal/org share for a game (composable high-level API)
func (u *UserClient) CreateGameShare(gameID string, sponsorShareID string, maxSessions *int) (routes.GameShareResponse, error) {
	u.t.Helper()
	shareUUID, err := uuid.Parse(sponsorShareID)
	if err != nil {
		return routes.GameShareResponse{}, fmt.Errorf("invalid sponsorShareID: %w", err)
	}
	var result routes.GameShareResponse
	err = u.Post("games/"+gameID+"/shares", routes.CreateGameShareRequest{
		SponsorKeyShareID: &shareUUID,
		MaxSessions:       maxSessions,
	}, &result)
	return result, err
}

// GetPrivateShareStatus returns the private share status for a game (composable high-level API)
func (u *UserClient) GetPrivateShareStatus(gameID string) (routes.PrivateShareStatus, error) {
	u.t.Helper()
	var result routes.PrivateShareStatus
	err := u.Get("games/"+gameID+"/private-share", &result)
	return result, err
}

// DeleteGameShare deletes a specific game share by ID (composable high-level API)
func (u *UserClient) DeleteGameShare(gameID string, shareID string) error {
	u.t.Helper()
	return u.makeRequest("DELETE", "games/"+gameID+"/shares/"+shareID, nil, nil)
}

// CreateWorkshopGameShare creates a workshop share for a game (composable high-level API)
func (u *UserClient) CreateWorkshopGameShare(gameID string, workshopID string, maxSessions *int) (routes.GameShareResponse, error) {
	u.t.Helper()
	wsUUID, err := uuid.Parse(workshopID)
	if err != nil {
		return routes.GameShareResponse{}, fmt.Errorf("invalid workshopID: %w", err)
	}
	var result routes.GameShareResponse
	err = u.Post("games/"+gameID+"/shares", routes.CreateGameShareRequest{
		WorkshopID:  &wsUUID,
		MaxSessions: maxSessions,
	}, &result)
	return result, err
}
