// package: workshop / workshop management CLI commands
// type:    cli
// job:     define the root "workshop" cobra command that groups the workshop subcommands
// limits:  wiring only; individual subcommands implement the actual behaviour
package workshop

import (
	"github.com/spf13/cobra"
)

// Cmd is the workshop subcommand
var Cmd = &cobra.Command{
	Use:   "workshop",
	Short: "Workshop management commands",
	Long:  "Commands for managing workshops in the CGL system.",
}
