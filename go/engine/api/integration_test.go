package api_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"engine"
	"engine/api"
)

// The headless playback-engine follows a real session against a real engine, with no
// browser. Go drives: it owns the server, so there is no port to guess and no
// readiness to poll, and `go test` stays the one command that runs everything.
//
// It watches rather than plays, and the limit is honest rather than a gap in
// the harness: a live genre's audio runs over WebRTC between a browser and the
// model, and node has neither a microphone nor a peer connection. What it does
// prove is everything up to the moment a player would join — the wiring is
// served, the portrait is made, the gate opens, and somebody is invited to
// connect — which is the whole init stem running with nobody acting.
//
// Skipped when node is absent, because the Go build never requires a JavaScript
// toolchain — web/dist is committed.
func TestHeadlessPlaybackEngineFollowsASession(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping the headless playback-engine integration test")
	}

	harness, err := filepath.Abs("../web/dist/play.js")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:     engine.GenreNPCLive,
		ID:        "integration",
		Scenario:  "You are the keeper of a bridge. You never concede passage.",
		Guardrail: "Suitable for a 13-year-old.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "integration", session)

	runCtx, cancelRun := context.WithTimeout(ctx, 30*time.Second)
	defer cancelRun()

	out, err := exec.CommandContext(runCtx, node, harness, srv.URL, "integration").CombinedOutput()
	t.Logf("headless run:\n%s", out)
	if err != nil {
		t.Fatalf("headless playback-engine failed: %v", err)
	}

	text := string(out)
	for _, want := range []string{
		"started:    true",   // the gate opened with nobody acting
		"invited:    true",   // and a player was asked to connect
		"live-session",       // the wiring reached the client
		"--- transcript ---", // the harness completed its report
	} {
		if !strings.Contains(text, want) {
			t.Errorf("headless output missing %q", want)
		}
	}
	if strings.Contains(text, "image:      (none)") {
		t.Error("the portrait never reached the client, so the init stem did not finish")
	}
}

// The headless playback-engine goes through the options dialog: it finds the session
// choosing, has a choice outside the caveats refused, reloads and finds the
// dialog again, then chooses and follows the game as before.
func TestHeadlessPlaybackEngineChooses(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping the headless playback-engine integration test")
	}
	harness, err := filepath.Abs("../web/dist/play.js")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live := engine.Toggles{Images: true, AudioLive: true}
	custom := engine.MockOffer()
	custom.ID = "custom"
	custom.Caveats = engine.Caveats{
		Allow:        map[engine.Tier]engine.Toggles{engine.TierLow: live, engine.TierMedium: live},
		Default:      engine.Choice{Tier: engine.TierMedium, Toggles: live},
		Customizable: true,
	}
	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:     engine.GenreNPCLive,
		ID:        "choosing",
		Scenario:  "You are the keeper of a bridge.",
		Guardrail: "Suitable for a 13-year-old.",
	}, []engine.Offer{engine.MockOffer(), custom})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "choosing", session)

	runCtx, cancelRun := context.WithTimeout(ctx, 30*time.Second)
	defer cancelRun()

	out, err := exec.CommandContext(runCtx, node, harness, srv.URL, "choosing",
		"--bad=custom:high:images,audio-live", "--choose=custom:low:images,audio-live").CombinedOutput()
	t.Logf("headless run:\n%s", out)
	if err != nil {
		t.Fatalf("headless playback-engine failed: %v", err)
	}

	text := string(out)
	for _, want := range []string{
		"choosing:   true",
		"offers:     2",
		"refused:    true",
		"reloaded:   choosing=true",
		"settled:    low",
		"started:    true",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("headless output missing %q", want)
		}
	}
}

// The headless playback-engine starts a session the way the page does for
// ?new&game=…: it creates one, sees the same one answer a repeated one-time
// key, rebuilds as a reload would, and follows the game.
func TestHeadlessPlaybackEngineCreates(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping the headless playback-engine integration test")
	}
	harness, err := filepath.Abs("../web/dist/play.js")
	if err != nil {
		t.Fatal(err)
	}

	store := &engine.MemoryPersist{}
	var launches atomic.Int32
	a := api.New(api.Options{
		Dev: true,
		Create: func(ctx context.Context, req api.CreateRequest) (*engine.Session, error) {
			n := launches.Add(1)
			return engine.Launch(context.WithoutCancel(ctx), engine.SessionSpec{
				Genre:     engine.GenreNPCLive,
				ID:        fmt.Sprintf("%s-%d", req.Game, n),
				Scenario:  "You are the keeper of a bridge.",
				Guardrail: "Suitable for a 13-year-old.",
				Language:  req.Language,
				Persist:   store,
			}, mockOffers())
		},
		Resume: func(ctx context.Context, id string) (*engine.Session, error) {
			return engine.Resume(context.WithoutCancel(ctx), id, store, mockOffers())
		},
	})
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, node, harness, srv.URL, "--new=game:bridge", "--once=k1").CombinedOutput()
	t.Logf("headless run:\n%s", out)
	if err != nil {
		t.Fatalf("headless playback-engine failed: %v", err)
	}

	text := string(out)
	for _, want := range []string{
		"created:    bridge-1",
		"repeated:   same",
		"resumed:    true",
		"started:    true",
		"invited:    true",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("headless output missing %q", want)
		}
	}
	if n := launches.Load(); n != 1 {
		t.Errorf("%d launches, want 1", n)
	}
}
