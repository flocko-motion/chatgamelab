// package: legal / versioned legal texts
// type:    data
// job:     embeds the institution declaration (Einverständniserklärung) and the youth-protection constraint suggestions so UI, mail and consent log all read one source.
// limits:  no HTTP (-> api/routes/legal.go), no persistence (-> db).
package legal

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed institution_declaration/*.md youth_protection_suggestions.json
var files embed.FS

// BindingLanguage is the only language a declaration is agreed to in.
// Other language files are reading aids; consent and hash always refer to the German text.
const BindingLanguage = "de"

// declarationVersions lists every published version. The last entry is current.
// To publish a new version: add institution_declaration/v<Version>.de.md and append here.
var declarationVersions = []struct {
	Version string
	Date    string // YYYY-MM-DD, date of the legal text
}{
	{Version: "1.0", Date: "2026-11-12"},
}

// Declaration is one version of the institution declaration.
type Declaration struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	// Text is the binding German text (Markdown).
	Text string `json:"text"`
	// Hash is the hex SHA-256 of Text, stored with every consent record.
	Hash string `json:"hash"`
	// Translations holds optional, non-binding reading aids keyed by language code.
	Translations map[string]string `json:"translations,omitempty"`
}

// YouthProtectionSuggestion is one suggested Jugendschutz-Prompt-Einschränkung.
type YouthProtectionSuggestion struct {
	MinAge int    `json:"minAge"`
	Text   string `json:"text"`
}

// CurrentDeclaration returns the current version of the declaration.
func CurrentDeclaration() Declaration {
	return mustDeclaration(declarationVersions[len(declarationVersions)-1].Version)
}

// DeclarationByVersion returns a published version, or false if unknown.
func DeclarationByVersion(version string) (Declaration, bool) {
	for _, v := range declarationVersions {
		if v.Version == version {
			return mustDeclaration(version), true
		}
	}
	return Declaration{}, false
}

func mustDeclaration(version string) Declaration {
	var date string
	for _, v := range declarationVersions {
		if v.Version == version {
			date = v.Date
		}
	}
	raw, err := files.ReadFile(fmt.Sprintf("institution_declaration/v%s.%s.md", version, BindingLanguage))
	if err != nil {
		panic(fmt.Sprintf("legal: declaration v%s missing: %v", version, err))
	}
	text := strings.TrimSpace(string(raw))
	sum := sha256.Sum256([]byte(text))

	d := Declaration{Version: version, Date: date, Text: text, Hash: hex.EncodeToString(sum[:])}

	entries, _ := files.ReadDir("institution_declaration")
	prefix := "v" + version + "."
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".md") {
			continue
		}
		lang := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".md")
		if lang == BindingLanguage {
			continue
		}
		if b, err := files.ReadFile("institution_declaration/" + name); err == nil {
			if d.Translations == nil {
				d.Translations = map[string]string{}
			}
			d.Translations[lang] = strings.TrimSpace(string(b))
		}
	}
	return d
}

// YouthProtectionSuggestions returns the suggestions, youngest age first.
func YouthProtectionSuggestions() []YouthProtectionSuggestion {
	raw, err := files.ReadFile("youth_protection_suggestions.json")
	if err != nil {
		panic("legal: youth_protection_suggestions.json missing: " + err.Error())
	}
	var parsed struct {
		Suggestions []YouthProtectionSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		panic("legal: youth_protection_suggestions.json invalid: " + err.Error())
	}
	return parsed.Suggestions
}
