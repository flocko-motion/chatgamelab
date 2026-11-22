// package: api / schemaf endpoints
// type:    logic
// job:     has a game's cover drawn and serves its picture
// limits:  who may see and change a cover is the core's (core/game_cover.go); drawing it is session/cover.go's
package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"cgl/api/wire"
	"cgl/core"
	"cgl/session"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// maxCoverHint caps the designer's hint, in characters.
const maxCoverHint = 300

// MakeGameCoverEndpoint draws a new cover for a game, replacing the one it
// had, and answers with the game as it now is (coverAt set). It takes
// mayEditCard (403 otherwise) and blocks until the picture is made.
//
// The caller pays as for a session of the game: the first key of theirs
// that draws pictures, workshop, sponsor, organisation, own, site. None:
// 400 no_api_key; a key its provider rejects: 400 invalid_api_key; a
// provider that fails: 503 ai_error. The hint, at most 300 characters, says
// what the cover should show; empty leaves it to the game.
type MakeGameCoverEndpoint struct{}

func (MakeGameCoverEndpoint) Method() string             { return "POST" }
func (MakeGameCoverEndpoint) Path() string               { return "/api/games/{id}/cover" }
func (MakeGameCoverEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (MakeGameCoverEndpoint) Handle(ctx context.Context, req MakeGameCoverReq) (GameViewResp, error) {
	gameID, err := parseID(req.GameID, "Invalid game ID")
	if err != nil {
		return GameViewResp{}, err
	}
	hint := strings.TrimSpace(req.Hint)
	if utf8.RuneCountInString(hint) > maxCoverHint {
		return GameViewResp{}, errBadRequest("the hint is longer than " + strconv.Itoa(maxCoverHint) + " characters")
	}
	return wire.GameViewOf(ctx)(session.MakeGameCover(ctx, gameID, hint))
}

// MakeGameCoverReq is the game from the path and the designer's hint.
type MakeGameCoverReq struct {
	GameID string `path:"id"`
	Hint   string `json:"hint"`
}

// GameCoverEndpoint serves a game's cover picture to whoever may see the
// game: 404 for a game they may not see or one without a cover. With ?v= (the game's coverAt) the answer may be
// cached for good, as a new cover comes with a new v.
type GameCoverEndpoint struct{}

func (GameCoverEndpoint) Method() string             { return "GET" }
func (GameCoverEndpoint) Path() string               { return "/api/games/{id}/cover" }
func (GameCoverEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GameCoverEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	gameID, err := parseID(r.PathValue("id"), "Invalid game ID")
	if err != nil {
		return err
	}
	cover, err := core.ReadGameCover(r.Context(), gameID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", http.DetectContentType(cover.Image))
	w.Header().Set("Content-Length", strconv.Itoa(len(cover.Image)))
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, no-cache")
	}
	_, err = w.Write(cover.Image)
	return err
}
