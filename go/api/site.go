// package: api / schemaf endpoints
// type:    logic
// job:     serves the site's branding to everyone and lets the admin change it
// limits:  the secure core decides and stores (core/branding.go)
package api

import (
	"context"

	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GetSiteEndpoint returns how the site presents itself.
//
// Public: the login page shows it before anyone signs in. The branding as it
// applies, and the defaults a field the admin never set shows.
type GetSiteEndpoint struct{}

func (GetSiteEndpoint) Method() string             { return "GET" }
func (GetSiteEndpoint) Path() string               { return "/api/site" }
func (GetSiteEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetSiteEndpoint) Handle(ctx context.Context, _ GetSiteReq) (SiteResp, error) {
	return siteView(core.ReadBranding(ctx))
}

// GetSiteReq takes no parameters.
type GetSiteReq struct{}

// SiteResp is how the site presents itself.
type SiteResp struct {
	Branding core.Branding `json:"branding"`
	// Defaults are what each branding field shows while the admin has not set it.
	Defaults core.Branding `json:"defaults"`
}

// UpdateSiteBrandingEndpoint changes the site's branding; admins only.
//
// Fields by name: a value sets the field (an empty one hides a link or
// address; the site name cannot be empty), null gives it back its default.
// Fields not sent stay. Answers how the site now presents itself.
type UpdateSiteBrandingEndpoint struct{}

func (UpdateSiteBrandingEndpoint) Method() string             { return "PATCH" }
func (UpdateSiteBrandingEndpoint) Path() string               { return "/api/system/settings/branding" }
func (UpdateSiteBrandingEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateSiteBrandingEndpoint) Handle(ctx context.Context, req UpdateSiteBrandingReq) (SiteResp, error) {
	change := core.BrandingChange{Set: map[string]string{}}
	for field, value := range req.Branding {
		if value == nil {
			change.Reset = append(change.Reset, field)
		} else {
			change.Set[field] = *value
		}
	}
	return siteView(core.EditBranding(ctx, change))
}

// UpdateSiteBrandingReq is the branding fields to change, by name; null
// resets a field to its default.
type UpdateSiteBrandingReq struct {
	Branding map[string]*string `json:"branding"`
}

func siteView(b core.Branding, err error) (SiteResp, error) {
	if err != nil {
		return SiteResp{}, err
	}
	return SiteResp{Branding: b, Defaults: core.DefaultBranding()}, nil
}
