// package: core / verified pages
// type:    logic
// job:     an organisation's pages that may frame its embedded games: list, register, change, remove
// limits:  asking a page's host (-> embed.go)
package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/url"
	"strings"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// A verified page is an organisation's page on its own website that may
// frame its embedded games (#340). Its head and staff register and manage
// it; no one approves it. Before a game embedded for the page starts, the
// platform asks the page's callback, and the host says yes or no
// (embed.go).

// pageIDBytes is the length of a page ID: 16 hex characters.
const pageIDBytes = 8

// VerifiedPage is one verified page as its organisation's head and staff
// see it.
type VerifiedPage struct {
	ID             uuid.UUID
	OrganisationID uuid.UUID
	PageID         string // the ID the host recognises in the callback
	Origin         string // who may frame the playback-engine: scheme, host, port
	PageURL        string
	CallbackURL    string
	CreatedAt      time.Time
	ModifiedAt     time.Time

	mayEdit, mayDelete bool
}

// MayEdit says whether the caller may change it: {Edit, VerifiedPage}.
func (p *VerifiedPage) MayEdit() bool { return p.mayEdit }

// MayDelete says whether the caller may remove it: {Delete, VerifiedPage}.
func (p *VerifiedPage) MayDelete() bool { return p.mayDelete }

// ListVerifiedPages lists organisation org's verified pages, for its head
// and staff: {List, VerifiedPage, OfOwnOrganisation}. Anyone else gets
// not found.
func ListVerifiedPages(ctx context.Context, org uuid.UUID) ([]*VerifiedPage, error) {
	rights := verifiedPageGates(ctx, org)
	if !rights.list {
		return nil, ErrNotFound
	}
	return loadVerifiedPages(ctx, org, nil, rights)
}

// readVerifiedPage opens one verified page for the caller who may list its
// organisation's.
func readVerifiedPage(ctx context.Context, id uuid.UUID) (*VerifiedPage, error) {
	org, err := store.Queries().V2VerifiedPageOrganisation(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	rights := verifiedPageGates(ctx, org)
	if !rights.list {
		return nil, ErrNotFound
	}
	pages, err := loadVerifiedPages(ctx, org, &id, rights)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, ErrNotFound
	}
	return pages[0], nil
}

func loadVerifiedPages(ctx context.Context, org uuid.UUID, id *uuid.UUID, rights verifiedPageRights) ([]*VerifiedPage, error) {
	rows, err := store.Queries().V2VerifiedPages(ctx, db.V2VerifiedPagesParams{InstitutionID: org, ID: nullUUID(id)})
	if err != nil {
		return nil, err
	}
	pages := make([]*VerifiedPage, len(rows))
	for i, r := range rows {
		pages[i] = &VerifiedPage{
			ID: r.ID, OrganisationID: r.InstitutionID, PageID: r.PageID, Origin: r.Origin,
			PageURL: r.PageUrl, CallbackURL: r.CallbackUrl, CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
			mayEdit: rights.edit, mayDelete: rights.delete,
		}
	}
	return pages, nil
}

// VerifiedPageDraft is a verified page's address and callback; the origin
// comes from the address.
type VerifiedPageDraft struct {
	PageURL     string
	CallbackURL string
}

// CreateVerifiedPage registers a verified page for organisation org, with a
// new page ID: {Create, VerifiedPage, OfOwnOrganisation}.
func CreateVerifiedPage(ctx context.Context, org uuid.UUID, d VerifiedPageDraft) (*VerifiedPage, error) {
	if !verifiedPageGates(ctx, org).create {
		return nil, forbidden("you may not register pages for this organisation")
	}
	origin, err := checkPageURLs(d)
	if err != nil {
		return nil, err
	}
	pageID, err := newPageID()
	if err != nil {
		return nil, err
	}
	id, err := store.Queries().V2CreateVerifiedPage(ctx, db.V2CreateVerifiedPageParams{
		InstitutionID: org, PageID: pageID, Origin: origin,
		PageUrl: d.PageURL, CallbackUrl: d.CallbackURL, CreatedBy: account(ctx).UUID,
	})
	if err != nil {
		return nil, err
	}
	log.Info("verified page registered", "verified_page_id", id, "institution_id", org, "by", account(ctx).UUID)
	return readVerifiedPage(ctx, id)
}

// UpdateVerifiedPage changes a verified page's address and callback; its
// page ID stays: {Edit, VerifiedPage, OfOwnOrganisation}.
func UpdateVerifiedPage(ctx context.Context, id uuid.UUID, d VerifiedPageDraft) (*VerifiedPage, error) {
	page, err := readVerifiedPage(ctx, id)
	if err != nil {
		return nil, err
	}
	if !page.MayEdit() {
		return nil, forbidden("you may not change this page")
	}
	origin, err := checkPageURLs(d)
	if err != nil {
		return nil, err
	}
	if err := store.Queries().V2UpdateVerifiedPage(ctx, db.V2UpdateVerifiedPageParams{
		ID: id, Origin: origin, PageUrl: d.PageURL, CallbackUrl: d.CallbackURL,
	}); err != nil {
		return nil, err
	}
	return readVerifiedPage(ctx, id)
}

// DeleteVerifiedPage removes a verified page: {Delete, VerifiedPage,
// OfOwnOrganisation}. The games embedded for it start nothing until they
// get another page.
func DeleteVerifiedPage(ctx context.Context, id uuid.UUID) error {
	page, err := readVerifiedPage(ctx, id)
	if err != nil {
		return err
	}
	if !page.MayDelete() {
		return forbidden("you may not remove this page")
	}
	if err := store.Queries().V2DeleteVerifiedPage(ctx, id); err != nil {
		return err
	}
	log.Info("verified page removed", "verified_page_id", id, "by", account(ctx).UUID)
	return nil
}

// checkPageURLs checks a page's address and callback and answers the
// page's origin. Both are https: TLS proves who answers the callback. In
// development, a page on the own machine may use plain http.
func checkPageURLs(d VerifiedPageDraft) (string, error) {
	page, err := pageURL(d.PageURL, "pageUrl")
	if err != nil {
		return "", err
	}
	if _, err := pageURL(d.CallbackURL, "callbackUrl"); err != nil {
		return "", err
	}
	return page.Scheme + "://" + page.Host, nil
}

// maxPageURL caps the length of a page's address and callback.
const maxPageURL = 2000

func pageURL(raw, field string) (*url.URL, error) {
	if raw == "" || len(raw) > maxPageURL {
		return nil, refuse("%s must be an address of at most %d characters", field, maxPageURL)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, refuse("%s must be a full address, such as https://example.org/play", field)
	}
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && schemafapi.DevMode() && isLoopback(u.Hostname()):
	default:
		return nil, refuse("%s must start with https://", field)
	}
	return u, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newPageID() (string, error) {
	b := make([]byte, pageIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
