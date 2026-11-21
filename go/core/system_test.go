package core

import "testing"

// ADMIN_EMAILS lists addresses separated by commas, spaces around them
// ignored; nothing else counts.
func TestIsAdminEmail(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", " a@example.org , b@example.org,,")
	for email, want := range map[string]bool{
		"a@example.org": true, "b@example.org": true, " b@example.org ": true,
		"c@example.org": false, "": false, "a@example.or": false,
	} {
		if got := isAdminEmail(email); got != want {
			t.Errorf("isAdminEmail(%q) = %v, want %v", email, got, want)
		}
	}
	t.Setenv("ADMIN_EMAILS", "")
	if isAdminEmail("a@example.org") {
		t.Error("an empty list names an admin")
	}
}
