package api_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"engine"
)

// The headless player follows a real session against a real engine, with no
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
func TestHeadlessPlayerFollowsASession(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping the headless player integration test")
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
	})
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
		t.Fatalf("headless player failed: %v", err)
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
