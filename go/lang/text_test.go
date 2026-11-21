package lang

import "testing"

// A text comes in the language asked for, with its placeholders filled; a
// language without it falls back to English, and an unknown key shows as
// itself.
func TestText(t *testing.T) {
	vars := map[string]string{"name": "Ada", "organisation": "JZ Nord"}
	if got := Text("de", "mail.consent.greeting", vars); got != "Hallo Ada," {
		t.Errorf("de: %q", got)
	}
	if got := Text("en", "mail.consent.greeting", vars); got != "Hello Ada," {
		t.Errorf("en: %q", got)
	}
	if got := Text("xx", "mail.consent.greeting", vars); got != "Hello Ada," {
		t.Errorf("unknown language: %q", got)
	}
	if got := Text("de", "mail.no.such.key", nil); got != "mail.no.such.key" {
		t.Errorf("unknown key: %q", got)
	}
}
