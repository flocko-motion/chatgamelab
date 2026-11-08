//go:build tools

// package: main / build-time tool dependencies
// type:    wiring
// job:     pins build-only tool dependencies (swag) so go modules retains them
// limits:  no runtime code; only imports under the tools build tag
package main

import (
	_ "github.com/swaggo/swag/cmd/swag"
)
