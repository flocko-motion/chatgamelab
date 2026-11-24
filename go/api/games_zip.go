// package: api / schemaf endpoints
// type:    logic
// job:     serves a game as a zip (its game file and its cover) and takes a cover brought in from one
// limits:  the file's shape lives in obj (GameFile); rights and storage in the core; checking a brought-in cover is session/cover.go's
package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"cgl/api/wire"
	"cgl/core"
	"cgl/obj"
	"cgl/picture"
	"cgl/session"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"gopkg.in/yaml.v3"
)

// MaxGameFileBytes caps an uploaded game file, zip or YAML, and a cover
// brought in from one. A stored cover is a JPEG of tens of kilobytes and a
// game file a few, so a megabyte leaves ample room and turns away what is no
// game file.
const MaxGameFileBytes = 1 << 20

// The names inside a game's zip.
const (
	zipGameFile = "game.yaml"
	zipCover    = "cover.jpg"
	// zipCoverSig is the cover's signature (picture.Sign), which a server
	// asks for before it takes the cover back.
	zipCoverSig = "cover.sig"
)

// CoverSignatureHeader carries a brought-in cover's signature, as its zip
// held it in cover.sig.
const CoverSignatureHeader = "X-Cover-Signature"

// GetGameZipEndpoint exports a game as a zip with its cover.
//
// The zip holds game.yaml, the game file as the YAML export writes it, and,
// when the game has a cover, cover.jpg and cover.sig, the signature an import
// asks for. It takes seeing the game's content (hasContent; 403 otherwise).
type GetGameZipEndpoint struct{}

func (GetGameZipEndpoint) Method() string             { return "GET" }
func (GetGameZipEndpoint) Path() string               { return "/api/games/{id}/zip" }
func (GetGameZipEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetGameZipEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	gameID, err := parseID(r.PathValue("id"), "Invalid game ID")
	if err != nil {
		return err
	}
	card, err := core.ReadGameCard(r.Context(), gameID)
	return writeGameZip(w, r, card, err)
}

// GetPublicWorkshopGameZipEndpoint exports a public workshop's game as a zip.
//
// The zip is the one GetGameZipEndpoint answers, for a game listed on a
// public workshop page.
type GetPublicWorkshopGameZipEndpoint struct{}

func (GetPublicWorkshopGameZipEndpoint) Method() string { return "GET" }
func (GetPublicWorkshopGameZipEndpoint) Path() string {
	return "/api/public/workshops/{slug}/games/{id}/zip"
}
func (GetPublicWorkshopGameZipEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetPublicWorkshopGameZipEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	gameID, err := parseID(r.PathValue("id"), "Invalid game ID")
	if err != nil {
		return err
	}
	card, err := core.PublicWorkshopGame(r.Context(), r.PathValue("slug"), gameID)
	return writeGameZip(w, r, card, err)
}

// writeGameZip answers with the game's zip, built whole before the first
// byte, so a failure is still an error answer.
func writeGameZip(w http.ResponseWriter, r *http.Request, card *core.GameCard, err error) error {
	file, err := wire.GameFileOf(card, err)
	if err != nil {
		return err
	}
	var cover []byte
	switch found, err := card.Cover(r.Context()); {
	case err == nil:
		cover = found.Image
	case !errors.Is(err, core.ErrNotFound):
		return err
	}
	body, err := gameZip(file, cover)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="game.zip"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, err = w.Write(body)
	return err
}

// gameZip packs a game file and its cover with its signature; no cover, no
// cover.jpg and no cover.sig.
func gameZip(file obj.GameFile, cover []byte) ([]byte, error) {
	text, err := gameFileYAML(file)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(zipGameFile)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(text); err != nil {
		return nil, err
	}
	if len(cover) > 0 {
		// A JPEG is compressed already.
		f, err := zw.CreateHeader(&zip.FileHeader{Name: zipCover, Method: zip.Store})
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(cover); err != nil {
			return nil, err
		}
		f, err = zw.Create(zipCoverSig)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(picture.Sign(cover))); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// gameFileYAML writes a game file as the YAML export does: the JSON body,
// field names and order kept, in block style.
func gameFileYAML(file obj.GameFile) ([]byte, error) {
	j, err := json.Marshal(file)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(j, &node); err != nil {
		return nil, err
	}
	var plain func(*yaml.Node)
	plain = func(n *yaml.Node) {
		n.Style = 0
		for _, c := range n.Content {
			plain(c)
		}
	}
	plain(&node)
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	if err := enc.Encode(&node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// SetGameCoverEndpoint takes a game's cover from its zip.
//
// It answers with the game as it now is (coverAt set). The body is the picture, PNG or JPEG, at most 1 MB (413 file_too_large
// otherwise), and the X-Cover-Signature header the zip's cover.sig; one
// without a fitting signature is 422 cover_unsigned, one that is no picture
// 400 not_a_picture.
//
// It takes mayEditCard (403 otherwise). The picture is checked against the
// youth-protection rule for the youngest players, as a drawn cover keeps to
// it, and the caller pays for the check as for drawing a cover: no key, 400
// no_api_key; a key its provider rejects, 400 invalid_api_key; a provider
// that fails, 503 ai_error. A picture that does not keep to the rule is 422
// cover_refused, and the game keeps the cover it had.
type SetGameCoverEndpoint struct{}

func (SetGameCoverEndpoint) Method() string             { return "PUT" }
func (SetGameCoverEndpoint) Path() string               { return "/api/games/{id}/cover" }
func (SetGameCoverEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetGameCoverEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	gameID, err := parseID(r.PathValue("id"), "Invalid game ID")
	if err != nil {
		return err
	}
	upload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxGameFileBytes))
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return schemafapi.Error(schemafapi.ErrTooLarge, obj.ErrCodeFileTooLarge, "the cover is larger than 1 MB")
	}
	if err != nil {
		return schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeInvalidInput, "the cover could not be read")
	}
	view, err := wire.GameViewOf(r.Context())(session.ImportGameCover(r.Context(), gameID, upload, r.Header.Get(CoverSignatureHeader)))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(view)
}
