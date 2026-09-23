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

// The headless player plays a full session against a real engine, with no
// browser. Go drives: it owns the server, so there is no port to guess and no
// readiness to poll, and `go test` stays the one command that runs everything.
//
// Skipped when node is absent, because the Go build never requires a JavaScript
// toolchain — web/dist is committed.
func TestHeadlessPlayerPlaysASession(t *testing.T) {
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
		Genre: engine.GenreNPCLive,
		ID:    "integration",
		// No script: the headless player is the one speaking, which is what
		// makes this a round trip rather than a broadcast.
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
		"You shall not pass", // the character held at first
		"observer",           // the observer section rendered
		"--- transcript ---", // the harness completed its report
	} {
		if !strings.Contains(text, want) {
			t.Errorf("headless output missing %q", want)
		}
	}
	if strings.Contains(text, "utterances:  0") {
		t.Error("headless player received no utterances")
	}
}
