// package: main / entrypoint
// type:    entrypoint
// job:     boots the cgl binary as a schemaf app: database, API endpoints, sign-in, background services and the embedded frontend
// limits:  holds no business logic (-> api, db)
package main

import (
	"context"
	"log"

	"github.com/flocko-motion/schemaf/schemaf"

	"cgl/api"
	"cgl/api/auth"
	"cgl/db"
)

func main() {
	app := schemaf.New(context.Background())

	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.OnLogin(auth.OnLogin)
	app.AddService(api.Startup)
	app.AddService(api.Janitor)
	app.SetFrontend(FrontendFS())

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
