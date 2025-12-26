package main

import (
	"cgl/api/routes"
	"cgl/cmd"
)

// Set via -ldflags at build time
var (
	GitCommit = "dev"
	BuildTime = "unknown"
)

func main() {
	// Pass version info to routes package
	routes.GitCommit = GitCommit
	routes.BuildTime = BuildTime

	cmd.Execute()
}
