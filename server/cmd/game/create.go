package game

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new game",
	Long:  "Create a new game with the specified name.",
	Args:  cobra.ExactArgs(1),
	Run:   runCreate,
}

func init() {
	Cmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	var resp obj.Game
	if err := client.ApiPost("games/new", routes.CreateGameRequest{Name: name}, &resp); err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	fmt.Printf("Created game: %s\n", resp.ID)
	fmt.Printf("\nTo edit: go run . game get %s > game.yaml\n", resp.ID)
}
