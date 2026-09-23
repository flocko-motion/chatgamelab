// Command engineserver is the standalone shape: a hand-assembled SessionSpec in
// a file, the engine's own handlers served directly, and the embedded player at
// /player/. It is how the gpt-live spike is run.
//
//	engineserver examples/npc-live.json
//	engineserver --api-key sk-... examples/npc-live.json
package main

import (
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

	spec, err := loadSpec(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	keySource := "none needed"
	if spec.Platform != "" && spec.Platform != engine.PlatformMock {
		var keys engine.KeyFunc
		keys, keySource, err = keyFunc(spec.Platform, *apiKey)
		if err != nil {
			log.Fatal(err)
		}
		spec.Keys = keys
	}

	// Persist is injected here rather than named in the spec file, because it
	// is behaviour. A standalone build would hand in SQLite; this one keeps the
	// session in memory and forgets it on exit.
	store := &engine.MemoryPersist{}
	spec.Persist = store

	s, err := engine.Launch(context.Background(), spec)
	if err != nil {
		log.Fatalf("launch: %v", err)
	}
	defer s.Close()

	if *describe {
		fmt.Print(s.Describe())
		return
	}

	srv := api.New()
	srv.Register(spec.ID, s)

	fmt.Printf("genre=%s platform=%s session=%s\n", spec.Genre, spec.Platform, spec.ID)
	fmt.Printf("  api key from: %s\n", keySource)
	fmt.Printf("  player  http://localhost%s/player/?session=%s\n", *addr, spec.ID)
	fmt.Printf("  graph   http://localhost%s/sessions/%s/graph\n", *addr, spec.ID)
	log.Fatal(http.ListenAndServe(*addr, srv.Handler()))
}

// loadSpec reads a hand-assembled spec. It carries no key: secrets are
// injected, never written into a file that gets persisted as a blob.
func loadSpec(path string) (engine.SessionSpec, error) {
	var spec engine.SessionSpec

	raw, err := os.ReadFile(path)
	if err != nil {
		return spec, err
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return spec, fmt.Errorf("%s: %w", path, err)
	}
	return spec, nil
}

// keyFunc is the standalone counterpart of the resolver the platform injects:
// the --api-key flag first, then the platform's entry in
// ~/.chatgamelab/config.yaml. It reports which source it used, so the operator
// never has to guess which key a session ran on.
func keyFunc(platform engine.Platform, override string) (engine.KeyFunc, string, error) {
	if override != "" {
		return func(context.Context) (string, error) { return override, nil }, "--api-key flag", nil
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
	return func(context.Context) (string, error) { return key, nil }, cfg, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: engineserver [flags] <sessionspec.json>\n\nflags:\n")
	flag.PrintDefaults()
}
