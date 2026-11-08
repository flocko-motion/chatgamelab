// package: main / entrypoint
// type:    entrypoint
// job:     boots the cgl binary as a schemaf app: database, API endpoints and the embedded frontend
// limits:  holds no business logic (-> api, db)
package main

import (
	"context"
	"log"

	"github.com/flocko-motion/schemaf/schemaf"

	"cgl/api"
	"cgl/db"
)

func main() {
	app := schemaf.New(context.Background())

	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.SetFrontend(FrontendFS())

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
