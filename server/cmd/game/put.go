package game

import (
	"cgl/api/client"
	"cgl/functional"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put <yaml-file> [game-id]",
	Short: "Create or update a game from YAML",
	Long:  "Create a new game or update an existing game from YAML file. If game-id is provided, updates that game. Otherwise creates a new game. Use --stdin to read from stdin instead of file.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runPut,
}

var useStdin bool

func init() {
	putCmd.Flags().BoolVar(&useStdin, "stdin", false, "Read YAML from stdin instead of file")
	Cmd.AddCommand(putCmd)
}

func runPut(cmd *cobra.Command, args []string) {
	var yamlContent []byte

	if useStdin {
		yamlContent = functional.MustReturn(io.ReadAll(os.Stdin))
	} else {
		yamlContent = functional.MustReturn(os.ReadFile(args[0]))
	}

	if len(args) > 1 {
		// Update existing game
		gameID := args[1]
		functional.Must(client.ApiPutRaw(fmt.Sprintf("games/%s/yaml", gameID), string(yamlContent)), "failed to update game")
		fmt.Println("Game updated successfully")
		printGameInfo(gameID)
	} else {
		// Create new game
		var resp struct {
			ID string `json:"id"`
		}
		functional.Must(client.ApiPostRaw("games/new", string(yamlContent), &resp), "failed to create game")
		fmt.Println("Game created successfully")
		printGameInfo(resp.ID)
	}
}
