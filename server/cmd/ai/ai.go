// package: ai / cobra AI platform command group
// type:    cli
// job:     defines the parent "ai" cobra command grouping AI platform and model subcommands.
// limits:  holds no subcommand logic; subcommands register themselves (-> ai/models.go).
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
