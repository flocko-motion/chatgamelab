// package: apikey / cobra share command group
// type:    cli
// job:     defines the parent "apikey share" cobra command grouping share subcommands.
// limits:  holds no share logic; add/delete subcommands register under it (-> apikey/share_add.go).
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
