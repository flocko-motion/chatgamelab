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
