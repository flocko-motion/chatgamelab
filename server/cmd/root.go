package cmd

import (
	"os"

	"cgl/cmd/server"
	"cgl/cmd/user"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cgl",
	Short: "Chat Game Lab server",
	Long:  "CGL (Chat Game Lab) - A server for interactive chat-based games.",
}

func init() {
	// Auto-load .env from current dir or parent dir
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	rootCmd.AddCommand(server.Cmd)
	rootCmd.AddCommand(user.Cmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
