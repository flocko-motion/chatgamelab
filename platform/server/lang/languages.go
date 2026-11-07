// package: lang / supported language registry
// type:    data
// job:     the supported language codes and their native names, read from the embedded languages.json
// limits:  the list itself is kept by the translation tool in /translation; locale content lives in locales.go
package lang

import "encoding/json"

// supportedLanguages maps each language code to its native name.
var supportedLanguages = func() map[string]string {
	data, err := localesFS.ReadFile("locales/languages.json")
	if err != nil {
		panic("lang: embedded languages.json missing: " + err.Error())
	}
	var list []Language
	if err := json.Unmarshal(data, &list); err != nil {
		panic("lang: embedded languages.json invalid: " + err.Error())
	}
	m := make(map[string]string, len(list))
	for _, l := range list {
		m[l.Code] = l.Label
	}
	return m
}()

const TranslateInstruction = "You are an expert in translation of json structured language files for games. Translate the given JSON object to the target language while preserving the exact same structure and keys. Only translate the string values. Return a valid JSON object. You get the original already in two languages, so that you have more context to understand the intention of each field."

// GetLanguageName returns the full name of a language from its code
func GetLanguageName(code string) string {
	if name, exists := supportedLanguages[code]; exists {
		return name
	}
	return code
}

// IsValidLanguageCode checks if a language code is supported
func IsValidLanguageCode(code string) bool {
	_, exists := supportedLanguages[code]
	return exists
}
