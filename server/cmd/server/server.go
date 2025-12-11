package server

import (
	"cgl/api"
	"log"
	"os"
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
	portStr := os.Getenv("API_PORT")
	if portStr == "" {
		portStr = "3000"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid API_PORT '%s': %v", portStr, err)
	}

	api.RunServer(cmd.Context(), port, devMode)
}
