// package: game / cobra game command group
// type:    cli
// job:     defines the parent "game" cobra command grouping game management subcommands.
// limits:  holds no subcommand logic; subcommands register themselves (-> game/create.go).
package game

import (
	"github.com/spf13/cobra"
)

// Cmd is the game subcommand
var Cmd = &cobra.Command{
	Use:   "game",
	Short: "Game management commands",
	Long:  "Commands for managing games in the CGL system.",
}
