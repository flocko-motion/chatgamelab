package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/lang"
)

type Language struct {
	Label string `json:"label"`
	ISO   string `json:"iso"`
}

// GetLanguages godoc
//
//	@Summary		Get available languages
//	@Description	Returns a list of available languages for the application
//	@Tags			languages
//	@Produce		json
//	@Success		200	{array}		Language
//	@Router			/languages [get]
func GetLanguages(w http.ResponseWriter, r *http.Request) {
	// Get languages from lang module (source of truth)
	langList := lang.GetAllLanguages()

	// Convert to API response format
	languages := make([]Language, len(langList))
	for i, l := range langList {
		languages[i] = Language{
			Label: l.Label,
			ISO:   l.Code,
		}
	}

	httpx.WriteJSON(w, http.StatusOK, languages)
}

// GetLocaleFile godoc
//
//	@Summary		Get locale file for a specific language
//	@Description	Returns the translation JSON file for the specified language code
//	@Tags			languages
//	@Produce		json
//	@Param			code	path		string	true	"Language code (e.g., 'fr', 'de', 'no')"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]string
//	@Router			/languages/{code} [get]
func GetLocaleFile(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if code == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "language code is required",
		})
		return
	}

	// Get locale content from lang module
	content, err := lang.GetLocaleContent(code)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{
			"error": "locale file not found for language: " + code,
		})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, content)
}
