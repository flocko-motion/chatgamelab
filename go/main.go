// package: main / entrypoint
// type:    entrypoint
// job:     boots the cgl binary as a schemaf app wired by app.Wire
// limits:  holds no wiring and no business logic (-> app)
package main

import (
	"context"
	"log"

	"github.com/flocko-motion/schemaf/schemaf"

	"cgl/app"
)

func main() {
	a := schemaf.New(context.Background())
	app.Wire(a)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
