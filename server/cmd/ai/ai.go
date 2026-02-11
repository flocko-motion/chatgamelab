package ai

import (
	"github.com/spf13/cobra"
)

// Cmd is the ai subcommand
var Cmd = &cobra.Command{
	Use:   "ai",
	Short: "AI platform management commands",
	Long:  "Commands for managing AI platforms and models.",
}
