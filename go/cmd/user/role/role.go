// package: role / user role management CLI commands
// type:    cli
// job:     define the root "role" cobra command that groups the role subcommands
// limits:  wiring only; individual subcommands implement the actual behaviour
package role

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "role",
	Short: "Manage user roles",
	Long:  "Commands for managing user roles (admin, head, staff, participant).",
}
