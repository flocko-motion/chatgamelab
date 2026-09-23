// Package web holds the engine's player: the frontend that talks to the
// session endpoints.
//
// Sources live in src/ and are built into dist/ by `make web`. Only dist/ is
// embedded, and it is committed, so the module builds and the portal embeds it
// without anyone needing a JavaScript toolchain.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var assets embed.FS

// FS is the player's document root, served from the engine's own subtree so the
// page can address the API with relative URLs — the same build under the
// platform's mount point and under a standalone server.
func FS() fs.FS {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
