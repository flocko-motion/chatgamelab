package legal

import (
	"strings"
	"testing"
	"time"
)

// The current declaration is version 1.0, whole, with a stable hash and a
// title without Markdown.
func TestCurrentDeclaration(t *testing.T) {
	d := CurrentDeclaration()
	if d.Version != "1.0" || d.Date != "2026-11-12" {
		t.Fatalf("unexpected version/date: %s %s", d.Version, d.Date)
	}
	if d.Title() != "Einverständniserklärung der Einrichtung zur Nutzung eines Organisationszugangs im ChatGameLab" {
		t.Fatalf("unexpected title: %q", d.Title())
	}
	if !strings.Contains(d.Body(), "10. Ich nehme zustimmend zur Kenntnis") {
		t.Fatal("text is missing point 10")
	}
	if strings.Contains(d.Body(), "**") {
		t.Fatal("the body still carries the title's Markdown")
	}
	if len(d.Hash) != 64 {
		t.Fatalf("hash should be hex sha256, got %q", d.Hash)
	}
	if again := CurrentDeclaration(); again.Hash != d.Hash {
		t.Fatal("hash is not stable")
	}
	if d.Consent == "" {
		t.Fatal("no consent wording")
	}
	if _, ok := DeclarationByVersion("0.9"); ok {
		t.Fatal("unknown version must not resolve")
	}
}

// Seven suggestions, youngest first, each fitting an organisation's
// restriction.
func TestYouthProtectionSuggestions(t *testing.T) {
	s := YouthProtectionSuggestions()
	want := []int{6, 8, 10, 12, 14, 16, 18}
	if len(s) != len(want) {
		t.Fatalf("want %d suggestions, got %d", len(want), len(s))
	}
	for i, sug := range s {
		if sug.MinAge != want[i] {
			t.Errorf("suggestion %d: want age %d, got %d", i, want[i], sug.MinAge)
		}
		if n := len([]rune(sug.Text)); n == 0 || n > MaxPromptConstraintLength {
			t.Errorf("suggestion %d: length %d not in 1..%d", i, n, MaxPromptConstraintLength)
		}
	}
}

// A consent covers the current declaration; none, a withdrawal or a consent
// to another version leaves it pending.
func TestConsentStatusOf(t *testing.T) {
	current := CurrentDeclaration()
	at := time.Now()
	cases := map[string]struct {
		latest  *LogEntry
		pending bool
	}{
		"none":            {nil, true},
		"current consent": {&LogEntry{Version: current.Version, Hash: current.Hash, At: at}, false},
		"older version":   {&LogEntry{Version: "0.9", Hash: "old", At: at}, true},
		"changed text":    {&LogEntry{Version: current.Version, Hash: "other", At: at}, true},
		"withdrawn":       {&LogEntry{Withdrawal: true, Version: current.Version, Hash: current.Hash, At: at}, true},
	}
	for name, c := range cases {
		status := ConsentStatusOf(c.latest)
		if status.Pending != c.pending {
			t.Errorf("%s: pending %v, want %v", name, status.Pending, c.pending)
		}
	}
	if s := ConsentStatusOf(&LogEntry{Withdrawal: true, At: at}); s.WithdrawnAt == nil || s.ConsentedVersion != nil {
		t.Errorf("a withdrawal shows when, and no standing consent: %+v", s)
	}
}
