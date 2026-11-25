package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No text names an operator: the software is neutral, and texts name the
// instance's operator through {{operatorName}}.
func TestNoTextNamesAnOperator(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("locales", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no locale files: %v", err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"JFF", "Jugend Film Fernsehen", "Aktion Jugendschutz", "chatgamelab.eu"} {
			if strings.Contains(string(raw), name) {
				t.Errorf("%s names %q", f, name)
			}
		}
	}
}
