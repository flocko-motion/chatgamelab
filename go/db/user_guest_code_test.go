package db

import "testing"

func TestGuestTokenWords(t *testing.T) {
	cases := map[string]string{
		"guest-artikel-drama-ehemann-affe":       "artikel-drama-ehemann-affe",
		"participant-artikel-drama-ehemann-affe": "artikel-drama-ehemann-affe", // legacy prefix, still read
		"participant-xK9mQvR8nL-pZ2wT4yB":        "",                           // long code from before word codes
		"guest-":                                 "",
		"artikel-drama-ehemann-affe":             "",
		"":                                       "",
	}
	for token, want := range cases {
		if got := GuestTokenWords(token); got != want {
			t.Errorf("GuestTokenWords(%q) = %q, want %q", token, got, want)
		}
	}
}
