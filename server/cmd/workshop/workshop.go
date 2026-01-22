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
