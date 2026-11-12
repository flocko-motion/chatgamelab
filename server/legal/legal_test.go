package legal

import (
	"strings"
	"testing"
)

func TestCurrentDeclaration(t *testing.T) {
	d := CurrentDeclaration()
	if d.Version != "1.0" || d.Date != "2026-11-12" {
		t.Fatalf("unexpected version/date: %s %s", d.Version, d.Date)
	}
	if !strings.HasPrefix(d.Text, "**Einverständniserklärung der Einrichtung") {
		t.Fatalf("unexpected text start: %q", d.Text[:60])
	}
	if !strings.Contains(d.Text, "10. Ich nehme zustimmend zur Kenntnis") {
		t.Fatal("text is missing point 10")
	}
	if len(d.Hash) != 64 {
		t.Fatalf("hash should be hex sha256, got %q", d.Hash)
	}
	if again := CurrentDeclaration(); again.Hash != d.Hash {
		t.Fatal("hash is not stable")
	}
	if _, ok := DeclarationByVersion("0.9"); ok {
		t.Fatal("unknown version must not resolve")
	}
}

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
		// Org constraints are limited to 200 characters.
		if n := len([]rune(sug.Text)); n == 0 || n > 200 {
			t.Errorf("suggestion %d: length %d not in 1..200", i, n)
		}
	}
}
