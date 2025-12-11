package game

import (
	"cgl/api/client"
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

	type CreateRequest struct {
		Name string `json:"name"`
	}
	type CreateResponse struct {
		ID string `json:"id"`
	}

	var resp CreateResponse
	if err := client.ApiPost("games/new", CreateRequest{Name: name}, &resp); err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	fmt.Printf("Created game: %s\n", resp.ID)
	fmt.Printf("\nTo edit: go run . game get %s > game.yaml\n", resp.ID)
}
