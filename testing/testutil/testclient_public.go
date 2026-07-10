// package: testutil / public & guest API helpers
// type:    logic
// job:     unauthenticated PublicClient request helpers and guest share-token session flows
// limits:  public/guest HTTP helpers only; no authenticated UserClient methods (-> testclient_orgs / testclient_games)
package testutil

import (
	"fmt"

	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/obj"
)

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

// GuestGetGameInfo returns game info via a share token (composable high-level API)
func (p *PublicClient) GuestGetGameInfo(token string) (routes.GuestGameInfo, error) {
	p.t.Helper()
	var result routes.GuestGameInfo
	err := p.Get("play/"+token+"/info", &result)
	return result, err
}

// GuestCreateSession creates a guest session via a share token (composable high-level API)
func (p *PublicClient) GuestCreateSession(token string) (routes.GuestSessionResponse, error) {
	p.t.Helper()
	var result routes.GuestSessionResponse
	err := p.Post("play/"+token, nil, &result)
	return result, err
}

// GuestCreateSessionWithStream creates a guest session and triggers opening scene generation.
// TWO-PHASE INITIALIZATION: Creates session (phase 1), then sends "init" system action (phase 2).
// Returns the session response with the opening message fully populated.
func (p *PublicClient) GuestCreateSessionWithStream(token string) (routes.GuestSessionResponse, *StreamResult, error) {
	p.t.Helper()

	// PHASE 1: Create session (returns empty messages array)
	resp, err := p.GuestCreateSession(token)
	if err != nil {
		return routes.GuestSessionResponse{}, nil, err
	}

	if resp.GameSession == nil {
		return routes.GuestSessionResponse{}, nil, fmt.Errorf("no session returned")
	}

	// PHASE 2: Send "init" system action to trigger opening scene generation
	openingMsg, streamResult, err := p.GuestSendSystemMessage(token, resp.GameSession.ID.String(), "init")
	if err != nil {
		return routes.GuestSessionResponse{}, nil, fmt.Errorf("failed to trigger opening scene: %w", err)
	}

	// Add the opening message to the response
	resp.Messages = []routes.SessionMessageResponse{{GameSessionMessage: openingMsg}}

	return resp, streamResult, nil
}

// GuestGetSession loads a guest session via a share token (composable high-level API)
func (p *PublicClient) GuestGetSession(token, sessionID string) (routes.SessionResponse, error) {
	p.t.Helper()
	var result routes.SessionResponse
	err := p.Get("play/"+token+"/sessions/"+sessionID+"?messages=all", &result)
	return result, err
}

// GuestSendAction sends a player action to a guest session (composable high-level API)
func (p *PublicClient) GuestSendAction(token, sessionID, message string) (obj.GameSessionMessage, error) {
	p.t.Helper()
	var result obj.GameSessionMessage
	err := p.Post("play/"+token+"/sessions/"+sessionID, routes.SessionActionRequest{
		Message: message,
	}, &result)
	return result, err
}

// guestSendActionWithStream sends an action to a guest session and consumes the SSE stream
func (p *PublicClient) guestSendActionWithStream(token, sessionID string, req routes.SessionActionRequest) (obj.GameSessionMessage, *StreamResult, error) {
	p.t.Helper()

	// Send action
	var response obj.GameSessionMessage
	err := p.Post("play/"+token+"/sessions/"+sessionID, req, &response)
	if err != nil {
		return obj.GameSessionMessage{}, nil, err
	}

	// If not streaming, return the initial response
	if !response.Stream {
		return response, &StreamResult{Text: response.Message, ImageData: response.Image}, nil
	}

	// Consume the SSE stream (no auth required for message streams)
	result, err := consumeMessageStreamNoAuth(p.t, response.ID.String())
	if err != nil {
		return obj.GameSessionMessage{}, nil, fmt.Errorf("failed to consume stream: %w", err)
	}

	// Update response with full content
	response.Message = result.Text
	response.Image = result.ImageData
	response.Stream = false

	return response, result, nil
}

// GuestSendSystemMessage sends a system-type message to a guest session (e.g., "init" for opening scene)
func (p *PublicClient) GuestSendSystemMessage(token, sessionID, message string) (obj.GameSessionMessage, *StreamResult, error) {
	p.t.Helper()
	return p.guestSendActionWithStream(token, sessionID, routes.SessionActionRequest{
		Type:    "system",
		Message: message,
	})
}
