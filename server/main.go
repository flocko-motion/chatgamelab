package main

import (
	"cgl/api/routes"
	"cgl/cmd"

	"github.com/joho/godotenv"
)

// Set via -ldflags at build time
var (
	GitCommit = "dev"
	Version   = "dev"
	BuildTime = "unknown"
)

func init() {
	// Load .env from root directory
	// Silently ignore if not found (in prod .env should not be used)
	_ = godotenv.Load("../.env")
}

func main() {
	// Pass version info to routes package
	routes.GitCommit = GitCommit
	routes.Version = Version
	routes.BuildTime = BuildTime

	cmd.Execute()
}
