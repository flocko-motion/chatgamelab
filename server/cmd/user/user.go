// package: user / user management CLI commands
// type:    cli
// job:     define the root "user" cobra command and register the role subcommand group
// limits:  wiring only; individual subcommands implement the actual behaviour
package user

import (
	"cgl/cmd/user/role"

	"github.com/spf13/cobra"
)

// Cmd is the user subcommand
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  "Commands for managing users in the CGL system.",
}

func init() {
	Cmd.AddCommand(role.Cmd)
}
