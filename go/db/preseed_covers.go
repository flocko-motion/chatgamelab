// package: db / database access and repository layer
// type:    data
// job:     give the demo games the covers kept beside their files, drawn once by cmd/seedcovers.
// limits:  draws nothing; does not run in production.
package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// A demo game's cover is kept beside its file, foo.yaml's as foo.cover.jpg:
// the 600-pixel JPEG a cover is stored at. It is drawn once, by
// cmd/seedcovers, so a fresh instance has covers without drawing them again.
const devCoverSuffix = ".cover.jpg"

// DevGameFiles lists the demo game files, in file name order.
func DevGameFiles() ([]string, error) {
	dir, err := checkoutPath(devGamesDir)
	if err != nil {
		return nil, err
	}
	return filepath.Glob(filepath.Join(dir, "*.yaml"))
}

// DevCoverPath is where the cover of the demo game file at path is kept.
func DevCoverPath(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + devCoverSuffix
}

// ReadGameFile reads a demo game file (readGameFile).
func ReadGameFile(path string) (obj.GameFile, error) { return readGameFile(path) }

// preseedDevCovers gives every demo game without a cover the one kept beside
// its file, if there is one. It matches the dev users' games to the files by
// name, so it also fills in covers on an instance seeded before they were
// kept; a game that has a cover keeps it.
func preseedDevCovers(ctx context.Context) {
	paths, err := DevGameFiles()
	if err != nil || len(paths) == 0 {
		return
	}
	type owned struct{ id, owner uuid.UUID }
	byName := map[string][]owned{}
	for _, slot := range devUserSlots() {
		games, err := GetGames(ctx, &slot.id, &GetGamesFilters{Filter: "own"})
		if err != nil {
			log.Warn("failed to get games", "user_id", slot.id, "error", err)
			return
		}
		for _, g := range games {
			byName[g.Name] = append(byName[g.Name], owned{g.ID, slot.id})
		}
	}

	for _, path := range paths {
		image, err := os.ReadFile(DevCoverPath(path))
		if err != nil {
			continue // no cover drawn for this game yet
		}
		file, err := readGameFile(path)
		if err != nil {
			continue
		}
		for _, game := range byName[file.Name] {
			if _, err := queries().V2GameCover(ctx, game.id); !errors.Is(err, sql.ErrNoRows) {
				continue // has a cover, or the lookup failed
			}
			err := queries().V2SetGameCover(ctx, V2SetGameCoverParams{
				GameID: game.id, Image: image, CreatedBy: game.owner,
			})
			if err != nil {
				log.Warn("failed to seed demo cover", "game_id", game.id, "error", err)
			}
		}
	}
}
