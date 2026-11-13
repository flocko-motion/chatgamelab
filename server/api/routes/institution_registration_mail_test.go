package routes

import (
	"strings"
	"testing"

	"cgl/legal"
)

func TestPlainDeclarationTextKeepsWording(t *testing.T) {
	d := legal.CurrentDeclaration()
	plain := plainDeclarationText(d.Text)
	if strings.Contains(plain, "**") {
		t.Fatal("bold markers left in mail text")
	}
	if !strings.HasPrefix(plain, "Einverständniserklärung der Einrichtung") {
		t.Fatalf("unexpected start: %q", plain[:50])
	}
	if strings.ReplaceAll(d.Text, "**", "") != plain {
		t.Fatal("wording changed beyond the bold markers")
	}
}
