// package: lang / language translation helper commands
// type:    cli
// job:     define the root "lang" cobra command and shared file-existence helper
// limits:  wiring only; subcommands implement listing, translating and verifying
package lang

import (
	"os"

	"github.com/spf13/cobra"
)

// Cmd is the lang subcommand
var Cmd = &cobra.Command{
	Use:   "lang",
	Short: "Language translation helper commands",
	Long:  "Commands for translating language files using AI.",
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
