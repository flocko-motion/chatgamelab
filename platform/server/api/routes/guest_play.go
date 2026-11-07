// package: routes / share-token guest play handlers
// type:    logic
// job:     answers what a share token offers: the game and how many plays are left
// limits:  no route registration (-> router.go); share management (-> games_private_share.go)
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/game"

	"github.com/google/uuid"
)

// GuestGameInfo is the public game info returned for the welcome screen.
type GuestGameInfo struct {
	// GameID and Public let the welcome screen offer a copy. Only a public game
	// may be copied: a share link alone does not release its AI instructions.
	GameID      uuid.UUID `json:"gameId"`
	Public      bool      `json:"public"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Remaining   *int      `json:"remaining"` // null = unlimited, 0 = exhausted
}

// PlayGuestGetGameInfo godoc
//
//	@Summary		Get game info via share token
//	@Description	Returns basic game info (name, description) for the welcome screen.
//	@Tags			play
//	@Produce		json
//	@Param			token	path		string	true	"Private share token"
//	@Success		200		{object}	GuestGameInfo
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid token"
//	@Failure		404		{object}	httpx.ErrorResponse	"Game not found"
//	@Router			/play/{token}/info [get]
func PlayGuestGetGameInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Missing share token")
		return
	}

	gameShare, gameObj, httpErr := game.ValidatePrivateShareToken(r.Context(), token)
	if httpErr != nil {
		httpx.WriteHTTPError(w, httpErr)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, GuestGameInfo{
		GameID:      gameObj.ID,
		Public:      gameObj.Public,
		Name:        gameObj.Name,
		Description: gameObj.Description,
		Remaining:   gameShare.Remaining,
	})
}
