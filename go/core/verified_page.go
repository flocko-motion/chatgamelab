// package: core / verified pages
// type:    logic
// job:     an organisation's page that may frame its embedded games: register, change, remove, approve
// limits:  asking a page's host (-> embed.go)
package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// A verified page is an organisation's one page on its own website that may
// frame its embedded games (#340). Its head and staff register and manage
// it; an admin approves its site (scheme and host), and only a change of
// site withdraws the approval: port and path are free, for the page and its
// callback alike. The host's yes to each start is embed.go's.

// pageIDBytes is the length of a page ID: 16 hex characters.
const pageIDBytes = 8

// VerifiedPage is one verified page.
type VerifiedPage struct {
	ID               uuid.UUID
	OrganisationID   uuid.UUID
	OrganisationName string
	PageID           string     // the ID the host recognises in the callback
	Site             string     // what the admin approves, scheme://host: any port and path on it may frame
	ApprovedAt       *time.Time // nil: not approved, its games start nothing
	PageURL          string
	CallbackURL      string
	CreatedAt        time.Time
	ModifiedAt       time.Time

	mayEdit, mayDelete, mayApprove bool
}

// MayEdit says whether the caller may change it: {Edit, VerifiedPage}.
func (p *VerifiedPage) MayEdit() bool { return p.mayEdit }

// MayDelete says whether the caller may remove it: {Delete, VerifiedPage}.
func (p *VerifiedPage) MayDelete() bool { return p.mayDelete }

// MayApprove says whether the caller may approve it or withdraw the
// approval: {Manage, VerifiedPage}.
func (p *VerifiedPage) MayApprove() bool { return p.mayApprove }

// ListVerifiedPages lists organisation org's verified page, at most one,
// for its head and staff, or anyone who lists every page. Anyone else gets
// not found.
func ListVerifiedPages(ctx context.Context, org uuid.UUID) ([]*VerifiedPage, error) {
	rights := verifiedPageGates(ctx, org)
	if !rights.list {
		return nil, ErrNotFound
	}
	return loadVerifiedPages(ctx, &org, nil, rights)
}

// ListAllVerifiedPages lists the verified pages the caller may see, the
// unapproved first: every one for an admin, the admins' allow list; their
// organisation's for its head and staff.
func ListAllVerifiedPages(ctx context.Context) ([]*VerifiedPage, error) {
	all, org := verifiedPageListScope(ctx)
	switch {
	case all:
		return loadVerifiedPages(ctx, nil, nil, verifiedPageRights{list: true})
	case org != nil:
		return ListVerifiedPages(ctx, *org)
	}
	return []*VerifiedPage{}, nil
}

// readVerifiedPage opens one verified page for a caller who may list it.
func readVerifiedPage(ctx context.Context, id uuid.UUID) (*VerifiedPage, error) {
	org, err := store.Queries().V2VerifiedPageOrganisation(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	rights := verifiedPageGates(ctx, org)
	if !rights.list {
		return nil, ErrNotFound
	}
	pages, err := loadVerifiedPages(ctx, &org, &id, rights)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, ErrNotFound
	}
	return pages[0], nil
}

func loadVerifiedPages(ctx context.Context, org, id *uuid.UUID, rights verifiedPageRights) ([]*VerifiedPage, error) {
	rows, err := store.Queries().V2VerifiedPages(ctx, db.V2VerifiedPagesParams{InstitutionID: nullUUID(org), ID: nullUUID(id)})
	if err != nil {
		return nil, err
	}
	mayApprove := mayApproveVerifiedPages(ctx)
	pages := make([]*VerifiedPage, len(rows))
	for i, r := range rows {
		p := &VerifiedPage{
			ID: r.ID, OrganisationID: r.InstitutionID, OrganisationName: r.InstitutionName, PageID: r.PageID,
			Site: r.Site, PageURL: r.PageUrl, CallbackURL: r.CallbackUrl, CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
			mayEdit: rights.edit, mayDelete: rights.delete, mayApprove: mayApprove,
		}
		if r.ApprovedAt.Valid {
			p.ApprovedAt = &r.ApprovedAt.Time
		}
		pages[i] = p
	}
	return pages, nil
}

// VerifiedPageDraft is a verified page's address and callback; the site
// comes from the address.
type VerifiedPageDraft struct {
	PageURL     string
	CallbackURL string
}

// CreateVerifiedPage registers organisation org's verified page, unapproved,
// with a new page ID: {Create, VerifiedPage, OfOwnOrganisation}. An
// organisation has one; a second is refused (verified_page_exists).
func CreateVerifiedPage(ctx context.Context, org uuid.UUID, d VerifiedPageDraft) (*VerifiedPage, error) {
	if !verifiedPageGates(ctx, org).create {
		return nil, forbidden("you may not register pages for this organisation")
	}
	site, err := checkPageURLs(d)
	if err != nil {
		return nil, err
	}
	pageID, err := newPageID()
	if err != nil {
		return nil, err
	}
	id, err := store.Queries().V2CreateVerifiedPage(ctx, db.V2CreateVerifiedPageParams{
		InstitutionID: org, PageID: pageID, Site: site,
		PageUrl: d.PageURL, CallbackUrl: d.CallbackURL, CreatedBy: account(ctx).UUID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, conflict("verified_page_exists", "the organisation has a verified page already")
	}
	if err != nil {
		return nil, err
	}
	log.Info("verified page registered", "verified_page_id", id, "institution_id", org, "by", account(ctx).UUID)
	return readVerifiedPage(ctx, id)
}

// UpdateVerifiedPage changes a verified page's address and callback; its
// page ID stays, and a change of its site withdraws its approval (a new
// port or path keeps it): {Edit, VerifiedPage, OfOwnOrganisation}.
func UpdateVerifiedPage(ctx context.Context, id uuid.UUID, d VerifiedPageDraft) (*VerifiedPage, error) {
	page, err := readVerifiedPage(ctx, id)
	if err != nil {
		return nil, err
	}
	if !page.MayEdit() {
		return nil, forbidden("you may not change this page")
	}
	site, err := checkPageURLs(d)
	if err != nil {
		return nil, err
	}
	if err := store.Queries().V2UpdateVerifiedPage(ctx, db.V2UpdateVerifiedPageParams{
		ID: id, Site: site, PageUrl: d.PageURL, CallbackUrl: d.CallbackURL,
	}); err != nil {
		return nil, err
	}
	return readVerifiedPage(ctx, id)
}

// ApproveVerifiedPage approves a verified page, or withdraws the approval:
// {Manage, VerifiedPage, Everywhere}.
func ApproveVerifiedPage(ctx context.Context, id uuid.UUID, approve bool) (*VerifiedPage, error) {
	page, err := readVerifiedPage(ctx, id)
	if err != nil {
		return nil, err
	}
	if !page.MayApprove() {
		return nil, forbidden("you may not approve verified pages")
	}
	var by uuid.NullUUID
	if approve {
		by = account(ctx)
	}
	if err := store.Queries().V2ApproveVerifiedPage(ctx, db.V2ApproveVerifiedPageParams{ID: id, ApprovedBy: by}); err != nil {
		return nil, err
	}
	log.Info("verified page approval set", "verified_page_id", id, "approved", approve, "by", account(ctx).UUID)
	return readVerifiedPage(ctx, id)
}

// DeleteVerifiedPage removes a verified page: {Delete, VerifiedPage,
// OfOwnOrganisation}. The organisation's embedded games start nothing until
// it has another, approved.
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
// page's site. Both are https: TLS proves who answers the callback. In
// development, plain http as well, on any host. Either way the admin
// approves the site, scheme and host. The callback is on the page's site,
// which the admin approves for both. Each refusal has a code of its own,
// naming the field.
func checkPageURLs(d VerifiedPageDraft) (string, error) {
	page, err := pageURL(d.PageURL, pageField)
	if err != nil {
		return "", err
	}
	callback, err := pageURL(d.CallbackURL, callbackField)
	if err != nil {
		return "", err
	}
	site := siteOf(page)
	if siteOf(callback) != site {
		return "", badRequest(obj.ErrCodeCallbackOffSite, "callbackUrl must be on the page's site, %s", site)
	}
	return site, nil
}

// urlField is one of a page's addresses, with the codes of its refusals.
type urlField struct {
	name     string // as the request names it
	invalid  string // no full web address
	notHTTPS string // plain http outside development
}

var (
	pageField     = urlField{"pageUrl", obj.ErrCodePageURLInvalid, obj.ErrCodePageURLNotHTTPS}
	callbackField = urlField{"callbackUrl", obj.ErrCodeCallbackURLInvalid, obj.ErrCodeCallbackURLNotHTTPS}
)

// siteOf is an address's site: its scheme and host, without port or path,
// the host in lower case (an IPv6 address in brackets).
func siteOf(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return u.Scheme + "://" + host
}

// maxPageURL caps the length of a page's address and callback.
const maxPageURL = 2000

// pageURL reads one of a page's addresses: a full web address, https, or in
// development plain http as well.
func pageURL(raw string, f urlField) (*url.URL, error) {
	if raw == "" || len(raw) > maxPageURL {
		return nil, badRequest(f.invalid, "%s must be an address of at most %d characters", f.name, maxPageURL)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, badRequest(f.invalid, "%s must be a full address, such as https://example.org/play", f.name)
	}
	if u.Scheme == "http" && !schemafapi.DevMode() {
		return nil, badRequest(f.notHTTPS, "%s must start with https://", f.name)
	}
	return u, nil
}

func newPageID() (string, error) {
	b := make([]byte, pageIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
