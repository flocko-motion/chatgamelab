// package: testutil / the words for embedding a game
// type:    logic
// job:     a game's embed link, an organisation's verified pages and the playback-engine page's
// framing, as their callers use them
// limits:  embedding words only; playing goes through sessions.go
package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"cgl/apiclient"

	"github.com/google/uuid"
)

// GameEmbed is a game's embed link; VerifiedPage an organisation's page.
type (
	GameEmbed    = apiclient.GameEmbedView
	VerifiedPage = apiclient.VerifiedPageView
)

// Embedding is what a designer sets for a game's embed link.
type Embedding struct {
	Enabled      bool
	Mode         string // the mode asked for; "" is the verified mode, the only one
	PaidBy       string // the key share it draws from; "" keeps the one paying now
	VerifiedPage string // the verified page it runs on
}

// SetGameEmbed sets a game's embed link, making it the first time.
func (c caller) SetGameEmbed(gameID string, e Embedding) (*GameEmbed, error) {
	req := apiclient.SetGameEmbedReq{Enabled: e.Enabled, Mode: "verified"}
	if e.Mode != "" {
		req.Mode = e.Mode
	}
	if e.PaidBy != "" {
		id, err := uuid.Parse(e.PaidBy)
		if err != nil {
			return nil, err
		}
		req.SponsorKeyShareId = &id
	}
	if e.VerifiedPage != "" {
		id, err := uuid.Parse(e.VerifiedPage)
		if err != nil {
			return nil, err
		}
		req.VerifiedPageId = &id
	}
	resp, err := decode[apiclient.GameEmbedResp](c.api.PutApiGamesIdEmbedWithResponse(ctx(), gameID, req))
	return resp.Embed, err
}

// GetGameEmbed is a game's embed link; nil while it has none.
func (c caller) GetGameEmbed(gameID string) (*GameEmbed, error) {
	resp, err := decode[apiclient.GameEmbedResp](c.api.GetApiGamesIdEmbedWithResponse(ctx(), gameID))
	return resp.Embed, err
}

// ListVerifiedPages is an organisation's verified pages.
func (c caller) ListVerifiedPages(orgID string) ([]VerifiedPage, error) {
	resp, err := decode[apiclient.VerifiedPagesResp](c.api.GetApiInstitutionsIdVerifiedPagesWithResponse(ctx(), orgID))
	return resp.Pages, err
}

// ListAllVerifiedPages is every verified page the caller may see: the
// admins' allow list.
func (c caller) ListAllVerifiedPages() ([]VerifiedPage, error) {
	return decodeItems[VerifiedPage](c.api.GetApiVerifiedPagesWithResponse(ctx(), &apiclient.GetApiVerifiedPagesParams{}))
}

// ApproveVerifiedPage approves a verified page, or withdraws the approval.
func (c caller) ApproveVerifiedPage(pageID string, approved bool) (VerifiedPage, error) {
	return decode[VerifiedPage](c.api.PutApiVerifiedPagesIdApprovalWithResponse(ctx(), pageID,
		apiclient.ApproveVerifiedPageReq{Approved: approved}))
}

// UpdateVerifiedPage changes a verified page's address and callback.
func (c caller) UpdateVerifiedPage(pageID, pageURL, callbackURL string) (VerifiedPage, error) {
	return decode[VerifiedPage](c.api.PatchApiVerifiedPagesIdWithResponse(ctx(), pageID,
		apiclient.UpdateVerifiedPageReq{PageUrl: pageURL, CallbackUrl: callbackURL}))
}

// DeleteVerifiedPage removes a verified page.
func (c caller) DeleteVerifiedPage(pageID string) error {
	return done(c.api.DeleteApiVerifiedPagesIdWithResponse(ctx(), pageID))
}

// CreateVerifiedPage registers a verified page for an organisation.
func (c caller) CreateVerifiedPage(orgID, pageURL, callbackURL string) (VerifiedPage, error) {
	return decode[VerifiedPage](c.api.PostApiInstitutionsIdVerifiedPagesWithResponse(ctx(), orgID,
		apiclient.CreateVerifiedPageReq{PageUrl: pageURL, CallbackUrl: callbackURL}))
}

// PlaybackPolicy is the Content-Security-Policy the playback-engine's page
// is served with, opened with query (such as "new&share=…").
func (c caller) PlaybackPolicy(query string) (string, error) {
	resp, err := c.api.GetApiEnginePlaybackEngineWithResponse(ctx(), func(_ context.Context, req *http.Request) error {
		req.URL.RawQuery = query
		return nil
	})
	if err != nil {
		return "", err
	}
	return resp.HTTPResponse.Header.Get("Content-Security-Policy"), nil
}

// Host is a host page's callback, as a verified page registers it: it
// answers every start as told (yes, until told otherwise) and remembers what
// it was asked.
type Host struct {
	URL    string
	mu     sync.Mutex
	answer func(w http.ResponseWriter)
	asked  []map[string]string
}

// StartHost runs a host page's callback for the test.
func (s *BaseSuite) StartHost() *Host {
	h := &Host{answer: hostYes}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		h.mu.Lock()
		h.asked = append(h.asked, body)
		answer := h.answer
		h.mu.Unlock()
		answer(w)
	}))
	s.T().Cleanup(server.Close)
	h.URL = server.URL
	return h
}

// SaysYes answers {"ok": true}.
func (h *Host) SaysYes() { h.says(hostYes) }

// SaysNo answers {"ok": false}.
func (h *Host) SaysNo() {
	h.says(func(w http.ResponseWriter) { _, _ = w.Write([]byte(`{"ok": false}`)) })
}

// Breaks answers a server error, with a yes in its body.
func (h *Host) Breaks() {
	h.says(func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"ok": true}`))
	})
}

// FallsSilent answers a yes only after the platform stopped waiting.
func (h *Host) FallsSilent() {
	h.says(func(w http.ResponseWriter) { time.Sleep(6 * time.Second); hostYes(w) })
}

// LastAsked is what the host was asked last.
func (h *Host) LastAsked() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.asked) == 0 {
		return nil
	}
	return h.asked[len(h.asked)-1]
}

func (h *Host) says(answer func(w http.ResponseWriter)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.answer = answer
}

func hostYes(w http.ResponseWriter) { _, _ = w.Write([]byte(`{"ok": true}`)) }
