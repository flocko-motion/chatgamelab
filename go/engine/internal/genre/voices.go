package genre

import (
	"slices"
	"strings"

	"engine/internal/adapters"
)

// VoiceChoice is what a game chooses for one of a genre's voices: a built-in
// voice of a gender, any built-in voice the cast finds fitting, or a voice
// made for the game where the genre makes one.
type VoiceChoice string

const (
	VoiceMale   VoiceChoice = "male"
	VoiceFemale VoiceChoice = "female"
	VoiceAny    VoiceChoice = "any"
	VoiceCustom VoiceChoice = "custom"
)

// The voices the genres speak in, by the name a game chooses for them.
const (
	VoiceNarrator  = "narrator"
	VoiceCharacter = "character"
	VoiceA         = "a"
	VoiceB         = "b"
)

// choiceOf is the game's choice for one of its voices; a voice it chose
// nothing for is any.
func choiceOf(choices map[string]VoiceChoice, voice string) VoiceChoice {
	if c, ok := choices[voice]; ok && c != "" {
		return c
	}
	return VoiceAny
}

// presentation is the gender a choice asks for, as a voice catalogue states
// it; empty asks for none.
func presentation(choice VoiceChoice) string {
	switch choice {
	case VoiceMale:
		return "masculine"
	case VoiceFemale:
		return "feminine"
	}
	return ""
}

// castable are the built-in voices a choice may be cast in: those of its
// gender, or all, the ones that speak the game's language first.
func castable(voices []adapters.BuiltinVoice, choice VoiceChoice, language string) []adapters.BuiltinVoice {
	gender := presentation(choice)
	base, _, _ := strings.Cut(language, "-")
	var own, other []adapters.BuiltinVoice
	for _, v := range voices {
		switch {
		case gender != "" && v.Gender != gender:
		case base != "" && v.Language == base:
			own = append(own, v)
		default:
			other = append(other, v)
		}
	}
	return append(own, other...)
}

// fallbackVoice is the voice a choice speaks in when no cast comes: the
// first it may be cast in that no other voice of the game took; "" when
// there is none.
func fallbackVoice(voices []adapters.BuiltinVoice, choice VoiceChoice, language string, taken ...string) string {
	candidates := castable(voices, choice, language)
	for _, v := range candidates {
		if !slices.Contains(taken, v.Name) {
			return v.Name
		}
	}
	if len(candidates) > 0 {
		return candidates[0].Name
	}
	return ""
}

// voiceNames are the voices' names, in their order.
func voiceNames(voices []adapters.BuiltinVoice) []string {
	names := make([]string, 0, len(voices))
	for _, v := range voices {
		names = append(names, v.Name)
	}
	return names
}

// voiceEnum are the voices' names as a schema's enum.
func voiceEnum(voices []adapters.BuiltinVoice) []any {
	enum := make([]any, 0, len(voices))
	for _, name := range voiceNames(voices) {
		enum = append(enum, name)
	}
	return enum
}

// voiceList lists the built-in voices for a prompt, each with the language
// it speaks natively, its accent and, where stated, its gender.
func voiceList(voices []adapters.BuiltinVoice) string {
	var b strings.Builder
	for _, v := range voices {
		line := "- " + v.Name + ": speaks " + languageName(v.Language)
		if v.Accent != "" {
			line += ", " + v.Accent
		}
		if v.Gender != "" {
			line += ", " + v.Gender
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
