package apikey

import (
	"github.com/spf13/cobra"
)

// Cmd is the apikey subcommand
var Cmd = &cobra.Command{
	Use:   "apikey",
	Short: "API key management commands",
	Long:  "Commands for managing API keys in the CGL system.",
}
