package api

import (
	"net/http"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine/web"
)

// playbackPrefix is where the playback-engine's files are served, under the
// engine's routes, so the page reaches them with the same origin.
const playbackPrefix = "/api/engine/playback-engine/"

var playbackFiles = http.StripPrefix(playbackPrefix, http.FileServerFS(web.FS()))

// PlaybackIndexEndpoint is the playback-engine's page.
//
// Opened with ?session=<id> to play a session, or ?new&game=<id> or
// ?new&share=<token> to start one.
type PlaybackIndexEndpoint struct{}

func (PlaybackIndexEndpoint) Method() string             { return "GET" }
func (PlaybackIndexEndpoint) Path() string               { return "/api/engine/playback-engine/" }
func (PlaybackIndexEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (PlaybackIndexEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	playbackFiles.ServeHTTP(w, r)
	return nil
}

// PlaybackFileEndpoint is one of the playback-engine's files beside its page.
//
// Its scripts, stylesheet and worklet.
type PlaybackFileEndpoint struct{}

func (PlaybackFileEndpoint) Method() string             { return "GET" }
func (PlaybackFileEndpoint) Path() string               { return "/api/engine/playback-engine/{file}" }
func (PlaybackFileEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (PlaybackFileEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	playbackFiles.ServeHTTP(w, r)
	return nil
}

// PlaybackNestedFileEndpoint is one of the playback-engine's files in a folder.
//
// Its chunks, fonts, art and language files.
type PlaybackNestedFileEndpoint struct{}

func (PlaybackNestedFileEndpoint) Method() string { return "GET" }
func (PlaybackNestedFileEndpoint) Path() string {
	return "/api/engine/playback-engine/{dir}/{file}"
}
func (PlaybackNestedFileEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (PlaybackNestedFileEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	playbackFiles.ServeHTTP(w, r)
	return nil
}
