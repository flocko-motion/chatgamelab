// package: routes / the engine's HTTP handlers, mounted here
// type:    wiring
// job:     holds the engine's API and documents its handlers for the OpenAPI spec
// limits:  no handler logic; the engine owns the handlers and their JSON (-> engine/api)
package routes

import engineapi "engine/api"

// Engine is the engine's HTTP transport; router.go mounts its handlers.
var Engine = engineapi.New()

// enginePlatformsDoc documents Engine.Platforms for the OpenAPI spec; swag reads
// annotations only from this module's functions.
//
//	@Summary		List AI platforms
//	@Description	Returns the engine's AI platforms with their toggles, playable genres and tiers
//	@Tags			platforms
//	@Produce		json
//	@Success		200	{array}	engine.PlatformInfo
//	@Router			/platforms [get]
func enginePlatformsDoc() {}
