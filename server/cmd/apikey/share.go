package apikey

import (
	"github.com/spf13/cobra"
)

// shareCmd is the parent command for share subcommands
var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Manage API key shares",
	Long:  "Commands for managing API key shares (list, add, delete).",
}

func init() {
	Cmd.AddCommand(shareCmd)
}
