// package: routes / AI platform HTTP handlers
// type:    logic
// job:     handles listing available AI platforms and their metadata
// limits:  no route registration (-> router.go); platform data sourced from game/ai
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/game/ai"
)

// GetPlatforms godoc
//
//	@Summary		List AI platforms
//	@Description	Returns all available AI platforms with their metadata
//	@Tags			platforms
//	@Produce		json
//	@Success		200	{array}		obj.AiPlatform
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/platforms [get]
func GetPlatforms(w http.ResponseWriter, r *http.Request) {
	platforms := ai.GetAiPlatformInfos()
	httpx.WriteJSON(w, http.StatusOK, platforms)
}
