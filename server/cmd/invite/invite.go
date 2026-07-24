// package: invite / invite management CLI commands
// type:    cli
// job:     define the root "invite" cobra command that groups the invite subcommands
// limits:  wiring only; individual subcommands implement the actual behaviour
package invite

import (
	"github.com/spf13/cobra"
)

// Cmd is the invite subcommand
var Cmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite management commands",
	Long:  "Commands for managing user role invitations in the CGL system.",
}
