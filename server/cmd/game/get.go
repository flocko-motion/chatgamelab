package game

import (
	"cgl/api/client"
	"cgl/functional"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <game-id> [output-file]",
	Short: "Get a game as YAML",
	Long:  "Fetch a game by ID and output it as YAML. Optionally write to a file.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runGet,
}

func init() {
	Cmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var yamlContent string
	functional.Require(client.ApiGetRaw(fmt.Sprintf("games/%s/yaml", gameID), &yamlContent), "failed to get game")
	fmt.Print(yamlContent)

	if len(args) > 1 {
		outputFile := args[1]
		functional.Require(os.WriteFile(outputFile, []byte(yamlContent), 0644), "failed to write to file")
	}
}
