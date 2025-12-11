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
