// package: api / schemaf endpoints
// type:    logic
// job:     serves the public workshop page /w/<slug> and its game downloads to visitors without an account
// limits:  play goes through the share-token endpoints; the page is the core's (core/workshop_public.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// PublicWorkshopPageResp is a public workshop page.
type PublicWorkshopPageResp = obj.PublicWorkshopPage

// PublicWorkshopSlugReq names a public workshop page by its slug.
type PublicWorkshopSlugReq struct {
	Slug string `path:"slug"`
}

// GetPublicWorkshopPageEndpoint returns a public workshop's name, description
// and public games.
//
// An unknown slug and a switched-off page both answer 404.
type GetPublicWorkshopPageEndpoint struct{}

func (GetPublicWorkshopPageEndpoint) Method() string             { return "GET" }
func (GetPublicWorkshopPageEndpoint) Path() string               { return "/api/public/workshops/{slug}" }
func (GetPublicWorkshopPageEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetPublicWorkshopPageEndpoint) Handle(ctx context.Context, req PublicWorkshopSlugReq) (PublicWorkshopPageResp, error) {
	page, err := core.PublicWorkshopPage(ctx, req.Slug)
	if err != nil {
		return obj.PublicWorkshopPage{}, err
	}
	return *page, nil
}

// PublicWorkshopGameReq names a game on a public workshop page.
type PublicWorkshopGameReq struct {
	Slug string `path:"slug"`
	ID   string `path:"id"`
}

// GetPublicWorkshopGameYAMLEndpoint exports a game listed on a public workshop
// page as a game file, YAML by default.
type GetPublicWorkshopGameYAMLEndpoint struct{}

func (GetPublicWorkshopGameYAMLEndpoint) Method() string { return "GET" }
func (GetPublicWorkshopGameYAMLEndpoint) Path() string {
	return "/api/public/workshops/{slug}/games/{id}/yaml"
}
func (GetPublicWorkshopGameYAMLEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetPublicWorkshopGameYAMLEndpoint) Produces() []string {
	return []string{"application/x-yaml", "application/json"}
}
func (GetPublicWorkshopGameYAMLEndpoint) Handle(ctx context.Context, req PublicWorkshopGameReq) (GameFileResp, error) {
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return obj.GameFile{}, err
	}
	return wire.GameFileOf(core.PublicWorkshopGame(ctx, req.Slug, gameID))
}
