package api

import "testing"

func TestSessionInPath(t *testing.T) {
	cases := map[string]string{
		"/sessions/abc/stream":        "abc",
		"/sessions/abc":               "abc",
		"/sessions":                   "",
		"/sessions/":                  "",
		"/playback-engine/index.html": "",
		"/genres":                     "",
	}
	for path, want := range cases {
		got, ok := sessionInPath(path)
		if got != want || ok != (want != "") {
			t.Errorf("sessionInPath(%q) = %q, %v; want %q", path, got, ok, want)
		}
	}
}
