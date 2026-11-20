// package: testutil / HTTP test client
// type:    logic
// job:     drives the engine's mounted routes as the playback-engine does (bearer, play cookie or anonymous) and reads the platform's session endpoints
// limits:  test-only HTTP helpers; no suite lifecycle management (-> testutil suite)
package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cgl/config"
	"cgl/db"
	"cgl/obj"
	"engine"
	engineapi "engine/api"
)

// EngineResponse is what one of the engine's routes answered.
type EngineResponse struct {
	Status int
	Body   []byte
}

// Refusal reads the body as the refusal the playback-engine would show.
func (r EngineResponse) Refusal() engine.Refusal {
	var refusal engine.Refusal
	_ = json.Unmarshal(r.Body, &refusal)
	return refusal
}

// engineRequest calls the engine's subtree at /api/engine/<path>; prepare adds
// credentials.
func engineRequest(method, path string, body any, prepare func(*http.Request)) (EngineResponse, error) {
	serverURL, err := config.GetServerURL()
	if err != nil {
		return EngineResponse{}, fmt.Errorf("no server configured: %w", err)
	}
	var reader io.Reader
	if body != nil {
		blob, err := json.Marshal(body)
		if err != nil {
			return EngineResponse{}, err
		}
		reader = bytes.NewReader(blob)
	}
	req, err := http.NewRequest(method, serverURL+"/api/engine/"+strings.TrimPrefix(path, "/"), reader)
	if err != nil {
		return EngineResponse{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if prepare != nil {
		prepare(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return EngineResponse{}, err
	}
	defer resp.Body.Close()
	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		return EngineResponse{}, err
	}
	return EngineResponse{Status: resp.StatusCode, Body: blob}, nil
}

// Engine calls an engine route with the user's bearer token, as a headless
// client does.
func (u *UserClient) Engine(method, path string, body any) (EngineResponse, error) {
	u.t.Helper()
	return engineRequest(method, path, body, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+u.Token)
	})
}

// Engine calls an engine route with no credentials, as a guest's playback-engine does.
func (p *PublicClient) Engine(method, path string, body any) (EngineResponse, error) {
	p.t.Helper()
	return engineRequest(method, path, body, nil)
}

// EngineWithCookie calls an engine route carrying only a cookie, as the
// playback-engine in its iframe does.
func EngineWithCookie(cookie *http.Cookie, method, path string, body any) (EngineResponse, error) {
	return engineRequest(method, path, body, func(r *http.Request) { r.AddCookie(cookie) })
}

// PlayCookie asks for the play token cookie and answers it as the browser received it.
func (u *UserClient) PlayCookie() (*http.Cookie, error) {
	u.t.Helper()
	serverURL, err := config.GetServerURL()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/auth/play-token", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+u.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		blob, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("play token (%d): %s", resp.StatusCode, blob)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "cgl_play" {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no play cookie set")
}

// created reads the session ID a creation answered, or fails with the refusal.
func created(resp EngineResponse, err error) (string, EngineResponse, error) {
	if err != nil {
		return "", resp, err
	}
	if resp.Status != http.StatusCreated {
		return "", resp, fmt.Errorf("create answered %d: %s", resp.Status, resp.Body)
	}
	var out engineapi.CreateResponse
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return "", resp, err
	}
	return out.ID, resp, nil
}

// CreateSession creates a session as the user's playback-engine does.
func (u *UserClient) CreateSession(req engineapi.CreateRequest) (string, EngineResponse, error) {
	u.t.Helper()
	return created(u.Engine(http.MethodPost, "sessions", req))
}

// CreateSession creates a session as a guest's playback-engine does.
func (p *PublicClient) CreateSession(req engineapi.CreateRequest) (string, EngineResponse, error) {
	p.t.Helper()
	return created(p.Engine(http.MethodPost, "sessions", req))
}

// ListSessions is the user's sessions list.
func (u *UserClient) ListSessions() ([]db.UserSessionWithGame, error) {
	u.t.Helper()
	var sessions []db.UserSessionWithGame
	err := u.Get("sessions", &sessions)
	return sessions, err
}

// GetGameSessions is the user's own sessions of a game.
func (u *UserClient) GetGameSessions(gameID string) ([]obj.GameSession, error) {
	u.t.Helper()
	var sessions []obj.GameSession
	err := u.Get("games/"+gameID+"/sessions", &sessions)
	return sessions, err
}

// GetStats is the user's profile statistics.
func (u *UserClient) GetStats() (obj.UserStats, error) {
	u.t.Helper()
	var stats obj.UserStats
	err := u.Get("users/me/stats", &stats)
	return stats, err
}
