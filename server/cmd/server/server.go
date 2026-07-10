// package: server / server startup command
// type:    cli
// job:     read PORT_BACKEND and DEV_MODE from the environment and start the HTTP server
// limits:  only bootstraps and launches; request handling lives in the api package (-> api)
package server

import (
	"cgl/api"
	"cgl/functional"
	"cgl/log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// Cmd is the server subcommand
var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Start the CGL server",
	Long:  "Start the Chat Game Lab HTTP server.",
	Run:   runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	portStr := functional.RequireEnv("PORT_BACKEND")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Error("invalid PORT_BACKEND", "value", portStr, "error", err)
		os.Exit(1)
	}

	// Dev mode enabled via DEV_MODE env variable (accepts "true" or "1")
	devModeStr := os.Getenv("DEV_MODE")
	devMode := devModeStr == "true" || devModeStr == "1"

	log.Info("starting server", "port", port, "dev_mode", devMode)
	api.RunServer(cmd.Context(), port, devMode, nil)
}
