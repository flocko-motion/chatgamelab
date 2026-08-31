// package: cmd / CLI entrypoint and command tree
// type:    wiring
// job:     define the root "cgl" cobra command, load .env, and register all top-level subcommands
// limits:  wiring only; each subcommand package implements its own behaviour
package cmd

import (
	"os"

	"cgl/cmd/ai"
	"cgl/cmd/apikey"
	"cgl/cmd/game"
	"cgl/cmd/healthcheck"
	"cgl/cmd/institution"
	"cgl/cmd/invite"
	"cgl/cmd/lang"
	"cgl/cmd/server"
	"cgl/cmd/user"
	"cgl/cmd/workshop"

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

	rootCmd.AddCommand(ai.Cmd)
	rootCmd.AddCommand(apikey.Cmd)
	rootCmd.AddCommand(healthcheck.Cmd)
	rootCmd.AddCommand(server.Cmd)
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(game.Cmd)
	rootCmd.AddCommand(lang.Cmd)
	rootCmd.AddCommand(institution.Cmd)
	rootCmd.AddCommand(workshop.Cmd)
	rootCmd.AddCommand(invite.Cmd)
}

// Execute runs the root command and exits with code 1 if it returns an error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
