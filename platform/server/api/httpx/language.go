// package: httpx / language resolution for incoming requests
// type:    logic
// job:     determines which language a newly created user should get
// limits:  does not persist anything (-> db); does not translate (-> lang)
package httpx

import (
	"net/http"
	"strings"

	"cgl/lang"
)

// DefaultLanguage is used when a request carries no usable language hint.
const DefaultLanguage = "en"

// PreferredLanguage resolves the language to store on a newly created user.
//
// Order of preference:
//  1. explicit — the language the client actively sends (what the user sees in
//     the app right now). Empty when the endpoint has no request body.
//  2. the browser's Accept-Language header.
//  3. English.
//
// This matters because a user's stored language is what game sessions are
// locked to at launch. Without it every new account silently defaults to
// English, so a German user gets a German interface but an English game.
func PreferredLanguage(r *http.Request, explicit string) string {
	if code := normalizeLanguage(explicit); code != "" {
		return code
	}
	if code := languageFromAcceptHeader(r.Header.Get("Accept-Language")); code != "" {
		return code
	}
	return DefaultLanguage
}

// normalizeLanguage lowercases a tag, strips any region ("de-AT" -> "de") and
// returns "" unless the result is a language the app actually supports.
func normalizeLanguage(tag string) string {
	code := strings.ToLower(strings.TrimSpace(tag))
	if code == "" {
		return ""
	}
	if base, _, found := strings.Cut(code, "-"); found {
		code = base
	}
	if !lang.IsValidLanguageCode(code) {
		return ""
	}
	return code
}

// languageFromAcceptHeader returns the first supported language from an
// Accept-Language header, e.g. "de-DE,de;q=0.9,en;q=0.8" -> "de".
//
// Entries are taken in the order the browser sent them. Browsers already list
// them by descending quality, so parsing q-values would not change the result.
func languageFromAcceptHeader(header string) string {
	for entry := range strings.SplitSeq(header, ",") {
		tag, _, _ := strings.Cut(entry, ";") // drop ";q=0.9"
		if code := normalizeLanguage(tag); code != "" {
			return code
		}
	}
	return ""
}
