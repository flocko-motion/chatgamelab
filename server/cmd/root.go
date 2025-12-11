package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cgl",
	Short: "Chat Game Lab server",
	Long:  "CGL (Chat Game Lab) - A server for interactive chat-based games.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
