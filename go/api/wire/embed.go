// package: api/wire / embedding views
// type:    logic
// job:     a game's embed link and an organisation's verified pages on the wire
// limits:  converts the core's types; decides nothing (-> core/embed.go, core/verified_page.go)
package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// GameEmbedView is a game's embed link as its designer sees it: the token
// its snippet carries, whether it is on, who may frame it and what pays.
type GameEmbedView struct {
	ID    uuid.UUID `json:"id"`
	Token string    `json:"token"`
	// Enabled is false while embedding is switched off; the token stays.
	Enabled bool `json:"enabled"`
	// Mode is "verified" (one verified page, whose host says yes to every
	// start), the only mode.
	Mode            string     `json:"mode"`
	VerifiedPageID  *uuid.UUID `json:"verifiedPageId,omitempty"`
	VerifiedPageURL *string    `json:"verifiedPageUrl,omitempty"`
	// SponsorKeyShareID is the share it draws from; KeyName and Platform
	// name the key that pays.
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId,omitempty"`
	KeyName           string     `json:"keyName"`
	Platform          string     `json:"platform"`
}

// GameEmbedViewOf converts a game's embed link, passing a loading error
// through; nil: the game has none.
func GameEmbedViewOf(e *core.GameEmbed, err error) (*GameEmbedView, error) {
	if err != nil || e == nil {
		return nil, err
	}
	return &GameEmbedView{
		ID: e.ID, Token: e.Token, Enabled: e.Enabled, Mode: string(e.Mode),
		VerifiedPageID: e.VerifiedPageID, VerifiedPageURL: e.VerifiedPageURL,
		SponsorKeyShareID: e.SourceShareID, KeyName: e.KeyName, Platform: e.Platform,
	}, nil
}

// VerifiedPageView is an organisation's verified page.
type VerifiedPageView struct {
	ID             uuid.UUID `json:"id"`
	OrganisationID uuid.UUID `json:"organisationId"`
	// PageID is what the host recognises in the callback.
	PageID      string    `json:"pageId"`
	Origin      string    `json:"origin"`
	PageURL     string    `json:"pageUrl"`
	CallbackURL string    `json:"callbackUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	MayEdit     bool      `json:"mayEdit"`
	MayDelete   bool      `json:"mayDelete"`
}

// VerifiedPageViewOf converts one verified page, passing a loading error
// through.
func VerifiedPageViewOf(p *core.VerifiedPage, err error) (VerifiedPageView, error) {
	if err != nil {
		return VerifiedPageView{}, err
	}
	return verifiedPageView(p), nil
}

// VerifiedPageViewsOf converts verified pages, passing a loading error
// through.
func VerifiedPageViewsOf(pages []*core.VerifiedPage, err error) ([]VerifiedPageView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]VerifiedPageView, len(pages))
	for i, p := range pages {
		views[i] = verifiedPageView(p)
	}
	return views, nil
}

func verifiedPageView(p *core.VerifiedPage) VerifiedPageView {
	return VerifiedPageView{
		ID: p.ID, OrganisationID: p.OrganisationID, PageID: p.PageID, Origin: p.Origin,
		PageURL: p.PageURL, CallbackURL: p.CallbackURL, CreatedAt: p.CreatedAt, ModifiedAt: p.ModifiedAt,
		MayEdit: p.MayEdit(), MayDelete: p.MayDelete(),
	}
}
