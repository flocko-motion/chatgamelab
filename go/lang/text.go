// package: lang / server-side texts
// type:    logic
// job:     a text from the embedded locales, by key, in a language, with its {{placeholders}} filled: for what the server writes itself, such as mails
// limits:  read-only; the texts are written in frontend/src/i18n/locales and mirrored here by the translation tool
package lang

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// fallbackLanguage is used for a language or a key the locales lack.
const fallbackLanguage = "en"

var (
	localesMu sync.Mutex
	loaded    = map[string]map[string]any{}
)

// Text is the text at key ("namespace.section.field") in language, with each
// {{name}} replaced by vars[name]. A language without the key falls back to
// English; a key no locale has is returned as it is, so a missing text shows
// up rather than vanishing.
func Text(language, key string, vars map[string]string) string {
	text, ok := lookup(language, key)
	if !ok {
		text, ok = lookup(fallbackLanguage, key)
	}
	if !ok {
		return key
	}
	for name, value := range vars {
		text = strings.ReplaceAll(text, "{{"+name+"}}", value)
	}
	return text
}

// lookup is the string at key in language's locale.
func lookup(language, key string) (string, bool) {
	var node any = locale(language)
	for _, part := range strings.Split(key, ".") {
		m, ok := node.(map[string]any)
		if !ok {
			return "", false
		}
		node = m[part]
	}
	text, ok := node.(string)
	return text, ok
}

// locale is language's embedded locale, parsed once; empty for an unknown
// language.
func locale(language string) map[string]any {
	localesMu.Lock()
	defer localesMu.Unlock()
	if m, ok := loaded[language]; ok {
		return m
	}
	m := map[string]any{}
	if data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.json", language)); err == nil {
		if err := json.Unmarshal(data, &m); err != nil {
			panic("lang: embedded locale " + language + " invalid: " + err.Error())
		}
	}
	loaded[language] = m
	return m
}
