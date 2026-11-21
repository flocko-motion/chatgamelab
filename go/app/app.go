// package: app / wiring
// type:    wiring
// job:     registers the cgl app's providers, hooks and services with schemaf, for the binary and the tests alike
// limits:  holds no business logic (-> api, core)
package app

import (
	"github.com/flocko-motion/schemaf/schemaf"

	"cgl/api"
	"cgl/api/auth"
	"cgl/core"
	"cgl/db"
)

// Wire registers the app's database, API endpoints, sign-in, the secure
// core's caller (resolved once per request), background services and the
// embedded frontend.
func Wire(app *schemaf.App) {
	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.OnLogin(auth.OnLogin)
	app.OnRequest(core.Resolve)
	app.AddService(api.Startup)
	app.AddService(core.Bootstrap)
	app.AddService(core.Janitor(func(id string) { api.Engine.Forget(id) }))
	app.SetFrontend(FrontendFS())
}
