// package: app / wiring
// type:    wiring
// job:     registers the cgl app's providers, hooks and services with schemaf, for the binary and the tests alike
// limits:  holds no business logic (-> api, db)
package app

import (
	"github.com/flocko-motion/schemaf/schemaf"

	"cgl/api"
	"cgl/api/auth"
	"cgl/db"
)

// Wire registers the app's database, API endpoints, sign-in, background
// services and the embedded frontend.
func Wire(app *schemaf.App) {
	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.OnLogin(auth.OnLogin)
	app.AddService(api.Startup)
	app.AddService(api.Janitor)
	app.SetFrontend(FrontendFS())
}
