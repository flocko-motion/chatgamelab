package game

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var gameInfoCmd = &cobra.Command{
	Use:   "info <id>",
	Short: "Show detailed info about a game",
	Long:  "Fetch and display all available details for a specific game.",
	Args:  cobra.ExactArgs(1),
	Run:   runGameInfo,
}

func init() {
	Cmd.AddCommand(gameInfoCmd)
}

func runGameInfo(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var game obj.Game
	if err := client.ApiGet("games/"+gameID, &game); err != nil {
		log.Fatalf("Failed to fetch game: %v", err)
	}

	fmt.Println("=== Game Info ===")
	fmt.Printf("ID:          %s\n", game.ID)
	fmt.Printf("Name:        %s\n", game.Name)
	fmt.Printf("Description: %s\n", functional.MaybeToString(game.Description, "n/a"))
	fmt.Printf("Public:      %v\n", game.Public)

	// Meta info
	fmt.Println("\n=== Metadata ===")
	fmt.Printf("Created By:  %s\n", game.Meta.CreatedBy.UUID)
	fmt.Printf("Created At:  %s\n", game.Meta.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified By: %s\n", game.Meta.ModifiedBy.UUID)
	fmt.Printf("Modified At: %s\n", game.Meta.ModifiedAt.Format("2006-01-02 15:04:05"))

	// Game content
	fmt.Println("\n=== Game Content ===")
	fmt.Printf("Scenario: %s\n", game.SystemMessageScenario)
	fmt.Printf("\nGame Start: %s\n", game.SystemMessageGameStart)
	fmt.Printf("\nImage Style: %s\n", game.ImageStyle)
	fmt.Printf("\nStatus Fields: %s\n", game.StatusFields)
	fmt.Printf("\nCustom CSS: %s\n", functional.MaybeToString(game.CSS, "n/a"))
	// Quick start content
	fmt.Println("\n=== Quick Start ===")
	fmt.Printf("First Message:%s\n", functional.MaybeToString(game.FirstMessage, "n/a"))
	fmt.Printf("First Status: %s\n", functional.MaybeToString(game.FirstStatus, "n/a"))

	// Sharing
	fmt.Println("\n=== Sharing ===")
	fmt.Printf("Public Sponsored API Key: %s\n", functional.MaybeToString(game.PublicSponsoredApiKeyID, "n/a"))
	fmt.Printf("Private Share Hash: %s\n", functional.MaybeToString(game.PrivateShareHash, "n/a"))
	fmt.Printf("Private Sponsored API Key: %s\n", functional.MaybeToString(game.PrivateSponsoredApiKeyID, "n/a"))

	// Tags
	fmt.Println("\n=== Tags ===")
	for _, tag := range game.Tags {
		fmt.Printf("- %s\n", tag.Tag)
	}
}
