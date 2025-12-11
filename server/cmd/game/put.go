package game

import (
	"cgl/api/client"
	"cgl/functional"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put <game-id> <yaml-file>",
	Short: "Update a game from YAML",
	Long:  "Update a game by ID from YAML file. Use --stdin to read from stdin instead.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runPut,
}

var useStdin bool

func init() {
	putCmd.Flags().BoolVar(&useStdin, "stdin", false, "Read YAML from stdin instead of file")
	Cmd.AddCommand(putCmd)
}

func runPut(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var yamlContent []byte

	if useStdin {
		yamlContent = functional.MustReturn(io.ReadAll(os.Stdin))
	} else if len(args) > 1 {
		yamlContent = functional.MustReturn(os.ReadFile(args[1]))
	} else {
		log.Fatalf("Either provide a YAML file as second argument or use --stdin")
	}

	functional.Must(client.ApiPutRaw(fmt.Sprintf("games/%s/yaml", gameID), string(yamlContent)), "failed to update game")

	fmt.Println("Game updated successfully")
	printGameInfo(gameID)
}
