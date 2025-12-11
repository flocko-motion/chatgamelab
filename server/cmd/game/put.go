package game

import (
	"cgl/api/client"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put <game-id>",
	Short: "Update a game from YAML",
	Long:  "Update a game by ID from YAML input (stdin or file).",
	Args:  cobra.ExactArgs(1),
	Run:   runPut,
}

var inputFile string

func init() {
	putCmd.Flags().StringVarP(&inputFile, "file", "f", "", "YAML file to read (default: stdin)")
	Cmd.AddCommand(putCmd)
}

func runPut(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var yamlContent []byte
	var err error

	if inputFile != "" {
		yamlContent, err = os.ReadFile(inputFile)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
	} else {
		yamlContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read stdin: %v", err)
		}
	}

	if err := client.ApiPutRaw(fmt.Sprintf("games/%s/yaml", gameID), string(yamlContent)); err != nil {
		log.Fatalf("Failed to update game: %v", err)
	}

	fmt.Println("Game updated successfully")
}
