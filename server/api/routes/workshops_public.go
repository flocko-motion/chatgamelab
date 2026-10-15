// package: routes / public workshop page handlers
// type:    logic
// job:     serves the public workshop page /w/<slug> and its game downloads to visitors without an account
// limits:  no route registration (-> router.go); play goes through the share-token routes (-> guest_play.go)
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
)

// GetPublicWorkshopPage godoc
//
//	@Summary		Get public workshop page
//	@Description	Returns a public workshop's name, description and public games. No authentication.
//	@Description	Unknown slugs and switched-off pages both return 404.
//	@Tags			public
//	@Produce		json
//	@Param			slug	path		string	true	"Public page slug"
//	@Success		200		{object}	obj.PublicWorkshopPage
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Router			/public/workshops/{slug} [get]
func GetPublicWorkshopPage(w http.ResponseWriter, r *http.Request) {
	page, err := db.GetPublicWorkshopPage(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeDBError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// GetPublicWorkshopGameYAML godoc
//
//	@Summary		Download a game from a public workshop page
//	@Description	Exports a game listed on a public workshop page as YAML. No authentication.
//	@Tags			public
//	@Produce		application/x-yaml
//	@Param			slug	path		string	true	"Public page slug"
//	@Param			id		path		string	true	"Game ID (UUID)"
//	@Success		200		{object}	obj.Game
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Router			/public/workshops/{slug}/games/{id}/yaml [get]
func GetPublicWorkshopGameYAML(w http.ResponseWriter, r *http.Request) {
	gameID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrNotFound("game not found"))
		return
	}
	game, err := db.GetPublicWorkshopGame(r.Context(), r.PathValue("slug"), gameID)
	if err != nil {
		writeDBError(w, err)
		return
	}
	httpx.WriteYAML(w, http.StatusOK, game)
}

func writeDBError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*obj.AppError); ok {
		httpx.WriteAppError(w, appErr)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, err.Error())
}
