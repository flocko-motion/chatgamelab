// package: api / schemaf endpoints
// type:    logic
// job:     serves the public workshop page /w/<slug> and its game downloads to visitors without an account
// limits:  play goes through the share-token endpoints; the page's data and visibility live in db
package api

import (
	"context"

	"cgl/db"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// PublicWorkshopPageResp is a public workshop page.
type PublicWorkshopPageResp = obj.PublicWorkshopPage

// PublicWorkshopSlugReq names a public workshop page by its slug.
type PublicWorkshopSlugReq struct {
	Slug string `path:"slug" json:"-"`
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
	page, err := db.GetPublicWorkshopPage(ctx, req.Slug)
	if err != nil {
		return obj.PublicWorkshopPage{}, appError(err)
	}
	return *page, nil
}

// PublicWorkshopGameReq names a game on a public workshop page.
type PublicWorkshopGameReq struct {
	Slug string `path:"slug" json:"-"`
	ID   string `path:"id" json:"-"`
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
	gameID, err := uuid.Parse(req.ID)
	if err != nil {
		return obj.GameFile{}, appError(obj.ErrNotFound("game not found"))
	}
	game, err := db.GetPublicWorkshopGame(ctx, req.Slug, gameID)
	if err != nil {
		return obj.GameFile{}, appError(err)
	}
	return obj.NewGameFile(*game), nil
}
