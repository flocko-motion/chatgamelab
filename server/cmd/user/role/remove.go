package role

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <user-id>",
	Short: "Remove a user's role",
	Long:  "Remove a user's role, making them a regular user without special permissions.",
	Args:  cobra.ExactArgs(1),
	Run:   runRemove,
}

func init() {
	Cmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) {
	userID, err := uuid.Parse(args[0])
	if err != nil {
		log.Fatalf("Invalid user ID: %v", err)
	}

	// Make API request to remove role
	if err := client.ApiDelete(fmt.Sprintf("users/%s/role", userID)); err != nil {
		log.Fatalf("Failed to remove user role: %v", err)
	}

	fmt.Printf("✓ Role removed successfully\n")
	fmt.Printf("  User ID: %s\n", userID)
}
