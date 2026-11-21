// package: api / schemaf endpoints
// type:    logic
// job:     lists the UI languages and serves each one's translation file
// limits:  the languages and their translations live in lang
package api

import (
	"context"

	"cgl/lang"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// ListLanguagesEndpoint lists the languages the application is available in.
type ListLanguagesEndpoint struct{}

func (ListLanguagesEndpoint) Method() string             { return "GET" }
func (ListLanguagesEndpoint) Path() string               { return "/api/languages" }
func (ListLanguagesEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (ListLanguagesEndpoint) Handle(_ context.Context, _ ListLanguagesReq) (LanguagesResp, error) {
	all := lang.GetAllLanguages()
	languages := make(LanguagesResp, len(all))
	for i, l := range all {
		languages[i] = Language{Label: l.Label, ISO: l.Code}
	}
	return languages, nil
}

// ListLanguagesReq takes no parameters.
type ListLanguagesReq struct{}

// LanguagesResp is a list of UI languages.
type LanguagesResp = []Language

// Language is a UI language by its display label and ISO code.
type Language struct {
	Label string `json:"label"`
	ISO   string `json:"iso"`
}

// LocaleReq names a language by its code.
type LocaleReq struct {
	Code string `path:"code"`
}

// LocaleResp is a language's translations by their keys.
type LocaleResp = map[string]any

// GetLocaleFileEndpoint returns the translations of a language, by its code
// (e.g. "fr", "de", "no"), as a JSON object.
type GetLocaleFileEndpoint struct{}

func (GetLocaleFileEndpoint) Method() string             { return "GET" }
func (GetLocaleFileEndpoint) Path() string               { return "/api/languages/{code}" }
func (GetLocaleFileEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetLocaleFileEndpoint) Handle(_ context.Context, req LocaleReq) (LocaleResp, error) {
	content, err := lang.GetLocaleContent(req.Code)
	if err != nil {
		return nil, errNotFound("locale file not found for language: " + req.Code)
	}
	return content, nil
}
