// package: apikey / cobra API key command group
// type:    cli
// job:     defines the parent "apikey" cobra command grouping API key management subcommands.
// limits:  holds no subcommand logic; subcommands register themselves (-> apikey/add.go).
package apikey

import (
	"github.com/spf13/cobra"
)

// Cmd is the apikey subcommand
var Cmd = &cobra.Command{
	Use:   "apikey",
	Short: "API key management commands",
	Long:  "Commands for managing API keys in the CGL system.",
}
