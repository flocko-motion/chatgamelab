package game

import "testing"

func TestBuildImageDownloadName(t *testing.T) {
	cases := []struct {
		name    string
		game    string
		index   int
		prompt  string
		want    string
	}{
		{
			name:   "typical german title and prompt",
			game:   "Der verzauberte Wald",
			index:  3,
			prompt: "Ein Drache kreist über dem dunklen See",
			want:   "der-verzau_03_ein-drache-kreist-ueber-dem-dunklen-see.png",
		},
		{
			name:   "index padded to two digits",
			game:   "Zauberschule",
			index:  1,
			prompt: "Tor",
			want:   "zauberschu_01_tor.png",
		},
		{
			name:   "index above 99 keeps all digits",
			game:   "Marathon",
			index:  123,
			prompt: "Ziel",
			want:   "marathon_123_ziel.png",
		},
		{
			name:   "empty prompt falls back to szene",
			game:   "Testspiel",
			index:  2,
			prompt: "   ",
			want:   "testspiel_02_szene.png",
		},
		{
			name:   "empty game name falls back to bild",
			game:   "!!!",
			index:  4,
			prompt: "Nebel",
			want:   "bild_04_nebel.png",
		},
		{
			name:   "zero/negative index clamped to 1",
			game:   "Spiel",
			index:  0,
			prompt: "Anfang",
			want:   "spiel_01_anfang.png",
		},
		{
			name:   "long prompt truncated at a word boundary",
			game:   "Weltraum",
			index:  7,
			prompt: "Eine gewaltige Raumstation treibt lautlos an einem rot leuchtenden Gasplaneten vorbei",
			want:   "weltraum_07_eine-gewaltige-raumstation-treibt.png",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildImageDownloadName(tc.game, tc.index, tc.prompt)
			if got != tc.want {
				t.Fatalf("BuildImageDownloadName(%q, %d, %q) = %q, want %q", tc.game, tc.index, tc.prompt, got, tc.want)
			}
			if len(got) == 0 {
				t.Fatal("filename must not be empty")
			}
		})
	}
}
