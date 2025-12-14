package server

import (
	"cgl/api"
	"cgl/functional"
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

var devMode bool

// Cmd is the server subcommand
var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Start the CGL server",
	Long:  "Start the Chat Game Lab HTTP server.",
	Run:   runServer,
}

func init() {
	Cmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")
}

func runServer(cmd *cobra.Command, args []string) {
	portStr := functional.RequireEnv("PORT_BACKEND")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid PORT_BACKEND '%s': %v", portStr, err)
	}

	api.RunServer(cmd.Context(), port, devMode)
}
