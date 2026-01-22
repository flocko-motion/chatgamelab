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
