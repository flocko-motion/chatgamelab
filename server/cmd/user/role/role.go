package role

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "role",
	Short: "Manage user roles",
	Long:  "Commands for managing user roles (admin, head, staff, participant).",
}
