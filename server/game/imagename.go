// package: game / gameplay orchestration
// type:    logic
// job:     builds a human-readable download filename for a generated scene image.
// limits:  pure string formatting; no IO.
package game

import (
	"fmt"
	"strings"
)

// maxPromptSlugLen caps the "what happens in the image" part of the filename.
const maxPromptSlugLen = 40

// gameNamePrefixLen is how many characters of the (slugified) game name go into
// the filename, per the agreed scheme "first 10 characters of the game name".
const gameNamePrefixLen = 10

var transliterations = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"Ä", "ae", "Ö", "oe", "Ü", "ue",
	"á", "a", "à", "a", "â", "a", "ã", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o",
	"ú", "u", "ù", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

// slugify lowercases, transliterates common accented letters, and reduces every
// other run of non [a-z0-9] characters to a single hyphen.
func slugify(s string) string {
	s = transliterations.Replace(strings.ToLower(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// truncateSlug shortens a slug to at most max characters, preferring to cut at a
// hyphen so words stay intact.
func truncateSlug(slug string, max int) string {
	if len(slug) <= max {
		return slug
	}
	cut := slug[:max]
	if i := strings.LastIndexByte(cut, '-'); i > 0 {
		cut = cut[:i]
	}
	return strings.Trim(cut, "-")
}

// BuildImageDownloadName builds the filename offered when a player saves a scene
// image: "<first 10 chars of game name>_<NN>_<what happens>.png", e.g.
// "der-verzau_03_ein-drache-kreist-ueber-dem-see.png".
func BuildImageDownloadName(gameName string, imageIndex int, imagePrompt string) string {
	namePart := slugify(gameName)
	if len(namePart) > gameNamePrefixLen {
		namePart = strings.Trim(namePart[:gameNamePrefixLen], "-")
	}
	if namePart == "" {
		namePart = "bild"
	}

	if imageIndex < 1 {
		imageIndex = 1
	}
	numPart := fmt.Sprintf("%02d", imageIndex)

	promptPart := truncateSlug(slugify(imagePrompt), maxPromptSlugLen)
	if promptPart == "" {
		promptPart = "szene"
	}

	return fmt.Sprintf("%s_%s_%s.png", namePart, numPart, promptPart)
}
