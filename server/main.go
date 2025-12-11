package main

import (
	"cgl/endpoints"
	"cgl/cmd"
)

// Set via -ldflags at build time
var (
	GitCommit = "dev"
	BuildTime = "unknown"
)

func main() {
	// Pass version info to api package
	endpoints.GitCommit = GitCommit
	endpoints.BuildTime = BuildTime

	cmd.Execute()
}
