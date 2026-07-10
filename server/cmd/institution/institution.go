// package: institution / institution management CLI commands
// type:    cli
// job:     define the root "institution" cobra command that groups the institution subcommands
// limits:  wiring only; individual subcommands implement the actual behaviour
package institution

import (
	"github.com/spf13/cobra"
)

// Cmd is the institution subcommand
var Cmd = &cobra.Command{
	Use:   "institution",
	Short: "Institution management commands",
	Long:  "Commands for managing institutions in the CGL system.",
}
