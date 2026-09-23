// Package player holds the embedded frontend.
package player

import (
	"embed"
	"io/fs"
)

// Package player holds the embedded frontend. It ships inside the engine and is
// served from the engine's own subtree, so it addresses the API with relative
// URLs — the same ones under the platform's mount point and under a standalone
// server.
//
//go:embed web
var assets embed.FS

// FS is the player's document root.
func FS() fs.FS {
	sub, err := fs.Sub(assets, "web")
	if err != nil {
		panic(err)
	}
	return sub
}
