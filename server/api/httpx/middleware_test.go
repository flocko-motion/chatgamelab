package httpx

import "testing"

func TestIsImmutableMediaPath(t *testing.T) {
	cases := map[string]bool{
		"/api/messages/2b1c/image":        true,
		"/api/messages/2b1c/audio":        true,
		"/api/messages/2b1c/image/status": false,
		"/api/messages/2b1c/status":       false,
		"/api/messages/2b1c/stream":       false,
		"/api/messages/2b1c":              false,
		"/api/messages/":                  false,
		"/api/sessions/2b1c":              false,
		"/image":                          false,
		"":                                false,
	}
	for path, want := range cases {
		if got := isImmutableMediaPath(path); got != want {
			t.Errorf("isImmutableMediaPath(%q) = %v, want %v", path, got, want)
		}
	}
}
