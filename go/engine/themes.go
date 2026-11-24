package engine

import "slices"

// ThemeInfo is one look the playback-engine can wear, as the game editor lists it.
type ThemeInfo struct {
	ID string `json:"id"`
}

// themeIDs are the playback-engine's presets, in web/src/ui/theme/presets; a
// test keeps the two lists equal.
var themeIDs = []string{
	"default", "minimal", "horror", "adventure", "detective", "noir", "steampunk", "zombie",
	"barbie", "retro", "western", "desert", "school", "candy", "superhero", "sunshine",
	"storybook", "educational", "scifi", "cyberpunk", "medieval", "mystery", "mystic", "space",
	"terminal", "hacker", "playful", "nature", "ocean", "underwater", "pirate", "fire", "tech",
	"greenFantasy", "abstract", "romance", "glitch", "snowy", "fairy", "jungle", "garden",
	"circus", "nautical", "matrix",
}

// Themes is every theme a spec may name, the default first.
func Themes() []ThemeInfo {
	out := make([]ThemeInfo, 0, len(themeIDs))
	for _, id := range themeIDs {
		out = append(out, ThemeInfo{ID: id})
	}
	return out
}

func knownTheme(id string) bool { return slices.Contains(themeIDs, id) }

// GenreInfo is one genre as the game editor lists it, with the toggles it
// cannot run without.
type GenreInfo struct {
	ID    Genre    `json:"id"`
	Needs []Toggle `json:"needs"`
	// Fields is the game's fields the genre reads, for an editor to offer.
	Fields []GameField `json:"fields"`
}

// GenreList is every genre the engine runs, with its needs and the game's
// fields it reads.
func GenreList() []GenreInfo {
	var out []GenreInfo
	for _, g := range Genres() {
		needs := g.Needs()
		if needs == nil {
			needs = []Toggle{}
		}
		out = append(out, GenreInfo{ID: g, Needs: needs, Fields: g.Fields()})
	}
	return out
}
