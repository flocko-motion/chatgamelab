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
	"storybook", "educational", "spielfieber", "scifi", "cyberpunk", "medieval", "mystery",
	"mystic", "space", "terminal", "hacker", "playful", "nature", "ocean", "underwater", "pirate",
	"fire", "tech", "greenFantasy", "abstract", "romance", "glitch", "snowy", "fairy", "jungle",
	"garden", "circus", "nautical", "matrix",
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

// GenreInfo is one genre as the game editor lists it: the toggles it cannot
// run without at all, the game's fields it reads, and its halves.
type GenreInfo struct {
	ID    Genre    `json:"id"`
	Needs []Toggle `json:"needs"`
	// Fields is the game's fields the genre reads, the same for both
	// halves, for an editor to offer.
	Fields []GameField `json:"fields"`
	// VoiceSlots are the voices the genre speaks in, both halves', and what
	// a game may choose for each, for an editor to offer.
	VoiceSlots []VoiceSlot `json:"voiceSlots"`
	// Subgenres are its halves, live first.
	Subgenres []SubgenreInfo `json:"subgenres"`
}

// SubgenreInfo is one half of a genre: its mode, the toggles it needs, the
// voices it speaks in, and its own prompts, for a launch that adapts them.
type SubgenreInfo struct {
	ID         Subgenre      `json:"id"`
	Mode       Mode          `json:"mode"`
	Needs      []Toggle      `json:"needs"`
	VoiceSlots []VoiceSlot   `json:"voiceSlots"`
	Prompts    []GenrePrompt `json:"prompts"`
}

// GenreList is every genre a game can be, with its needs, the game's fields
// it reads and its halves.
func GenreList() []GenreInfo {
	var out []GenreInfo
	for _, g := range Genres() {
		info := GenreInfo{ID: g, Needs: orNone(g.Needs()), Fields: g.Fields(), VoiceSlots: g.VoiceSlots(), Subgenres: []SubgenreInfo{}}
		for _, s := range g.Subgenres() {
			info.Subgenres = append(info.Subgenres, SubgenreInfo{
				ID: s, Mode: s.Mode(), Needs: orNone(s.Needs()), VoiceSlots: s.VoiceSlots(), Prompts: s.Prompts(),
			})
		}
		out = append(out, info)
	}
	return out
}

// orNone lists no toggles as an empty list, never as null.
func orNone(toggles []Toggle) []Toggle {
	if toggles == nil {
		return []Toggle{}
	}
	return toggles
}
