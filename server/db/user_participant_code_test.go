package db

import "testing"

func TestParticipantTokenWords(t *testing.T) {
	cases := map[string]string{
		"participant-artikel-drama-ehemann-affe": "artikel-drama-ehemann-affe",
		"participant-xK9mQvR8nL-pZ2wT4yB":        "",
		"participant-":                           "",
		"artikel-drama-ehemann-affe":             "",
		"":                                       "",
	}
	for token, want := range cases {
		if got := ParticipantTokenWords(token); got != want {
			t.Errorf("ParticipantTokenWords(%q) = %q, want %q", token, got, want)
		}
	}
}
