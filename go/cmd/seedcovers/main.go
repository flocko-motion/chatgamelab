// Command seedcovers draws the covers of the demo games once and keeps them
// in the repo, beside the game files (testing/testdata/games/foo.cover.jpg),
// so a fresh dev instance shows covers without drawing them again
// (db/preseed_covers.go).
//
// It draws as the editor does (engine.MakeCover): on OpenAI, at the cheapest
// tier that draws pictures, under the youth-protection rule a dev instance
// starts with, scaled to the width a cover is stored at. It costs money and
// is run by hand, never by `make test`. The key comes from OPENAI_API_KEY, or
// from ~/.chatgamelab/config.yaml as the engine's dev tools read it.
//
//	go run ./cmd/seedcovers                    # every demo game without a cover
//	go run ./cmd/seedcovers -force pirate-cove # draw this one again
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cgl/db"
	"cgl/picture"
	"engine"
)

func main() {
	force := flag.Bool("force", false, "draw covers that exist again")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: seedcovers [-force] [game ...]   (games by file name, without .yaml)")
		flag.PrintDefaults()
	}
	flag.Parse()

	key, err := openAIKey()
	if err != nil {
		fail(err)
	}
	paths, err := db.DevGameFiles()
	if err != nil {
		fail(err)
	}
	offer := engine.Offer{
		Platform: engine.PlatformOpenAI,
		ID:       "seedcovers",
		Keys:     func(context.Context) (string, error) { return key, nil },
		// The cheapest tier that draws pictures, as every cover is (CoverTier).
		Caveats: engine.Caveats{
			Allow:   map[engine.Tier]engine.Toggles{engine.TierLow: {Images: true}},
			Default: engine.Choice{Tier: engine.TierLow, Toggles: engine.Toggles{Images: true}},
		},
	}

	var total float64
	drawn, failed := 0, 0
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if flag.NArg() > 0 && !slices.Contains(flag.Args(), name) {
			continue
		}
		out := db.DevCoverPath(path)
		if _, err := os.Stat(out); err == nil && !*force {
			continue
		}
		cost, err := draw(path, out, offer)
		total += cost
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "%-24s failed: %v\n", name, err)
			continue
		}
		drawn++
		fmt.Printf("%-24s drawn  $%.4f\n", name, cost)
	}
	fmt.Printf("%d drawn, %d failed, $%.4f\n", drawn, failed, total)
	if failed > 0 {
		os.Exit(1)
	}
}

// draw draws the cover of the game file at path and writes it to out;
// it returns what it cost.
func draw(path, out string, offer engine.Offer) (float64, error) {
	file, err := db.ReadGameFile(path)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cover, err := engine.MakeCover(ctx, engine.CoverSpec{
		Title: file.Name, Description: file.Description, Scenario: file.SystemMessageScenario,
		ImageStyle: file.ImageStyle, Guardrail: db.DevSiteConstraintU13,
	}, offer)
	cost := cover.Usage.TotalCost
	if err != nil {
		return cost, err
	}
	image, err := picture.Scale(cover.Image, picture.CoverWidth)
	if err != nil {
		return cost, err
	}
	return cost, os.WriteFile(out, image, 0o644)
}

// openAIKey is the key from OPENAI_API_KEY, else the one in the developer's
// config, ~/.chatgamelab/config.yaml (db.DevPlatformKey).
func openAIKey() (string, error) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key, nil
	}
	key, err := db.DevPlatformKey("openai")
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", errors.New("no OpenAI key: set OPENAI_API_KEY or platforms.openai.apikey in ~/.chatgamelab/config.yaml")
	}
	return key, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "seedcovers:", err)
	os.Exit(1)
}
