package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"engine"
)

// Every example spec in the repo must parse and name a genre the engine knows,
// so a broken example fails here rather than in front of a customer.
func TestExampleSpecsLoad(t *testing.T) {
	matches, err := filepath.Glob("../../examples/*.json")
	if err != nil || len(matches) == 0 {
		t.Fatalf("no example specs found: %v", err)
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			spec, err := loadSpec(path)
			if err != nil {
				t.Fatal(err)
			}
			if spec.Genre == "" {
				t.Error("spec names no genre")
			}
			if spec.Scenario == "" || spec.Guardrail == "" {
				t.Error("a spec wants both a scenario and a guardrail")
			}
		})
	}
}

// A spec must never carry a secret: it is persisted as a blob, so a key in the
// file would be written to the database once per session.
func TestSpecFilesCarryNoSecret(t *testing.T) {
	matches, _ := filepath.Glob("../../examples/*.json")
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{"apiKey", "api_key", "sk-"} {
			if strings.Contains(string(raw), banned) {
				t.Errorf("%s contains %q; keys are injected, never written into a spec", path, banned)
			}
		}
	}
}

func TestFlagBeatsConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	keys, source, err := keyFunc(engine.PlatformOpenAI, "sk-flag")
	if err != nil {
		t.Fatal(err)
	}
	got, err := keys(context.Background())
	if err != nil || got != "sk-flag" {
		t.Fatalf("got %q, %v", got, err)
	}
	// Where it came from, and enough of it to recognise — never the whole key.
	if !strings.Contains(source, "--api-key flag") {
		t.Errorf("source %q does not say where the key came from", source)
	}
	if !strings.Contains(source, "…flag") {
		t.Errorf("source %q does not show the key's tail", source)
	}
	if strings.Contains(source, "sk-flag") {
		t.Errorf("source %q contains the whole key", source)
	}
}

func TestConfigFallbackAndPlaceholder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	os.MkdirAll(filepath.Join(home, ".chatgamelab"), 0o755)
	cfgPath := filepath.Join(home, ".chatgamelab", "config.yaml")

	os.WriteFile(cfgPath, []byte("platforms:\n  openai:\n    apikey: sk-from-config\n"), 0o600)
	keys, source, err := keyFunc(engine.PlatformOpenAI, "")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := keys(context.Background())
	if got != "sk-from-config" {
		t.Errorf("got %q, want the config key", got)
	}
	if !strings.Contains(source, "config.yaml") {
		t.Errorf("source %q should name the config file", source)
	}

	// The placeholder a fresh config ships with is not a key.
	os.WriteFile(cfgPath, []byte("platforms:\n  openai:\n    apikey: your-api-key-here\n"), 0o600)
	if _, _, err := keyFunc(engine.PlatformOpenAI, ""); err == nil {
		t.Error("the placeholder must not count as a configured key")
	}
}

func TestNoKeyAnywhereIsRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, _, err := keyFunc(engine.PlatformOpenAI, ""); err == nil {
		t.Fatal("expected a missing-key error")
	}
}

// A key is what a session spends, so the launcher says which one it found. Four
// characters is enough to recognise an account and not enough to use it.
func TestFingerprint(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{"sk-proj-abcdefghijkl3f9Q", "…3f9Q"},
		{"short", "…hort"},
		{"abcd", "…"},
		{"", "…"},
	} {
		if got := fingerprint(tc.key); got != tc.want {
			t.Errorf("fingerprint(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}

	// Whatever it prints must not be enough to make a call with: at most four
	// characters of the key itself, and never the part that identifies it.
	const key = "sk-proj-averylongsecretvalue"
	shown := strings.TrimPrefix(fingerprint(key), "…")
	if len([]rune(shown)) > 4 {
		t.Errorf("fingerprint reveals %q, which is more of the key than it should", shown)
	}
	if strings.HasPrefix(key, shown) {
		t.Errorf("fingerprint reveals %q, the start of the key rather than its tail", shown)
	}
}
