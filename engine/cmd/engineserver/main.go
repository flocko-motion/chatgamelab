// Command engineserver serves one game standalone, with the embedded player at /player/.
//
// A game file wraps what a platform keeps apart: the stored spec, and the live
// offers a platform would build from its key cascade.
//
//	{ "spec": { … }, "offers": [ … ] }
//
//	engineserver examples/stopped-clock.json
//	engineserver --api-key sk-... examples/stopped-clock.json
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"engine"
	"engine/api"
)

func main() {
	var (
		addr     = flag.String("addr", ":8080", "listen address")
		apiKey   = flag.String("api-key", "", "override the key in the spec")
		describe = flag.Bool("describe", false, "print the wiring and exit")
	)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	game, err := loadGame(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	spec, offers := game.Spec, game.Offers

	keySource := "none needed: this game makes no offer, so it runs on the mock"
	// A dev launcher, so a game that makes no offer runs on the mock.
	if len(offers) == 0 {
		offers = []engine.Offer{engine.MockOffer()}
	}
	for i, offer := range offers {
		if offer.Platform == engine.PlatformMock {
			keySource = "none needed: the mock platform makes no calls"
			continue
		}
		var keys engine.KeyFunc
		keys, keySource, err = keyFunc(offer.Platform, *apiKey)
		if err != nil {
			log.Fatal(err)
		}
		offers[i].Keys = keys
	}

	// In-memory persistence: the session is forgotten on exit.
	store := &engine.MemoryPersist{}
	spec.Persist = store

	s, err := engine.Launch(context.Background(), spec, offers)
	if err != nil {
		log.Fatalf("launch: %v", err)
	}
	defer s.Close()

	if *describe {
		fmt.Print(s.Describe())
		return
	}

	srv := api.New(api.Options{Dev: true})
	srv.RegisterOnly(spec.ID, s)

	if settled := s.Snapshot().Settled; settled != nil {
		fmt.Printf("genre=%s platform=%s tier=%s toggles=%v session=%s\n",
			spec.Genre, settled.Platform, settled.Choice.Tier, settled.Choice.Toggles.On(), spec.ID)
	} else {
		fmt.Printf("genre=%s session=%s: waiting for the player to choose among %d offers\n",
			spec.Genre, spec.ID, len(offers))
	}
	fmt.Printf("  api key:  %s\n", keySource)
	fmt.Printf("  player  http://localhost%s/player/\n", *addr)
	fmt.Printf("  graph   http://localhost%s/sessions/%s/graph\n", *addr, spec.ID)
	log.Fatal(http.ListenAndServe(*addr, srv.Handler()))
}

// game is a dev launcher's file: the spec a platform would store, and the
// offers it would build live. Keeping them apart in the file keeps the offers
// from being mistaken for something stored.
type game struct {
	Spec   engine.SessionSpec `json:"spec"`
	Offers []engine.Offer     `json:"offers,omitempty"`
}

// loadGame reads a hand-assembled game file, which carries no key.
func loadGame(path string) (game, error) {
	var g game

	raw, err := os.ReadFile(path)
	if err != nil {
		return g, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// A field the format does not know, such as offers inside the spec, is a mistake.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&g); err != nil {
		return g, fmt.Errorf("%s: %w", path, err)
	}
	return g, nil
}

// keyFunc reads --api-key, then ~/.chatgamelab/config.yaml, reporting the source so the account is visible.
func keyFunc(platform engine.Platform, override string) (engine.KeyFunc, string, error) {
	if override != "" {
		return func(context.Context) (string, error) { return override, nil },
			fingerprint(override) + " (--api-key flag)", nil
	}

	key, err := apiKeyFromConfig(string(platform))
	if err != nil {
		return nil, "", err
	}
	cfg, _ := configPath()
	if key == "" {
		return nil, "", fmt.Errorf("no API key for platform %q: pass --api-key, or set platforms.%s.apikey in %s",
			platform, platform, cfg)
	}
	return func(context.Context) (string, error) { return key, nil },
		fingerprint(key) + " (" + cfg + ")", nil
}

// fingerprint is enough of a key to recognise and too little to use.
func fingerprint(key string) string {
	const shown = 4
	if len(key) <= shown {
		return "…"
	}
	return "…" + key[len(key)-shown:]
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: engineserver [flags] <game.json>\n\nflags:\n")
	flag.PrintDefaults()
}
