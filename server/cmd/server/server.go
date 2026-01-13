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

	// Dev mode enabled via DEV_MODE env variable
	devMode := os.Getenv("DEV_MODE") == "true"

	log.Debug("starting server", "port", port, "dev_mode", devMode)
	api.RunServer(cmd.Context(), port, devMode)
}
