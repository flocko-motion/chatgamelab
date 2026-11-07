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
