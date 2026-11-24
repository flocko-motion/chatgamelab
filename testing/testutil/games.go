// package: testutil / the words for games and their shares
// type:    logic
// job:     creating games (empty or from a game file in testdata/games), reading, listing, changing, cloning and deleting them, their sponsors, favorites and share links
// limits:  game words only; playing a game lives in sessions.go
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cgl/apiclient"

	"github.com/google/uuid"
)

// The content types of a game file.
const (
	contentYAML = "application/x-yaml"
)

// CreateGame creates an empty game.
func (c caller) CreateGame(name string) (Game, error) {
	return decode[Game](c.api.PostApiGamesNewWithResponse(ctx(), GameCreate{Name: &name}))
}

// NewGame is the template for a game the caller creates.
func (c caller) NewGame() (GameTemplate, error) {
	return decode[GameTemplate](c.api.GetApiGamesNewWithResponse(ctx()))
}

// CreateGameWith creates a game from a filled-in template.
func (c caller) CreateGameWith(fields GameCreate) (GameView, error) {
	return decode[GameView](c.api.PostApiGamesNewWithResponse(ctx(), fields))
}

// UploadGame creates a game from testdata/games/<name>.yaml, under a unique
// name so games from different tests never clash.
// Example: game := Must(alice.UploadGame("simple-quest"))
func (c caller) UploadGame(name string) (Game, error) {
	file, err := readGameFile(name)
	if err != nil {
		return Game{}, err
	}
	unique := "Test Game " + uuid.NewString()[:8]
	game, err := c.CreateGame(unique)
	if err != nil {
		return Game{}, fmt.Errorf("creating the game: %w", err)
	}
	file = nameLine.ReplaceAllString(file, "name: "+unique)
	if err := c.PutGameYAML(game.Id.String(), file); err != nil {
		return Game{}, fmt.Errorf("uploading %s.yaml: %w", name, err)
	}
	return c.GetGameByID(game.Id.String())
}

// nameLine is a game file's name.
var nameLine = regexp.MustCompile(`(?m)^name: .*$`)

// readGameFile reads testdata/games/<name>.yaml.
func readGameFile(name string) (string, error) {
	path := filepath.Join("testdata", "games", name+".yaml")
	blob, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading game file %s: %w", path, err)
	}
	return string(blob), nil
}

// PutGameYAML replaces a game's content with a game file.
func (c caller) PutGameYAML(gameID, yaml string) error {
	return done(c.api.PutApiGamesIdYamlWithBodyWithResponse(ctx(), gameID, contentYAML, strings.NewReader(yaml)))
}

// GetGameYAML exports a game as a game file.
func (c caller) GetGameYAML(gameID string) (string, error) {
	r, err := c.api.GetApiGamesIdYamlWithResponse(ctx(), gameID, accept(contentYAML))
	if err := done(r, err); err != nil {
		return "", err
	}
	return string(r.Body), nil
}

// accept asks for a content type.
func accept(contentType string) apiclient.RequestEditorFn {
	return header("Accept", contentType)
}

// ReadGame is one game as the caller sees it, with their rights on it.
func (c caller) ReadGame(gameID string) (GameView, error) {
	return decode[GameView](c.api.GetApiGamesIdWithResponse(ctx(), gameID))
}

// GetGameByID is one game.
func (c caller) GetGameByID(gameID string) (Game, error) {
	return decode[Game](c.api.GetApiGamesIdWithResponse(ctx(), gameID))
}

// ListGames is the games the caller sees.
func (c caller) ListGames() ([]Game, error) {
	return c.listGames(apiclient.GetApiGamesParams{})
}

// ListGamesWithFilter is the games the caller sees, filtered ("own",
// "public", …).
func (c caller) ListGamesWithFilter(filter string) ([]Game, error) {
	return c.listGames(apiclient.GetApiGamesParams{Filter: some(gameFilters(filter))})
}

// ListGamesWithSearch is the games the caller sees that match a search.
func (c caller) ListGamesWithSearch(search string) ([]Game, error) {
	return c.listGames(apiclient.GetApiGamesParams{Search: some(search)})
}

// GamePage is one page of the game list, and how many games match in all.
type GamePage struct {
	Items []Game `json:"items"`
	Total int    `json:"total"`
}

// ListGamesPage is one page of the caller's game list, newest first, narrowed
// by every filter given ("own", "favorites", …); limit 0 is all.
func (c caller) ListGamesPage(limit, offset int, filters ...string) (GamePage, error) {
	return decode[GamePage](c.api.GetApiGamesWithResponse(ctx(), &apiclient.GetApiGamesParams{
		Filter: some(gameFilters(filters...)), Limit: some(limit), Offset: some(offset),
	}))
}

// ListGamesSorted is the caller's game list in an order ("name", "asc").
func (c caller) ListGamesSorted(sortBy, sortDir string, filters ...string) ([]Game, error) {
	params := apiclient.GetApiGamesParams{
		SortBy:  some(apiclient.GameSort(sortBy)),
		SortDir: some(apiclient.SortDir(sortDir)),
	}
	if len(filters) > 0 {
		params.Filter = some(gameFilters(filters...))
	}
	return c.listGames(params)
}

// ListGamesUnasked is the caller's game list with no parameter sent at all,
// as a client sends it that leaves every option out.
func (c caller) ListGamesUnasked() (GamePage, error) {
	return decode[GamePage](c.api.GetApiGamesWithResponse(ctx(), nil))
}

// GameInList is a game as the caller's game list shows it, with what the
// caller may do with it (MayEditCard, MayEditContent, MayEditSettings, MayCopy).
func (c caller) GameInList(gameID string) (GameView, error) {
	page, err := decode[struct {
		Items []GameView `json:"items"`
	}](c.api.GetApiGamesWithResponse(ctx(), &apiclient.GetApiGamesParams{}))
	if err != nil {
		return GameView{}, err
	}
	for _, g := range page.Items {
		if g.Id.String() == gameID {
			return g, nil
		}
	}
	return GameView{}, fmt.Errorf("game %s is not in the list", gameID)
}

// listGames reads every game of the list, from its first page holding all.
// The list's items carry a game's keys, so they read as games.
func (c caller) listGames(params apiclient.GetApiGamesParams) ([]Game, error) {
	page, err := decode[struct {
		Items []Game `json:"items"`
	}](c.api.GetApiGamesWithResponse(ctx(), &params))
	return page.Items, err
}

// UpdateGame saves a game's properties; change the game a test read and pass
// it back whole.
func (c caller) UpdateGame(gameID string, game Game) (Game, error) {
	return decode[Game](c.api.PostApiGamesIdWithResponse(ctx(), gameID, GameUpdate{
		Name:                   &game.Name,
		Description:            &game.Description,
		Icon:                   &game.Icon,
		Genre:                  &game.Genre,
		Theme:                  game.Theme,
		Public:                 &game.Public,
		SystemMessageScenario:  &game.SystemMessageScenario,
		SystemMessageGameStart: &game.SystemMessageGameStart,
		ImageStyle:             &game.ImageStyle,
		VoiceStyle:             &game.VoiceStyle,
		Css:                    &game.Css,
		StatusFields:           &game.StatusFields,
		FirstMessage:           game.FirstMessage,
		FirstStatus:            game.FirstStatus,
		FirstImage:             &game.FirstImage,
	}))
}

// ChangeGame sends only the changes in update, and answers with the game as
// the caller now sees it.
func (c caller) ChangeGame(gameID string, update GameUpdate) (GameView, error) {
	return decode[GameView](c.api.PostApiGamesIdWithResponse(ctx(), gameID, update))
}

// RenameGame renames a game, keeping everything else.
func (c caller) RenameGame(gameID, name string) (Game, error) {
	game, err := c.GetGameByID(gameID)
	if err != nil {
		return Game{}, err
	}
	game.Name = name
	return c.UpdateGame(gameID, game)
}

// DeleteGame deletes a game.
func (c caller) DeleteGame(gameID string) error {
	return done(c.api.DeleteApiGamesIdWithResponse(ctx(), gameID))
}

// CloneGame copies a game into the caller's own.
func (c caller) CloneGame(gameID string) (Game, error) {
	return decode[Game](c.api.PostApiGamesIdCloneWithResponse(ctx(), gameID))
}

// GetApiKeyStatus says whether a key can be found to play the game with.
func (c caller) GetApiKeyStatus(gameID string) (bool, error) {
	resp, err := decode[apiclient.GameApiKeyStatusResp](c.api.GetApiGamesIdApiKeyStatusWithResponse(ctx(), gameID))
	return resp.Available, err
}

// GetAvailableKeys is the keys the caller could play the game with.
func (c caller) GetAvailableKeys(gameID string) ([]AvailableKey, error) {
	return decode[[]AvailableKey](c.api.GetApiGamesIdAvailableKeysWithResponse(ctx(), gameID))
}

// SetGameSponsor sponsors a game publicly with a key share.
func (c caller) SetGameSponsor(gameID, shareID string) (Game, error) {
	id, err := uuid.Parse(shareID)
	if err != nil {
		return Game{}, err
	}
	return decode[Game](c.api.PutApiGamesIdSponsorWithResponse(ctx(), gameID, apiclient.GameSponsorReq{ShareId: id}))
}

// RemoveGameSponsor ends a game's public sponsorship.
func (c caller) RemoveGameSponsor(gameID string) (Game, error) {
	return decode[Game](c.api.DeleteApiGamesIdSponsorWithResponse(ctx(), gameID))
}

// FavoriteGame marks a game as a favorite.
func (c caller) FavoriteGame(gameID string) error {
	return done(c.api.PostApiGamesIdFavouriteWithResponse(ctx(), gameID))
}

// UnfavoriteGame removes a game from the favorites.
func (c caller) UnfavoriteGame(gameID string) error {
	return done(c.api.DeleteApiGamesIdFavouriteWithResponse(ctx(), gameID))
}

// --- Shares ---

// CreateGameShare creates a share link for a game, sponsored by a key share;
// maxSessions limits its plays (nil for unlimited).
func (c caller) CreateGameShare(gameID, sponsorShareID string, maxSessions *int) (GameShare, error) {
	id, err := uuid.Parse(sponsorShareID)
	if err != nil {
		return GameShare{}, err
	}
	return decode[GameShare](c.api.PostApiGamesIdSharesWithResponse(ctx(), gameID, apiclient.CreateGameShareReq{
		SponsorKeyShareId: &id,
		MaxSessions:       maxSessions,
	}))
}

// CreateWorkshopGameShare shares a game into a workshop; maxSessions limits
// its plays (nil for unlimited).
func (c caller) CreateWorkshopGameShare(gameID, workshopID string, maxSessions *int) (GameShare, error) {
	id, err := uuid.Parse(workshopID)
	if err != nil {
		return GameShare{}, err
	}
	return decode[GameShare](c.api.PostApiGamesIdSharesWithResponse(ctx(), gameID, apiclient.CreateGameShareReq{
		WorkshopId:  &id,
		MaxSessions: maxSessions,
	}))
}

// SetGameShareLimit changes how many plays a share has left (nil for
// unlimited).
func (c caller) SetGameShareLimit(gameID, shareID string, maxSessions *int) (GameShare, error) {
	return decode[GameShare](c.api.PatchApiGamesIdSharesShareIdWithResponse(ctx(), gameID, shareID, apiclient.UpdateGameShareReq{MaxSessions: maxSessions}))
}

// DeleteGameShare deletes a share link.
func (c caller) DeleteGameShare(gameID, shareID string) error {
	return done(c.api.DeleteApiGamesIdSharesShareIdWithResponse(ctx(), gameID, shareID))
}

// GetPrivateShareStatus is a game's share links.
func (c caller) GetPrivateShareStatus(gameID string) (PrivateShareStatus, error) {
	return decode[PrivateShareStatus](c.api.GetApiGamesIdPrivateShareWithResponse(ctx(), gameID))
}

// GuestGetGameInfo is the game a share link plays.
func (c caller) GuestGetGameInfo(shareToken string) (GuestGameInfo, error) {
	return decode[GuestGameInfo](c.api.GetApiPlayTokenInfoWithResponse(ctx(), shareToken))
}

// --- Voices ---

// VoiceSample has the voice a game of the genre speaks in say its sample, for
// the voice style and scenario as an editor's form holds them.
func (c caller) VoiceSample(genre, voiceStyle, scenario string) (VoiceSample, error) {
	return decode[VoiceSample](c.api.PostApiVoicesSampleWithResponse(ctx(), apiclient.VoiceSampleReq{Genre: genre, VoiceStyle: voiceStyle, Scenario: scenario}))
}

// NewVoice makes a new voice for a voice style and answers its sample.
func (c caller) NewVoice(genre, voiceStyle, scenario string) (VoiceSample, error) {
	return decode[VoiceSample](c.api.PostApiVoicesNewWithResponse(ctx(), apiclient.VoiceSampleReq{Genre: genre, VoiceStyle: voiceStyle, Scenario: scenario}))
}

// ReviewGame has a game's form reviewed: one point to improve.
func (c caller) ReviewGame(form ReviewForm) (Review, error) {
	return decode[Review](c.api.PostApiGamesReviewWithResponse(ctx(), form))
}

// --- Covers ---

// MakeCover has a game's cover drawn, with a hint of what it should show
// (empty: none), and answers with the game as it now is.
func (c caller) MakeCover(gameID, hint string) (GameView, error) {
	return decode[GameView](c.api.PostApiGamesIdCoverWithResponse(ctx(), gameID, apiclient.MakeGameCoverReq{Hint: hint}))
}

// GetCover is a game's cover picture.
func (c caller) GetCover(gameID string) ([]byte, error) {
	r, err := c.api.GetApiGamesIdCoverWithResponse(ctx(), gameID)
	if err := done(r, err); err != nil {
		return nil, err
	}
	return r.Body, nil
}

// SetCover makes a picture, as a game's .cgl file brings it, a game's cover, with
// the signature the file's cover.sig holds, and answers with the game as it
// now is.
func (c caller) SetCover(gameID string, picture []byte, signature string) (GameView, error) {
	return decode[GameView](c.api.PutApiGamesIdCoverWithResponse(ctx(), gameID,
		body("image/png", picture), header("X-Cover-Signature", signature)))
}

// GetGameCgl exports a game as a .cgl file, a zip of its game file and its cover.
func (c caller) GetGameCgl(gameID string) ([]byte, error) {
	r, err := c.api.GetApiGamesIdCglWithResponse(ctx(), gameID)
	if err := done(r, err); err != nil {
		return nil, err
	}
	return r.Body, nil
}

// body sends a raw request body, which a raw endpoint's generated method has
// no argument for.
func body(contentType string, data []byte) apiclient.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Content-Type", contentType)
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.ContentLength = int64(len(data))
		return nil
	}
}

// PNG is a picture of w×h pixels, as a game's .cgl file may bring one.
func PNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// IsJPEG says whether a picture is a JPEG file.
func IsJPEG(picture []byte) bool {
	return bytes.HasPrefix(picture, []byte{0xFF, 0xD8, 0xFF})
}

// IsPNG says whether a picture is a PNG file.
func IsPNG(picture []byte) bool {
	return bytes.HasPrefix(picture, []byte("\x89PNG\r\n\x1a\n"))
}

// --- Catalogues ---

// ListGenres is every genre the engine plays.
func (c caller) ListGenres() ([]Genre, error) {
	return decode[[]Genre](c.api.GetApiGenresWithResponse(ctx()))
}

// ListThemes is every look the playback-engine can wear.
func (c caller) ListThemes() ([]Theme, error) {
	return decode[[]Theme](c.api.GetApiThemesWithResponse(ctx()))
}

// --- Public workshop pages ---

// GetPublicWorkshopPage is a workshop's public page, by its link.
func (c caller) GetPublicWorkshopPage(slug string) (PublicWorkshopPage, error) {
	return decode[PublicWorkshopPage](c.api.GetApiPublicWorkshopsSlugWithResponse(ctx(), slug))
}

// GetPublicWorkshopGameYAML downloads a game shown on a workshop's public
// page as a game file.
func (c caller) GetPublicWorkshopGameYAML(slug, gameID string) (string, error) {
	r, err := c.api.GetApiPublicWorkshopsSlugGamesIdYamlWithResponse(ctx(), slug, gameID, accept(contentYAML))
	if err := done(r, err); err != nil {
		return "", err
	}
	return string(r.Body), nil
}

// --- The platform ---

// Status is the platform's status report.
func (c caller) Status() (map[string]any, error) {
	return decode[map[string]any](c.api.GetApiStatusWithResponse(ctx()))
}

// SetGamePublic switches a game's public flag, keeping everything else: it
// reads the game as the caller and saves it back.
func (c caller) SetGamePublic(gameID string, public bool) (Game, error) {
	game, err := c.GetGameByID(gameID)
	if err != nil {
		return Game{}, err
	}
	game.Public = public
	return c.UpdateGame(gameID, game)
}

// GetPublicWorkshopPageJSON is a workshop's public page as the platform
// sends it, so a test can check what the page never contains.
func (c caller) GetPublicWorkshopPageJSON(slug string) (string, error) {
	r, err := c.api.GetApiPublicWorkshopsSlugWithResponse(ctx(), slug)
	if err := done(r, err); err != nil {
		return "", err
	}
	return string(r.Body), nil
}

// gameFilters is filters by name as the client's type; the server refuses
// those it does not know.
func gameFilters(names ...string) []apiclient.GameFilter {
	out := make([]apiclient.GameFilter, len(names))
	for i, n := range names {
		out[i] = apiclient.GameFilter(n)
	}
	return out
}
