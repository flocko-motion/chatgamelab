// Command engineserver serves one game standalone, with the embedded
// playback-engine at /api/engine/playback-engine/: the engine's routes, at the
// platform's paths, through schemaf's endpoint pipeline.
//
// A game file wraps what a platform keeps apart: the stored spec, and the live
// offers a platform would build from its key cascade.
//
//	{ "spec": { … }, "offers": [ … ] }
//
// The playback-engine opened with ?new&game=<name> creates a session from the
// file, checking each offer's key first; the session is kept in memory and
// resumes under its ID until the server exits.
//
//	engineserver examples/stopped-clock.json
//	engineserver --api-key sk-... examples/stopped-clock.json
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"engine"
	"engine/api"
	"engine/internal/devconfig"
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

	if *describe {
		s, err := engine.Launch(context.Background(), spec, offers)
		if err != nil {
			log.Fatalf("launch: %v", err)
		}
		fmt.Print(s.Describe())
		s.Close()
		return
	}

	// In-memory persistence: sessions are forgotten on exit.
	store := &engine.MemoryPersist{}
	name := strings.TrimSuffix(filepath.Base(flag.Arg(0)), filepath.Ext(flag.Arg(0)))
	srv := api.New(api.Options{
		Dev: func() bool { return true },
		Create: func(ctx context.Context, req api.CreateRequest) (*engine.Session, error) {
			usable, err := checked(ctx, offers)
			if err != nil {
				return nil, err
			}
			spec := spec
			spec.ID = newID()
			spec.Language = req.Language
			spec.Persist = store
			s, err := engine.Launch(context.WithoutCancel(ctx), spec, usable)
			if err == nil {
				log.Printf("session %s created", spec.ID)
			}
			return s, err
		},
		Resume: func(ctx context.Context, id string) (*engine.Session, error) {
			return engine.Resume(context.WithoutCancel(ctx), id, store, offers)
		},
	})

	fmt.Printf("genre=%s game=%s offers=%d\n", spec.Genre, name, len(offers))
	fmt.Printf("  api key:  %s\n", keySource)
	fmt.Printf("  playback-engine  http://localhost%s/api/engine/playback-engine/?new&game=%s\n", *addr, name)
	log.Fatal(http.ListenAndServe(*addr, api.Mux(srv.SchemafRoutes())))
}

// checked keeps the offers whose key the provider accepts, as a platform
// does before creating a session; with none, the first refusal answers.
func checked(ctx context.Context, offers []engine.Offer) ([]engine.Offer, error) {
	var usable []engine.Offer
	var first error
	for _, offer := range offers {
		if err := engine.CheckOffer(ctx, offer); err != nil {
			log.Printf("offer %s: %v", offer.ID, err)
			if first == nil {
				first = err
			}
			continue
		}
		usable = append(usable, offer)
	}
	if len(usable) == 0 {
		return nil, first
	}
	return usable, nil
}

// newID is a session's handle, unguessable because it is a bearer token.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
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

	key, err := devconfig.APIKey(string(platform))
	if err != nil {
		return nil, "", err
	}
	cfg, _ := devconfig.Path()
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
