package apikey

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var defaultCmd = &cobra.Command{
	Use:   "default [share-id]",
	Short: "Get or set the default API key share",
	Long:  `Get or set the default API key share for session creation.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runDefault,
}

func init() {
	Cmd.AddCommand(defaultCmd)
}

func runDefault(cmd *cobra.Command, args []string) {
	// Get current user info
	var user obj.User
	if err := client.ApiGet("users/me", &user); err != nil {
		log.Fatalf("Failed to get user info: %v", err)
	}

	if len(args) == 0 {
		// Show current default - just find the one marked as default in the list
		var keys []obj.ApiKeyShare
		if err := client.ApiGet("apikeys", &keys); err != nil {
			log.Fatalf("Failed to fetch API keys: %v", err)
		}

		for _, k := range keys {
			if k.IsUserDefault && k.ApiKey != nil {
				fmt.Printf("Default API key share:\n")
				fmt.Printf("  Share ID: %s\n", k.ID)
				fmt.Printf("  Key ID:   %s\n", k.ApiKey.ID)
				fmt.Printf("  Platform: %s\n", k.ApiKey.Platform)
				fmt.Printf("  Name:     %s\n", k.ApiKey.Name)
				fmt.Printf("  Owner:    %s\n", k.ApiKey.UserName)
				return
			}
		}

		fmt.Println("No default API key share set.")
		fmt.Println("Use 'apikey default <share-id>' to set one.")
		fmt.Println("Use 'apikey list' to see available shares.")
		return
	}

	// Set or clear default
	arg := args[0]

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	shareID, err := uuid.Parse(arg)
	if err != nil {
		log.Fatalf("Invalid share ID: %v", err)
	}

	req := endpoints.UserUpdateRequest{
		Name:                 user.Name,
		Email:                email,
		DefaultApiKeyShareID: &shareID,
	}
	if err := client.ApiPost("users/"+user.ID.String(), req, nil); err != nil {
		log.Fatalf("Failed to set default API key: %v", err)
	}

	fmt.Printf("Default API key share set to: %s\n", shareID)
}
