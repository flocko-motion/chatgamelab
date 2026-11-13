package game

import "testing"

func TestCleanNarration(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain prose stays",
			in:   "  Du stehst im Festsaal. Was tust du?  ",
			want: "Du stehst im Festsaal. Was tust du?",
		},
		{
			name: "prose with braces stays",
			in:   "Auf dem Schild steht {geheim}. Was tust du?",
			want: "Auf dem Schild steht {geheim}. Was tust du?",
		},
		{
			name: "whole answer as JSON with narration field",
			in:   `{"narration":"Du bist Ehrengast. Was tust du zuerst?","status":{"Health":100,"CluesFound":0}}`,
			want: "Du bist Ehrengast. Was tust du zuerst?",
		},
		{
			name: "whole answer as JSON with scene field",
			in:   `{"scene":"Du stehst im Ballsaal.","status":{"ruf":80,"inventar":["Notizblock"]}}`,
			want: "Du stehst im Ballsaal.",
		},
		{
			name: "narration preferred over summary",
			in:   "{\n  \"narration\": \"Lange Erzählung.\",\n  \"summary\": \"Kurz.\",\n  \"status\": {\"Stress\": 8}\n}",
			want: "Lange Erzählung.",
		},
		{
			name: "unknown field falls back to longest string",
			in:   `{"foo":"kurz","bar":"Das ist der eigentliche Text."}`,
			want: "Das ist der eigentliche Text.",
		},
		{
			name: "status block appended to prose",
			in:   "Du bist Ehrengast. Womit beginnst du?\n\n{\"health\":100,\"inventory\":[\"Notizbuch\",\"Stift\"],\"score\":64}",
			want: "Du bist Ehrengast. Womit beginnst du?",
		},
		{
			name: "fenced JSON appended to prose",
			in:   "Du bist Ehrengast. Womit beginnst du?\n```json\n{\"health\":100}\n```",
			want: "Du bist Ehrengast. Womit beginnst du?",
		},
		{
			name: "fenced whole JSON",
			in:   "```json\n{\"narration\":\"Du bist Ehrengast.\"}\n```",
			want: "Du bist Ehrengast.",
		},
		{
			name: "bold and italic stars removed",
			in:   "Du siehst **den goldenen Pokal** und *flüsterst* leise. ***Achtung!***",
			want: "Du siehst den goldenen Pokal und flüsterst leise. Achtung!",
		},
		{
			name: "stars inside JSON narration removed",
			in:   `{"narration":"Der **Panther** springt."}`,
			want: "Der Panther springt.",
		},
		{
			name: "lone or spaced stars stay",
			in:   "Rechne 5 * 3 aus. Ein Stern* bleibt. ** nicht fett **",
			want: "Rechne 5 * 3 aus. Ein Stern* bleibt. ** nicht fett **",
		},
		{
			name: "no pairing across lines",
			in:   "Erste Zeile *mit Stern\nzweite Zeile* hier",
			want: "Erste Zeile *mit Stern\nzweite Zeile* hier",
		},
		{
			name: "broken JSON stays as is",
			in:   `{"narration":"abgebrochen`,
			want: `{"narration":"abgebrochen`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanNarration(tt.in); got != tt.want {
				t.Errorf("cleanNarration() = %q, want %q", got, tt.want)
			}
		})
	}
}
