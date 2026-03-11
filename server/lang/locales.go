package lang

import (
	"embed"
	"encoding/json"
	"fmt"
)

// Embed all locale JSON files at compile time from the symlinked directory
//
//go:embed locales/*.json
var localesFS embed.FS

// Language represents a language with its code and name
type Language struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// GetAllLanguages returns all supported languages with their codes and labels
// This is the source of truth for available languages in the system
func GetAllLanguages() []Language {
	languages := make([]Language, 0, len(supportedLanguages))

	// Add all other supported languages from supportedLanguages map
	for code, label := range supportedLanguages {
		languages = append(languages, Language{Code: code, Label: label})
	}

	return languages
}

// GetLocaleContent returns the JSON content for a specific language code
// Returns the embedded locale file content or error if not found
func GetLocaleContent(langCode string) (map[string]interface{}, error) {
	filename := fmt.Sprintf("locales/%s.json", langCode)

	data, err := localesFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("locale file not found for language: %s", langCode)
	}

	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse locale file for %s: %w", langCode, err)
	}

	return content, nil
}

