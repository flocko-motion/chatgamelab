package user

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var userInfoCmd = &cobra.Command{
	Use:   "info <uuid|me>",
	Short: "Get detailed information about a user",
	Long:  "Fetch and display detailed information about a user by UUID or 'me' for current user.",
	Args:  cobra.ExactArgs(1),
	Run:   runUserInfo,
}

func init() {
	Cmd.AddCommand(userInfoCmd)
}

func runUserInfo(cmd *cobra.Command, args []string) {
	var user obj.User
	var err error

	if args[0] == "me" {
		// Get current user info
		err = client.ApiGet("users/me", &user)
	} else {
		// Parse as UUID
		userID, parseErr := uuid.Parse(args[0])
		if parseErr != nil {
			log.Fatalf("Invalid UUID: %v", parseErr)
		}
		err = client.ApiGet(fmt.Sprintf("users/%s", userID), &user)
	}

	if err != nil {
		log.Fatalf("Failed to fetch user: %v", err)
	}

	// Display user information
	fmt.Println("User Information:")
	fmt.Printf("  ID:       %s\n", user.ID)
	fmt.Printf("  Name:     %s\n", user.Name)
	fmt.Printf("  Email:    %s\n", functional.MaybeToString(user.Email, "n/a"))
	fmt.Printf("  Auth0 ID: %s\n", functional.MaybeToString(user.Auth0Id, "n/a"))

	if user.Role != nil {
		fmt.Printf("\nRole:\n")
		fmt.Printf("  Role:        %s\n", user.Role.Role)
		if user.Role.Institution != nil {
			fmt.Printf("  Institution: %s (ID: %s)\n", user.Role.Institution.Name, user.Role.Institution.ID)
		}
		if user.Role.Workshop != nil {
			fmt.Printf("  Workshop:    %s (ID: %s)\n", user.Role.Workshop.Name, user.Role.Workshop.ID)
		}
	} else {
		fmt.Printf("\nRole: none\n")
	}

	if len(user.ApiKeys) > 0 {
		fmt.Printf("\nAPI Keys (%d):\n", len(user.ApiKeys))
		for _, share := range user.ApiKeys {
			if share.ApiKey != nil {
				fmt.Printf("  - %s (%s)\n", share.ApiKey.Name, share.ApiKey.Platform)
			} else {
				fmt.Printf("  - Share ID: %s\n", share.ID)
			}
		}
	} else {
		fmt.Printf("\nAPI Keys: none\n")
	}

	if user.Meta.CreatedAt != nil {
		fmt.Printf("\nCreated: %s\n", user.Meta.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if user.Meta.ModifiedAt != nil {
		fmt.Printf("Modified: %s\n", user.Meta.ModifiedAt.Format("2006-01-02 15:04:05"))
	}
}
