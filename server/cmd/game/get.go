package game

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <game-id>",
	Short: "Get a game as YAML",
	Long:  "Fetch a game by ID and output it as YAML. Redirect to a file to edit.",
	Args:  cobra.ExactArgs(1),
	Run:   runGet,
}

func init() {
	Cmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var yamlContent string
	if err := client.ApiGetRaw(fmt.Sprintf("games/%s/yaml", gameID), &yamlContent); err != nil {
		log.Fatalf("Failed to get game: %v", err)
	}

	fmt.Print(yamlContent)
}
