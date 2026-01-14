package routes

import (
	"net/http"

	"cgl/api/httpx"
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
	languages := []Language{
		{Label: "English", ISO: "en"},
		{Label: "Deutsch", ISO: "de"},
		{Label: "Español", ISO: "es"},
		{Label: "Français", ISO: "fr"},
		{Label: "Italiano", ISO: "it"},
		{Label: "Português", ISO: "pt"},
		{Label: "Nederlands", ISO: "nl"},
		{Label: "Polski", ISO: "pl"},
		{Label: "日本語", ISO: "ja"},
		{Label: "中文", ISO: "zh"},
	}

	httpx.WriteJSON(w, http.StatusOK, languages)
}
